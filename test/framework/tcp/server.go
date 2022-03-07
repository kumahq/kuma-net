package tcp

import (
	"fmt"
	"net"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/test/framework/tcp/socket_options"
)

type ConnectionHandler func(conn *net.TCPConn) error

func ReplyWithOriginalDestination(conn *net.TCPConn) error {
	originalDst, err := socket_options.ExtractOriginalDst(conn)
	if err != nil {
		return fmt.Errorf("cannot extract original destination: %s", err)
	}

	if _, err := conn.Write([]byte(originalDst.String())); err != nil {
		return fmt.Errorf("cannot send original destination to the connection: %s", err)
	}

	return nil
}

func NoopConnectionHandler(*net.TCPConn) error {
	return nil
}

type Server struct {
	host             string
	port             *uint16
	listener         *net.TCPListener
	handleConnection ConnectionHandler
}

func NewServer() *Server {
	return &Server{
		host:             "localhost",
		handleConnection: NoopConnectionHandler,
	}
}

func (s *Server) WithHost(host string) *Server {
	s.host = host

	return s
}

func (s *Server) WithPort(port uint16) *Server {
	s.port = &port

	return s
}

func (s *Server) WithConnectionHandler(connectionHandler ConnectionHandler) *Server {
	s.handleConnection = connectionHandler

	return s
}

func (s *Server) Listen() *Server {
	port := strconv.Itoa(int(*s.port))
	hostPort := net.JoinHostPort(s.host, port)

	address, err := net.ResolveTCPAddr(Network, hostPort)
	Expect(err).To(BeNil())

	listener, err := net.ListenTCP(Network, address)
	Expect(err).To(BeNil())

	s.listener = listener

	go func() {
		defer GinkgoRecover()

		conn, err := listener.AcceptTCP()
		Expect(err).To(BeNil())

		Expect(s.handleConnection(conn)).To(Succeed())
	}()

	return s
}

func (s *Server) Close() error {
	return s.listener.Close()
}
