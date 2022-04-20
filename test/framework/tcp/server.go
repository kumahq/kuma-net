package tcp

import (
	"net"

	"github.com/kumahq/kuma-net/test/framework/tcp/socket_options"
)

func ReplyWithOriginalDst(conn *net.TCPConn) {
	originalDst, err := socket_options.ExtractOriginalDst(conn)
	if err != nil {
		_, _ = conn.Write([]byte(err.Error()))
	} else {
		_, _ = conn.Write(originalDst.Bytes())
	}
}

func ReplyWith(msg []byte) func(conn *net.TCPConn) {
	return func(conn *net.TCPConn) {
		_, _ = conn.Write(msg)
	}
}
