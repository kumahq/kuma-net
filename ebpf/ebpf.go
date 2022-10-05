//go:build linux

package ebpf

import (
	"fmt"
	"net"
	"strings"
	"unsafe"

	"github.com/kumahq/kuma-net/transparent-proxy/config"
)

const (
	// MaxItemLen is the maximal amount of items like ports or IP ranges to include
	// or/and exclude. It's currently hardcoded to 10 as merbridge during creation
	// of this map is assigning hardcoded 244 bytes for map values:
	//
	//  Cidr:        8 bytes
	//    Cidr.Net:  4 bytes
	//    Cidr.Mask: 1 byte
	//    pad:       3 bytes
	//
	//  PodConfig:                                  244 bytes
	//    PodConfig.StatusPort:                       2 bytes
	//    pad:                                        2 bytes
	//    PodConfig.ExcludeOutRanges (10x Cidr):     80 bytes
	//    PodConfig.IncludeOutRanges (10x Cidr):     80 bytes
	//    PodConfig.IncludeInPorts   (10x 2 bytes):  20 bytes
	//    PodConfig.IncludeOutPorts  (10x 2 bytes):  20 bytes
	//    PodConfig.ExcludeInPorts   (10x 2 bytes):  20 bytes
	//    PodConfig.ExcludeOutPorts  (10x 2 bytes):  20 bytes
	//
	// todo (bartsmykla): merbridge flagged this constant to be changed, so if
	//                    it will be changed, we have to update it
	MaxItemLen = 10
	// MapRelativePathLocalPodIPs is a path where the local_pod_ips map
	// is pinned, it's hardcoded as "{BPFFS_path}/tc/globals/local_pod_ips" because
	// merbridge is hard-coding it as well, and we don't want to allot to change it
	// by mistake
	MapRelativePathLocalPodIPs   = "/local_pod_ips"
	MapRelativePathNetNSPodIPs   = "/netns_pod_ips"
	MapRelativePathCookieOrigDst = "/cookie_orig_dst"
	MapRelativePathProcessIP     = "/process_ip"
	MapRelativePathPairOrigDst   = "/pair_orig_dst"
	MapRelativePathSockPairMap   = "/sock_pair_map"
)

var programs = []*Program{
	{
		Name:  "mb_connect",
		Flags: cgroupFlags,
		Cleanup: cleanPathsRelativeToBPFFS(
			"connect", // directory
			MapRelativePathCookieOrigDst,
			MapRelativePathNetNSPodIPs,
			MapRelativePathLocalPodIPs,
			MapRelativePathProcessIP,
		),
	},
	{
		Name:  "mb_sockops",
		Flags: cgroupFlags,
		Cleanup: cleanPathsRelativeToBPFFS(
			"sockops",
			MapRelativePathCookieOrigDst,
			MapRelativePathProcessIP,
			MapRelativePathPairOrigDst,
			MapRelativePathSockPairMap,
		),
	},
	{
		Name:  "mb_get_sockopts",
		Flags: cgroupFlags,
		Cleanup: cleanPathsRelativeToBPFFS(
			"get_sockopts",
			MapRelativePathPairOrigDst,
		),
	},
	{
		Name:  "mb_sendmsg",
		Flags: cgroupFlags,
		Cleanup: cleanPathsRelativeToBPFFS(
			"sendmsg",
			MapRelativePathCookieOrigDst,
		),
	},
	{
		Name:  "mb_recvmsg",
		Flags: cgroupFlags,
		Cleanup: cleanPathsRelativeToBPFFS(
			"recvmsg",
			MapRelativePathCookieOrigDst,
		),
	},
	{
		Name:  "mb_redir",
		Flags: flags(nil),
		Cleanup: cleanPathsRelativeToBPFFS(
			"redir",
			MapRelativePathSockPairMap,
		),
	},
	{
		Name:  "mb_netns_cleanup",
		Flags: flags(nil),
		Cleanup: cleanPathsRelativeToBPFFS(
			"netns_cleanup_prog",
			"netns_cleanup_link",
			MapRelativePathNetNSPodIPs,
			MapRelativePathLocalPodIPs,
		),
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
		Cleanup: cleanPathsRelativeToBPFFS(
			MapRelativePathLocalPodIPs,
			MapRelativePathPairOrigDst,
		),
	},
}

type Cidr struct {
	Net  uint32 // network order
	Mask uint8
	_    [3]uint8 // pad
}

type PodConfig struct {
	StatusPort       uint16
	_                uint16 // pad
	ExcludeOutRanges [MaxItemLen]Cidr
	IncludeOutRanges [MaxItemLen]Cidr
	IncludeInPorts   [MaxItemLen]uint16
	IncludeOutPorts  [MaxItemLen]uint16
	ExcludeInPorts   [MaxItemLen]uint16
	ExcludeOutPorts  [MaxItemLen]uint16
}

func ipStrToPtr(ipstr string) (unsafe.Pointer, error) {
	var ip net.IP

	if ip = net.ParseIP(ipstr); ip == nil {
		return nil, fmt.Errorf("error parse ip: %s", ipstr)
	} else if ip.To4() != nil {
		// ipv4, we need to clear the bytes
		for i := 0; i < 12; i++ {
			ip[i] = 0
		}
	}

	return unsafe.Pointer(&ip[0]), nil
}

func LoadAndAttachEbpfPrograms(programs []*Program, cfg config.Config) error {
	var errs []string

	cgroup, err := getCgroupPath(cfg)
	if err != nil {
		return fmt.Errorf("getting cgroup failed with error: %s", err)
	}

	bpffs, err := getBpffsPath(cfg)
	if err != nil {
		return fmt.Errorf("getting bpffs failed with error: %s", err)
	}

	for _, p := range programs {
		if err := p.LoadAndAttach(cfg, cgroup, bpffs); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("loading and attaching ebpf programs failed:\n\t%s",
			strings.Join(errs, "\n\t"))
	}

	return nil
}
