package ebpf

import (
	"fmt"
	"os"

	ciliumebpf "github.com/cilium/ebpf"

	"github.com/kumahq/kuma-net/transparent-proxy/config"
)

var programs = []*Program{
	{
		PinName:          "connect",
		MakeLoadTarget:   "load-connect",
		MakeAttachTarget: "attach-connect",
	},
	{
		PinName:          "sockops",
		MakeLoadTarget:   "load-sockops",
		MakeAttachTarget: "attach-sockops",
	},
	{
		PinName:          "get_sockopts",
		MakeLoadTarget:   "load-getsock",
		MakeAttachTarget: "attach-getsock",
	},
	{
		PinName:          "redir",
		MakeLoadTarget:   "load-redir",
		MakeAttachTarget: "attach-redir",
	},
	{
		PinName:          "sendmsg",
		MakeLoadTarget:   "load-sendmsg",
		MakeAttachTarget: "attach-sendmsg",
	},
	{
		PinName:          "recvmsg",
		MakeLoadTarget:   "load-recvmsg",
		MakeAttachTarget: "attach-recvmsg",
	},
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

	instanceIP := os.Getenv(cfg.Ebpf.InstanceIPEnvVarName)

	ip, err := IpStrToUint32(instanceIP)
	if err != nil {
		return "", err
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
			instanceIP,
			err,
		)
	}

	_, _ = cfg.RuntimeStdout.Write([]byte(fmt.Sprintf("local_pod_ips map was updated with current instance IP: %s\n\n", instanceIP)))

	return "", nil
}
