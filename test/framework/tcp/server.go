package tcp

import (
	"fmt"
	"net"
	"strconv"
	"time"

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
	errorsC          chan error
}

func NewServer() *Server {
	return &Server{
		host:             "localhost",
		handleConnection: NoopConnectionHandler,
		errorsC:          make(chan error),
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

func (s *Server) WithConnectionHandler(handleConn ConnectionHandler) *Server {
	s.handleConnection = handleConn

	return s
}

func (s *Server) Listen() (*Server, error) {
	port := strconv.Itoa(int(*s.port))
	hostPort := net.JoinHostPort(s.host, port)

	address, err := net.ResolveTCPAddr(Network, hostPort)
	if err != nil {
		// TODO: Think if not to wrap the error
		return nil, err
	}

	listener, err := net.ListenTCP(Network, address)
	if err != nil {
		// TODO: Think if not to wrap the error
		return nil, err
	}

	s.listener = listener

	go func() {
		conn, err := listener.AcceptTCP()
		if err != nil {
			// TODO: Think if not to wrap the error
			s.errorsC <- err
		}

		if err := s.handleConnection(conn); err != nil {
			// TODO: Think if not to wrap the error
			s.errorsC <- err
		}

		close(s.errorsC)
	}()

	return s, nil
}

func (s *Server) Close() error {
	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("closing of the listener failed: %s", err)
	}

	t := time.NewTimer(time.Second)

	select {
	case <-t.C:
		// TODO: improve error message
		return fmt.Errorf("close timeout")
	case err := <-s.errorsC:
		// TODO: Think if not to wrap the error
		return err
	}
}
