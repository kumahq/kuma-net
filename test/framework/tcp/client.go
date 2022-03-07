package tcp

import (
	"fmt"
	"net"
	"strconv"

	. "github.com/onsi/gomega"
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
	Expect(c.address).ToNot(BeNil())

	return c.address
}

func (c *Client) Dial() *net.TCPConn {
	Expect(c.port).ToNot(BeNil())
	Expect(c.host).ToNot(BeEmpty())

	port := strconv.Itoa(int(*c.port))
	hostPort := net.JoinHostPort(c.host, port)

	address, err := net.ResolveTCPAddr(Network, hostPort)
	Expect(err).To(BeNil())
	c.address = address

	connection, err := net.DialTCP(Network, nil, address)
	Expect(err).To(BeNil())
	c.connection = connection

	return connection
}

func (c *Client) DialAndWaitForReply(
	handleConnection ClientConnectionHandler,
) ([]byte, error) {
	return handleConnection(c.Dial())
}

func (c *Client) DialAndWaitForStringReply(
	handleConnection ClientConnectionHandler,
) (string, error) {
	bs, err := handleConnection(c.Dial())
	if err != nil {
		return "", fmt.Errorf("cannot handle connection: %s", err)
	}

	return string(bs), nil
}

func (c *Client) Close() error {
	return c.connection.Close()
}
