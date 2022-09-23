//go:build linux

package ebpf

import (
	"fmt"
	"net"
	"os"
	"syscall"

	ciliumebpf "github.com/cilium/ebpf"
	"github.com/cilium/ebpf/rlimit"

	"github.com/kumahq/kuma-net/transparent-proxy/config"
)

var programs = []*Program{
	{
		Name:  "mb_connect",
		Flags: cgroupFlags,
	},
	{
		Name:  "mb_sockops",
		Flags: cgroupFlags,
	},
	{
		Name:  "mb_get_sockopts",
		Flags: cgroupFlags,
	},
	{
		Name:  "mb_sendmsg",
		Flags: cgroupFlags,
	},
	{
		Name:  "mb_recvmsg",
		Flags: cgroupFlags,
	},
	{
		Name:  "mb_redir",
		Flags: flags(nil),
	},
	{
		Name: "mb_tc",
		Flags: func(
			cfg config.Config,
			cgroup string,
			bpffs string,
		) ([]string, error) {
			var err error
			var iface string

			if cfg.Ebpf.TCAttachIface != "" && ifaceIsUp(cfg.Ebpf.TCAttachIface) {
				iface = cfg.Ebpf.TCAttachIface
			} else if iface, err = getNonLoopbackRunningInterface(); err != nil {
				return nil, fmt.Errorf("getting non-loopback interface failed: %v", err)
			}

			return flags(map[string]string{
				"--iface": iface,
			})(cfg, cgroup, bpffs)
		},
	},
}

func getNonLoopbackRunningInterface() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to list network interfaces: %v", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			return iface.Name, nil
		}
	}

	return "", fmt.Errorf("cannot find other than loopback interface")
}

func ifaceIsUp(ifName string) bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	for _, iface := range interfaces {
		if iface.Name == ifName && iface.Flags&net.FlagUp != 0 {
			return true
		}
	}

	return false
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

	if err := rlimit.RemoveMemlock(); err != nil {
		return "", fmt.Errorf("removing memory lock failed with error: %s", err)
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
