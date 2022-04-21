package ip

import (
	"math/rand"
	"net"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

var reserved = []*net.IPNet{
	// Current network
	{
		IP:   net.IPv4(0, 0, 0, 0),
		Mask: net.CIDRMask(8, 32),
	},
	// Used for loopback addresses to the local host
	{
		IP:   net.IPv4(127, 0, 0, 0),
		Mask: net.CIDRMask(8, 32),
	},
	// Used for link-local addresses between two hosts on a single link
	// when no IP address is otherwise specified, such as would have normally
	// been retrieved from a DHCP server
	{
		IP:   net.IPv4(169, 254, 0, 0),
		Mask: net.CIDRMask(16, 32),
	},
	// Reserved. Formerly used for IPv6 to IPv4 relay (included IPv6 address
	// block 2002::/16)
	{
		IP:   net.IPv4(192, 88, 99, 0),
		Mask: net.CIDRMask(16, 32),
	},
	// In use for IP multicast (Former Class D network)
	{
		IP:   net.IPv4(224, 0, 0, 0),
		Mask: net.CIDRMask(4, 32),
	},
	// Reserved for future use (Former Class E network)
	{
		IP:   net.IPv4(240, 0, 0, 0),
		Mask: net.CIDRMask(4, 32),
	},
	// Reserved for the "limited broadcast" destination address
	{
		IP:   net.IPv4(255, 255, 255, 255),
		Mask: net.CIDRMask(32, 32),
	},
}

// GenRandomIPv4 will return random, non-reserved IPv4 address
func GenRandomIPv4() net.IP {
	size := 4
	ipBytes := make([]byte, size)
	for i := 0; i < size; i++ {
		ipBytes[i] = byte(r.Intn(256))
	}

	ip := net.IP(ipBytes)

	for _, ipNet := range reserved {
		if ipNet.Contains(ip) {
			return GenRandomIPv4()
		}
	}

	return ip.To4()
}
