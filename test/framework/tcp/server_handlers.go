package tcp

import (
	"fmt"
	"net"
	"strings"

	"github.com/kumahq/kuma-cni/test/framework/tcp/socket_options"
)

// UnexpectedConnMessage ir a message which a TCP client would receive from
// the server when the server receive connection it is not expecting
// (see UnexpectedConn for more information)
const UnexpectedConnMessage = "This connection should not happen. " +
	"If you see this, something went wrong!"

// ConnHandler
// TODO (bartsmykla): write description
type ConnHandler func(conn *net.TCPConn) error

// ReplyWithOriginalDst will extract original destination address from provided
// *net.TCPConn and then send it as a []byte back to the same *net.TCPConn
// It will fail when:
// - original destination extraction will fail;
// - sending extracted original destination of provided *net.TCPConn to the same
//   *net.TCPConn will fail.
// TODO (bartsmykla): write tests
func ReplyWithOriginalDst(conn *net.TCPConn) error {
	originalDst, err := socket_options.ExtractOriginalDst(conn)
	if err != nil {
		return fmt.Errorf("cannot extract original destination: %s", err)
	}

	if _, err := conn.Write([]byte(originalDst.String())); err != nil {
		return fmt.Errorf("cannot send original destination to the connection: %s", err)
	}

	return nil
}

// UnexpectedConn will send to the provided *net.TCPConn string with
// information that this connection should not happen. It's used when you want
// to test if your packet redirection logic work, and when you don't want
// the server to receive any connection, i.e:
// 1. creating 2 TCP servers - ServerA(port:8080)(ConnHandler:any) and
//    ServerB(port:7070)(ConnHandler:UnexpectedConn)
// 2. redirecting all incoming TCP packets to ServerA
// 3. sending request to ServerB
// 4. expecting:
//    a) ServerA to receive a connection addressed to ServerB
//    b) ServerB to not receive any connection thus if UnexpectedConn will be
//       ever called, it means that the packet redirection doesn't work
//       as expected
// TODO (bartsmykla): write tests
func UnexpectedConn(conn *net.TCPConn) error {
	if _, err := conn.Write([]byte(UnexpectedConnMessage)); err != nil {
		return fmt.Errorf("cannot send original destination to the connection: %s", err)
	}

	return nil
}

// NoopConnHandler
// TODO (bartsmykla): write description
// TODO (bartsmykla): write tests
func NoopConnHandler(*net.TCPConn) error {
	return nil
}

// NoopConnErrHandler
// TODO (bartsmykla): write description
// TODO (bartsmykla): write tests
func NoopConnErrHandler(err error) error {
	return err
}

// ConnErrHandler
// TODO (bartsmykla): write description
type ConnErrHandler func(err error) error

// IgnoreUseClosedNetworkConnection
// TODO (bartsmykla): write description
// TODO (bartsmykla): write tests
func IgnoreUseClosedNetworkConnection(err error) error {
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		return err
	}

	return nil
}
