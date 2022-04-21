package udp

import (
	"fmt"
	"net"

	"github.com/kumahq/kuma-net/test/framework/ip"
)

// DialWithHelloMsgAndGetReply will open a UDP socket with provided UDP address
// and send there a helloMsg (fmt.Stringer), and block goroutine waiting for the
// message back, which will be returned as a string
func DialWithHelloMsgAndGetReply(
	address *net.UDPAddr,
	helloMsg fmt.Stringer,
) (string, error) {
	socket, err := net.DialUDP("udp", nil, address)
	if err != nil {
		return "", fmt.Errorf("cannot dial provided address: %s", err)
	}
	defer socket.Close()

	n, err := socket.Write([]byte(helloMsg.String()))
	if err != nil {
		return "", fmt.Errorf("cannot send hello message %q: %s", helloMsg, err)
	}

	buf := make([]byte, 1024)
	n, err = socket.Read(buf)
	if err != nil {
		return "", fmt.Errorf("cannot read replied message: %s", err)
	}

	return string(buf[:n]), nil
}

// GenRandomAddress will generate random *net.UDPAddr with provided port
func GenRandomAddress(port uint16) *net.UDPAddr {
	return &net.UDPAddr{
		IP:   ip.GenRandomIPv4(),
		Port: int(port),
	}
}
