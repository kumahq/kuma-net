package socket_options

import (
	"net"
	"syscall"
)

// SO_ORIGINAL_DST is the "optname" for "getsockopt" syscall
// ref: https://man7.org/linux/man-pages/man2/getsockopt.2.html
// ref: https://github.com/torvalds/linux/blob/5bfc75d92efd494db37f5c4c173d3639d4772966/include/uapi/linux/netfilter_ipv4.h#L52
//goland:noinspection GoSnakeCaseUsage
const SO_ORIGINAL_DST = 80

type OriginalDst struct {
	*net.TCPAddr
}

func (o *OriginalDst) Bytes() []byte {
	return []byte(o.String())
}

func ParseOriginalDst(multiaddr [16]byte) *OriginalDst {
	address := net.IPv4(multiaddr[4], multiaddr[5], multiaddr[6], multiaddr[7])
	port := uint16(multiaddr[2])<<8 + uint16(multiaddr[3])

	return &OriginalDst{
		TCPAddr: &net.TCPAddr{
			IP:   address,
			Port: int(port),
		},
	}
}

func ExtractOriginalDst(conn *net.TCPConn) (*OriginalDst, error) {
	file, err := conn.File()
	if err != nil {
		// TODO (bartsmykla): wrap the error maybe?
		return nil, err
	}

	fd := int(file.Fd())

	mreq, err := syscall.GetsockoptIPv6Mreq(fd, syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok && errno == syscall.ENOENT {
			return nil, nil
		}

		// TODO (bartsmykla): wrap the error maybe?
		return nil, err
	}

	return ParseOriginalDst(mreq.Multiaddr), nil
}
