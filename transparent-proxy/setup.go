package transparent_proxy

import (
	"github.com/kumahq/kuma-net/ebpf"
	"github.com/kumahq/kuma-net/iptables"
	"github.com/kumahq/kuma-net/transparent-proxy/config"
)

func Setup(cfg config.Config) (string, error) {
	if cfg.Ebpf.Enabled {
		return ebpf.Setup(cfg)
	}

	return iptables.Setup(cfg)
}

func Cleanup(cfg config.Config) (string, error) {
	if cfg.Ebpf.Enabled {
		return ebpf.Cleanup(cfg)
	}

	panic("currently not supported")
}
