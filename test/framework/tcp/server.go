package tcp

import (
	"fmt"
	"net"
	"time"
)

const (
	CloseServerTimeout = time.Second
	DefaultServerHost  = "localhost"
)

type Server struct {
	host          string
	port          uint16
	address       *net.TCPAddr
	listener      *net.TCPListener
	handleConn    ConnHandler
	handleConnErr ConnErrHandler
	errorsC       chan error
}

func NewServer() *Server {
	return &Server{
		host:          DefaultServerHost,
		handleConn:    NoopConnHandler,
		handleConnErr: NoopConnErrHandler,
		errorsC:       make(chan error),
	}
}

// WithHost
// TODO (bartsmykla): when WithAddress used in combination with WithHost or/and
//  WithPort these other calls will be ignored - think how to handle this case
//  (maybe not handling it at all is fine, but I'm not sure)
func (s *Server) WithHost(host string) *Server {
	s.host = host

	return s
}

func (s *Server) WithPort(port uint16) *Server {
	s.port = port

	return s
}

func (s *Server) WithAddress(address *net.TCPAddr) *Server {
	s.address = address

	return s
}

func (s *Server) WithConnHandler(handler ConnHandler) *Server {
	s.handleConn = handler

	return s
}

func (s *Server) WithConnErrHandler(handler ConnErrHandler) *Server {
	s.handleConnErr = handler

	return s
}

func (s *Server) init() error {
	if s.handleConn == nil {
		return fmt.Errorf("ConnHandler has to be defined")
	}

	if s.handleConnErr == nil {
		return fmt.Errorf("ConnHandlerErr has to be defined")
	}

	if s.address == nil {
		address, err := ResolveAddress(s.host, s.port)
		if err != nil {
			return fmt.Errorf("address resolving failed: %s", err)
		}
		s.address = address
	}

	return nil
}

func (s *Server) Listen() (*Server, error) {
	if err := s.init(); err != nil {
		return nil, fmt.Errorf("cannot initialize the server: %s", err)
	}

	listener, err := net.ListenTCP(Network, s.address)
	if err != nil {
		return nil, fmt.Errorf("initializing TCP listener failed: %s", err)
	}
	s.listener = listener

	// TODO (bartsmykla): extract it to the separate member function probably
	go func() {
		defer close(s.errorsC)
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			s.errorsC <- fmt.Errorf("accepting a TCP connection failed: %s", err)

			return
		}

		if err := s.handleConn(conn); err != nil {
			s.errorsC <- fmt.Errorf("handling a TCP connection failed: %s", err)
		}
	}()

	return s, nil
}

// Close will try to close the underlying tcp listener
// TODO (bartsmykla): The logic here is probably wrong, as we are waiting for
//  errors from the goroutine handling TCP connections when closing the server
//  and it may introduce deadlock, when you are spawning a server, initializing
//  the connection, and waiting for the result of the connection handler.
//  If connection handler or accepting the tcp connection will fail, then
//  we have a deadlock
func (s *Server) Close() error {
	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("closing of the listener failed: %s", err)
	}

	t := time.NewTimer(CloseServerTimeout)

	select {
	case <-t.C:
		return fmt.Errorf("closing server timeouted after %s", CloseServerTimeout)
	case err := <-s.errorsC:
		return s.handleConnErr(err)
	}
}
