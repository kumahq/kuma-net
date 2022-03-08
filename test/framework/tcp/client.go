package tcp

import (
	"fmt"
	"net"
	"strconv"
)

type ClientConnectionHandler func(conn *net.TCPConn) ([]byte, error)

func ReadBytes(conn *net.TCPConn) ([]byte, error) {
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

type Client struct {
	host              string
	port              *uint16
	address           *net.TCPAddr
	connection        *net.TCPConn
	connectionHandler ConnectionHandler
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) WithHost(host string) *Client {
	c.host = host

	return c
}

func (c *Client) WithPort(port uint16) *Client {
	c.port = &port

	return c
}

func (c *Client) Address() *net.TCPAddr {
	return c.address
}

func (c *Client) Dial() (*net.TCPConn, error) {
	if c.port == nil {
		return nil, fmt.Errorf("missing port")
	}

	if c.host == "" {
		return nil, fmt.Errorf("missing host")
	}

	port := strconv.Itoa(int(*c.port))
	hostPort := net.JoinHostPort(c.host, port)

	address, err := net.ResolveTCPAddr(Network, hostPort)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot resolve TCP address from provided host and port (%s:%d): %s",
			c.host,
			*c.port,
			err,
		)
	}
	c.address = address

	connection, err := net.DialTCP(Network, nil, address)
	if err != nil {
		return nil, fmt.Errorf("error when dialling a TCP server (%s): %s", address, err)
	}
	c.connection = connection

	return connection, nil
}

func (c *Client) DialAndWaitForReply(
	handleConn ClientConnectionHandler,
) ([]byte, error) {
	conn, err := c.Dial()
	if err != nil {
		// TODO: think if not to wrap the error
		return nil, err
	}

	return handleConn(conn)
}

func (c *Client) DialAndWaitForStringReply(
	handleConnection ClientConnectionHandler,
) (string, error) {
	conn, err := c.Dial()
	if err != nil {
		// TODO: think if not to wrap the error
		return "", err
	}

	bs, err := handleConnection(conn)
	if err != nil {
		return "", fmt.Errorf("cannot handle connection: %s", err)
	}

	return string(bs), nil
}

func (c *Client) Close() error {
	return c.connection.Close()
}
