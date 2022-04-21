package tcp

import (
	"net"

	"github.com/kumahq/kuma-net/test/framework/tcp/socket_options"
)

// ReplyWithOriginalDst will send to provided *net.TCPConn extracted from the socket
// original destination (as []byte) of the connection if extraction will succeed,
// and the error message (as []byte) otherwise
func ReplyWithOriginalDst(conn *net.TCPConn) {
	originalDst, err := socket_options.ExtractOriginalDst(conn)
	if err != nil {
		_, _ = conn.Write([]byte(err.Error()))
	} else {
		_, _ = conn.Write(originalDst.Bytes())
	}
}

// ReplyWith will return a function which will send to provided *net.TCPConn
// the message (string) from closure which was provided as a parameter to ReplyWith
// function
func ReplyWith(msg string) func(conn *net.TCPConn) {
	return func(conn *net.TCPConn) {
		_, _ = conn.Write([]byte(msg))
	}
}
