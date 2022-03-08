package tcp

import (
	"fmt"
	"net"
)

type ClientConnectionHandler func(conn *net.TCPConn) (interface{}, error)

func ReadBytes(conn *net.TCPConn) (interface{}, error) {
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

func ReadTCPAddr(conn *net.TCPConn) (interface{}, error) {
	bs, err := ReadBytes(conn)
	if err != nil {
		return nil, fmt.Errorf("reading bytes from the connection failed: %s", err)
	}

	return net.ResolveTCPAddr(Network, string(bs.([]byte)))
}

type Client struct {
	host              string
	port              uint16
	address           *net.TCPAddr
	connection        *net.TCPConn
	connectionHandler ConnHandler
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) WithHost(host string) *Client {
	c.host = host

	return c
}

func (c *Client) WithPort(port uint16) *Client {
	c.port = port

	return c
}

func (c *Client) WithAddress(address *net.TCPAddr) *Client {
	c.address = address

	return c
}

func (c *Client) Address() *net.TCPAddr {
	return c.address
}

func (c *Client) Dial() (*net.TCPConn, error) {
	if c.address == nil {
		address, err := ResolveAddress(c.host, c.port)
		if err != nil {
			return nil, fmt.Errorf("address resolving failed: %s", err)
		}
		c.address = address
	}

	connection, err := net.DialTCP(Network, nil, c.address)
	if err != nil {
		return nil, fmt.Errorf(
			"error when dialling a TCP server (%s): %s",
			c.address,
			err,
		)
	}
	c.connection = connection

	return connection, nil
}

func (c *Client) DialAndGetReply(
	handleConnection ClientConnectionHandler,
) (interface{}, error) {
	conn, err := c.Dial()
	if err != nil {
		return "", fmt.Errorf("cannot dial the server: %s", err)
	}

	reply, err := handleConnection(conn)
	if err != nil {
		return nil, fmt.Errorf("connection handling failed: %s", err)
	}

	return reply, nil
}

func (c *Client) Close() error {
	if err := c.connection.Close(); err != nil {
		return fmt.Errorf("connection closing failed: %s", err)
	}

	return nil
}
