package tcp

import (
	"fmt"
	"net"
	"runtime"

	"github.com/onsi/ginkgo/v2"

	"github.com/kumahq/kuma-net/test/framework/netns"
	"github.com/kumahq/kuma-net/test/framework/tcp/socket_options"
)

// CloseConn will run CloseWrite member function on provided as a parameter
// *net.TCPConn
func CloseConn(conn *net.TCPConn) {
	_ = conn.CloseWrite()
}

// ReplyWithOriginalDst will send to provided *net.TCPConn extracted
// from the socket's original destination (as []byte) of the connection
// if extraction will succeed, and the error message (as []byte) otherwise
func ReplyWithOriginalDst(conn *net.TCPConn) {
	originalDst, err := socket_options.ExtractOriginalDst(conn)
	if err != nil {
		_, _ = conn.Write([]byte(err.Error()))
	} else {
		_, _ = conn.Write(originalDst.Bytes())
	}
}

// ReplyWith will return a function which will send to provided *net.TCPConn
// the message (string) from closure which was provided as a parameter to
// ReplyWith function
func ReplyWith(msg string) func(conn *net.TCPConn) {
	return func(conn *net.TCPConn) {
		_, _ = conn.Write([]byte(msg))
	}
}

// UnsafeStartTCPServer will start TCP server in provided *netns.NesNS.
// Every initialized tcp connection will be processed via provided callback
// functions. It was named UnsafeStartTCPServer instead of StartTCPServer
// because you have to be very cautious and remember to not spawn new goroutines
// inside provided callbacks (more info in warning below)
//
// WARNING!:
//  Don't spawn new goroutines inside callback functions as the goroutine inside
//  UnsafeStartTCPServer function have exclusive access to the current network
//  namespace, and you should assume, that any new goroutine will be placed
//  in a different namespace
func UnsafeStartTCPServer(
	ns *netns.NetNS,
	address string,
	callbacks ...func(conn *net.TCPConn),
) (<-chan struct{}, <-chan error) {
	readyC := make(chan struct{})
	errorC := make(chan error)

	go func() {
		defer ginkgo.GinkgoRecover()
		defer close(errorC)

		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := ns.Set(); err != nil {
			errorC <- fmt.Errorf("cannot start TCP server: %s", err)
		}
		defer ns.Unset()

		l, err := net.Listen("tcp", address)
		if err != nil {
			errorC <- fmt.Errorf("cannot start TCP server: %s", err)
		}
		defer l.Close()

		close(readyC)

		// As we have to remember that when locking os thread inside goroutine,
		// any new goroutine will be spawned in different os thread,
		// our tcp server is designed to handle just one connection (not one
		// at a time, but just one at all). In other case we would have to
		// accept new connections inside for loop, which would introduce huge
		// complexity to overcome locking problems as we couldn't handle
		// the connections inside different goroutines.
		conn, err := l.Accept()
		if err != nil {
			errorC <- fmt.Errorf("cannot accept connection: %s", err)
			return
		}

		tcpConn := conn.(*net.TCPConn)

		for _, callback := range callbacks {
			callback(tcpConn)
		}
	}()

	return readyC, errorC
}
