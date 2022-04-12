package tcp

import (
	"fmt"
	"net"
)

func DialAndGetReply(host string, port uint16) ([]byte, error) {
	address, err := ResolveTCPAddress(host, port)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve tcp address: %s", err)
	}

	conn, err := net.DialTCP("tcp", nil, address)
	if err != nil {
		return nil, fmt.Errorf("cannot dial provided address: %s", err)
	}
	defer conn.Close()

	return readBytes(conn)
}

func readBytes(conn *net.TCPConn) ([]byte, error) {
	buff := make([]byte, 1024)
	n, err := conn.Read(buff)
	if err != nil {
		return nil, fmt.Errorf("cannot read from the connection: %s", err)
	}

	if err := conn.Close(); err != nil {
		return nil, fmt.Errorf("cannot close the connection: %s", err)
	}

	return buff[:n], nil
}
