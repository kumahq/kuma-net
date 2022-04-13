package netns

import (
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/kumahq/kuma-net/test/framework/tcp/socket_options"
)

const Loopback = "lo"

type Veth struct {
	Name        string
	PeerName    string
	Address     string
	PeerAddress string
}

type Config struct {
	Name string
	Veth *Veth
}

type NetNS struct {
	name       string
	ns         netns.NsHandle
	originalNS netns.NsHandle
	veth       *netlink.Veth
}

func (ns *NetNS) Name() string {
	return ns.name
}

func (ns *NetNS) Set() error {
	if err := netns.Set(ns.ns); err != nil {
		return fmt.Errorf("cannot switch to the network namespace %q: %s", ns.ns.String(), err)
	}

	return nil
}

func (ns *NetNS) Unset() error {
	if err := netns.Set(ns.originalNS); err != nil {
		return fmt.Errorf(
			"cannot switch to the original network namespace %q: %s",
			ns.originalNS.String(),
			err,
		)
	}

	return nil
}

func (ns *NetNS) StartTCPServer(
	address string,
	callbacks ...func() error,
) (<-chan struct{}, <-chan error) {
	readyC := make(chan struct{})
	errorC := make(chan error)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := ns.Set(); err != nil {
			errorC <- fmt.Errorf("cannot start TCP server: %s", err)
		}

		for _, callback := range callbacks {
			if err := callback(); err != nil {
				errorC <- err
			}
		}

		l, err := net.Listen("tcp", address)
		if err != nil {
			errorC <- fmt.Errorf("cannot start TCP server: %s", err)
		}
		defer l.Close()

		close(readyC)

		for {
			conn, err := l.Accept()
			if err != nil {
				errorC <- fmt.Errorf("cannot accept connection: %s", err)
				break
			}

			go func() {
				tcpConn := conn.(*net.TCPConn)

				originalDst, err := socket_options.ExtractOriginalDst(tcpConn)
				if err != nil {
					tcpConn.Write([]byte(err.Error()))
				} else {
					tcpConn.Write(originalDst.Bytes())
				}

				tcpConn.CloseWrite()
			}()
		}
	}()

	return readyC, errorC
}

func (ns *NetNS) Cleanup() error {
	var errs []string

	if ns.originalNS.IsOpen() {
		if err := ns.originalNS.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("cannot close the original namespace fd: %s", err))
		}
	}

	if ns.ns.IsOpen() {
		if err := ns.ns.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("cannot close the network namespace fd: %s", err))
		}
	}

	if err := netns.DeleteNamed(ns.name); err != nil {
		errs = append(errs, fmt.Sprintf("cannot delete network namespace: %s", err))
	}

	if err := netlink.LinkDel(ns.veth); err != nil {
		errs = append(errs, fmt.Sprintf("cannot delete veth interfaces: %s", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

func newVeth(config *Veth) *netlink.Veth {
	la := netlink.NewLinkAttrs()
	la.Name = config.Name

	return &netlink.Veth{
		LinkAttrs: la,
		PeerName:  config.PeerName,
	}
}

func NewNetNS(config *Config) (*NetNS, error) {
	var ns *NetNS

	done := make(chan error)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		originalNS, err := netns.Get()
		if err != nil {
			done <- fmt.Errorf("cannot get the original network namespace: %s", err)
		}

		// Create a pair of veth interfaces
		veth := newVeth(config.Veth)
		if addLinkErr := netlink.LinkAdd(veth); addLinkErr != nil {
			done <- fmt.Errorf("cannot add veth interfaces: %s", addLinkErr)
		}

		mainLink, err := netlink.LinkByName(veth.Name)
		if err != nil {
			done <- fmt.Errorf("cannot get main veth interface: %s", err)
		}

		mainAddr, err := netlink.ParseAddr(config.Veth.Address)
		if err != nil {
			done <- fmt.Errorf("cannot parse main veth interface address: %s", err)
		}

		if err := netlink.AddrAdd(mainLink, mainAddr); err != nil {
			done <- fmt.Errorf("cannot add address to main veth interface: %s", err)
		}

		if err := netlink.LinkSetUp(mainLink); err != nil {
			done <- fmt.Errorf("cannot set main veth interface up: %s", err)
		}

		// peer link - interface which will be moved to the custom network namespace
		peerLink, err := netlink.LinkByName(veth.PeerName)
		if err != nil {
			done <- fmt.Errorf("cannot get peer veth interface: %s", err)
		}

		// Create a new network namespace (when creating new namespace,
		// we are automatically switching into it)
		//
		// netns.NewNamed calls unix.Unshare(CLONE_NEWNET) which requires CAP_SYS_ADMIN
		// capability (ref. https://man7.org/linux/man-pages/man2/unshare.2.html)
		newNS, err := netns.NewNamed(config.Name)
		if err != nil {
			done <- fmt.Errorf("cannot create new network namespace: %s", err)
		}

		// set the loopback interface up
		lo, err := netlink.LinkByName(Loopback)
		if err != nil {
			done <- fmt.Errorf("cannot get loopback interface: %s", err)
		}

		if err := netlink.LinkSetUp(lo); err != nil {
			done <- fmt.Errorf("cannot set loopback interface up: %s", err)
		}

		// switch to the original namespace to assign veth peer interface
		// to our freshly made namespace
		if err := netns.Set(originalNS); err != nil {
			done <- fmt.Errorf("cannot switch to original network namespace: %s", err)
		}

		// Adding an interface to a network namespace will cause the interface
		// to lose its existing IP address, so we cannot assign it earlier.
		if err := netlink.LinkSetNsFd(peerLink, int(newNS)); err != nil {
			done <- fmt.Errorf("cannot put peer veth interface inside new network interface: %s", err)
		}

		if err := netns.Set(newNS); err != nil {
			done <- fmt.Errorf("cannot switch to new network interface: %s", err)
		}

		peerAddr, err := netlink.ParseAddr(config.Veth.PeerAddress)
		if err != nil {
			done <- fmt.Errorf("cannot parse peer veth interface address: %s", err)
		}

		if err := netlink.AddrAdd(peerLink, peerAddr); err != nil {
			done <- fmt.Errorf("cannot add address to peer veth interface: %s", err)
		}

		if err := netlink.LinkSetUp(peerLink); err != nil {
			done <- fmt.Errorf("cannot set peer veth interface up: %s", err)
		}

		if err := netns.Set(originalNS); err != nil {
			done <- fmt.Errorf("cannot switch to original network namespace: %s", err)
		}

		ns = &NetNS{
			name:       config.Name,
			ns:         newNS,
			originalNS: originalNS,
			veth:       veth,
		}

		done <- nil
	}()

	return ns, <-done
}
