package framework

import (
	"net"
	"strconv"

	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/test/framework/tcp"
)

type ConfigTCPServer struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

func (c *ConfigTCPServer) Address() *net.TCPAddr {
	port := strconv.Itoa(int(c.Port))
	hostPort := net.JoinHostPort(c.Host, port)

	address, err := net.ResolveTCPAddr(tcp.Network, hostPort)
	Expect(err).To(BeNil())

	return address
}

func DefaultConfigTCPServer() *ConfigTCPServer {
	return &ConfigTCPServer{
		Host: "localhost",
		Port: 7878,
	}
}

type Config struct {
	TCPServer            *ConfigTCPServer
	ExcludeInboundPorts  []uint16
	ExcludeOutboundPorts []uint16
}

func DefaultConfig() *Config {
	return &Config{
		TCPServer: DefaultConfigTCPServer(),
	}
}

type ConfigRedirectTCPTrafficDefault struct {
	AmountOfPortsToTest uint `json:"amountOfPortsToTest"`
	TCPServer           *ConfigTCPServer
}

func DefaultConfigRedirectTCPTrafficDefault() *ConfigRedirectTCPTrafficDefault {
	return &ConfigRedirectTCPTrafficDefault{
		AmountOfPortsToTest: 5,
		TCPServer:           DefaultConfigTCPServer(),
	}
}
