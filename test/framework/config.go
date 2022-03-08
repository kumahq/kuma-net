package framework

import (
	"net"

	"github.com/kumahq/kuma-net/test/framework/tcp"
)

// ConfigTCPServer
// TODO (bartsmykla): write description
type ConfigTCPServer struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

// Address
// TODO (bartsmykla): write description
// TODO (bartsmykla): write tests maybe(?)
func (c *ConfigTCPServer) Address() (*net.TCPAddr, error) {
	return tcp.ResolveAddress(c.Host, c.Port)
}

// Config
// TODO (bartsmykla): write description
type Config struct {
	TCPServer            *ConfigTCPServer `json:"tcpServer"`
	ExcludeInboundPorts  []uint16         `json:"excludeInboundPorts"`
	ExcludeOutboundPorts []uint16         `json:"excludeOutboundPorts"`
}

// ConfigRedirectTCPTrafficDefault
// TODO (bartsmykla): write description
type ConfigRedirectTCPTrafficDefault struct {
	// TODO (bartsmykla): write description
	AmountOfPortsToTest uint             `json:"amountOfPortsToTest"`
	TCPServer           *ConfigTCPServer `json:"tcpServer"`
}
