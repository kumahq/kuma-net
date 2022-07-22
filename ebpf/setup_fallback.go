//go:build !linux

package ebpf

import (
	"fmt"

	"github.com/kumahq/kuma-net/transparent-proxy/config"
)

func Setup(config.Config) (string, error) {
	return "", fmt.Errorf("ebpf is currently supported only on linux")
}
