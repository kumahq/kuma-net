package tcp

import (
	"fmt"
	"net"
	"strconv"
)

func ResolveAddress(host string, port uint16) (*net.TCPAddr, error) {
	if host == "" {
		return nil, fmt.Errorf("host cannot be empty")
	}

	if port == 0 {
		return nil, fmt.Errorf("port cannot be 0")
	}

	portString := strconv.Itoa(int(port))
	hostPort := net.JoinHostPort(host, portString)

	addr, err := net.ResolveTCPAddr(Network, hostPort)
	if err != nil {
		// TODO (bartsmykla): think of better error message
		return nil, fmt.Errorf(
			"ResolveTCPAddr (%s:%d) failed: %s",
			host,
			port,
			err,
		)
	}

	return addr, nil
}
