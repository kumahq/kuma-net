//go:build linux

package ebpf

import (
	"fmt"
	"net"
	"os"
	"syscall"

	ciliumebpf "github.com/cilium/ebpf"

	"github.com/kumahq/kuma-net/transparent-proxy/config"
)

const CgroupPath = "/sys/fs/cgroup"
const BpfFSPath = "/run/kuma/bpf"

var programs = []*Program{
	{
		Name:  "mb_connect",
		Flags: cgroupFlags(),
	},
	{
		Name:  "mb_sockops",
		Flags: cgroupFlags(),
	},
	{
		Name:  "mb_get_sockopts",
		Flags: cgroupFlags(),
	},
	{
		Name:  "mb_sendmsg",
		Flags: cgroupFlags(),
	},
	{
		Name:  "mb_recvmsg",
		Flags: cgroupFlags(),
	},
	{
		Name:  "mb_redir",
		Flags: flags(nil),
	},
	{
		Name: "mb_tc",
		Flags: func(verbose bool) ([]string, error) {
			if iface, err := getNonLoopbackInterface(); err != nil {
				return nil, fmt.Errorf("getting non-loopback interface failed: %v", err)
			} else {
				return flags(map[string]string{
					"--iface": iface.Name,
				})(verbose)
			}
		},
	},
}

func flags(flags map[string]string) func(bool) ([]string, error) {
	f := map[string]string{
		"--bpffs": BpfFSPath,
	}

	return func(verbose bool) ([]string, error) {
		if verbose {
			f["--verbose"] = ""
		}

		if flags == nil {
			return mapFlagsToSlice(f), nil
		}

		for k, v := range flags {
			f[k] = v
		}

		return mapFlagsToSlice(f), nil
	}
}

func cgroupFlags() func(bool) ([]string, error) {
	return flags(map[string]string{
		"--cgroup": CgroupPath,
	})
}

func mapFlagsToSlice(flags map[string]string) []string {
	var result []string

	for k, v := range flags {
		result = append(result, k)

		if v != "" {
			result = append(result, v)
		}
	}

	return result
}

// TODO (bartsmykla): currently we are assuming there is only one other than
//  loopback interface, and we are attaching eBPF programs only to it.
//  It's probably fine in the context of k8s and init containers,
//  but not for vms/universal
func getNonLoopbackInterface() (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %v", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 {
			return &iface, nil
		}
	}

	return nil, fmt.Errorf("cannot find other than loopback interface")
}

func GetFileInode(path string) (uint64, error) {
	f, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to get the inode of %s", path)
	}
	stat, ok := f.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("not syscall.Stat_t")
	}
	return stat.Ino, nil
}

func Setup(cfg config.Config) (string, error) {
	if os.Getuid() != 0 {
		return "", fmt.Errorf("root user in required for this process or container")
	}

	if err := InitBPFFSMaybe(cfg.Ebpf.BPFFSPath); err != nil {
		return "", fmt.Errorf("initializing BPF file system failed: %v", err)
	}

	if err := LoadAndAttachEbpfPrograms(programs, cfg); err != nil {
		return "", err
	}

	localPodIPsMap, err := ciliumebpf.LoadPinnedMap(
		cfg.Ebpf.BPFFSPath+LocalPodIPSPinnedMapPathRelativeToBPFFS,
		&ciliumebpf.LoadPinOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("loading pinned local_pod_ips map failed: %v", err)
	}

	netnsPodIPsMap, err := ciliumebpf.LoadPinnedMap(
		cfg.Ebpf.BPFFSPath+NetNSPodIPSPinnedMapPathRelativeToBPFFS,
		&ciliumebpf.LoadPinOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("loading pinned netns_pod_ips map failed: %v", err)
	}

	netnsInode, err := GetFileInode("/proc/self/ns/net")
	if err != nil {
		return "", fmt.Errorf("getting netns inode failed: %s", err)
	}

	ip, err := ipStrToPtr(cfg.Ebpf.InstanceIP)
	if err != nil {
		return "", err
	}

	if err := netnsPodIPsMap.Update(netnsInode, ip, ciliumebpf.UpdateAny); err != nil {
		return "", fmt.Errorf("updating netns_pod_ips map failed (ip: %v, nens: %v): %v", ip, netnsInode, err)
	}

	// exclude inbound ports

	excludeInboundPorts := [MaxItemLen]uint16{
		cfg.Redirect.Inbound.Port,
		cfg.Redirect.Inbound.PortIPv6,
		cfg.Redirect.Outbound.Port,
	}

	allowedAmountOfExcludeInPorts := MaxItemLen - len(excludeInboundPorts)

	if len(cfg.Redirect.Inbound.ExcludePorts) > allowedAmountOfExcludeInPorts {
		return "", fmt.Errorf(
			"maximal allowed amound of exclude inbound ports (%d) exceeded (%d): %+v",
			allowedAmountOfExcludeInPorts,
			len(cfg.Redirect.Inbound.ExcludePorts),
			cfg.Redirect.Inbound.ExcludePorts,
		)
	}

	// exclude outbound ports

	excludeOutPorts := [MaxItemLen]uint16{}

	if len(cfg.Redirect.Outbound.ExcludePorts) > MaxItemLen {
		return "", fmt.Errorf(
			"maximal allowed amound of exclude outbound ports (%d) exceeded (%d): %+v",
			MaxItemLen,
			len(cfg.Redirect.Outbound.ExcludePorts),
			cfg.Redirect.Outbound.ExcludePorts,
		)
	}

	if err := localPodIPsMap.Update(ip, &PodConfig{
		ExcludeInPorts:  excludeInboundPorts,
		ExcludeOutPorts: excludeOutPorts,
	}, ciliumebpf.UpdateAny); err != nil {
		return "", fmt.Errorf(
			"updating pinned local_pod_ips map with current instance IP (%s) failed: %v",
			cfg.Ebpf.InstanceIP,
			err,
		)
	}

	_, _ = cfg.RuntimeStdout.Write([]byte(fmt.Sprintf("local_pod_ips map was updated with current instance IP: %s\n\n", cfg.Ebpf.InstanceIP)))

	return "", nil
}
