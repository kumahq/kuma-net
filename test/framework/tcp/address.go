package tcp

import (
	"fmt"
	"net"
	"strconv"
)

func ResolveTCPAddress(host string, port uint16) (*net.TCPAddr, error) {
	if host == "" {
		return nil, fmt.Errorf("host cannot be empty")
	}

	if port == 0 {
		return nil, fmt.Errorf("port cannot be 0")
	}

	portString := strconv.Itoa(int(port))
	hostPort := net.JoinHostPort(host, portString)

	addr, err := net.ResolveTCPAddr("tcp", hostPort)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot resolve the tcp address (%s:%d): %s", host, port, err,
		)
	}

	return addr, nil
}
