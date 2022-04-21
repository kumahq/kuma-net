package netns

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"syscall"

	"github.com/onsi/ginkgo/v2"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// CLONE_NEWNET requires Linux Kernel 3.0+

type Veth struct {
	veth        *netlink.Veth
	name        string
	peerName    string
	address     string
	cidr        string
	peerAddress string
	peerCIDR    string
}

func (v *Veth) Veth() *netlink.Veth {
	return v.veth
}

func (v *Veth) PeerName() string {
	return v.peerName
}

func (v *Veth) Address() string {
	return v.address
}

func (v *Veth) Cidr() string {
	return v.cidr
}

func (v *Veth) PeerAddress() string {
	return v.peerAddress
}

func (v *Veth) PeerCIDR() string {
	return v.peerCIDR
}

func (v *Veth) Name() string {
	return v.name
}

type NetNS struct {
	name       string
	ns         netns.NsHandle
	originalNS netns.NsHandle
	veth       *Veth
}

func (ns *NetNS) Name() string {
	return ns.name
}

func (ns *NetNS) Veth() *Veth {
	return ns.veth
}

func (ns *NetNS) Set() error {
	if err := netns.Set(ns.ns); err != nil {
		return fmt.Errorf("cannot switch to the network namespace %q: %s", ns.ns.String(), err)
	}

	return nil
}

// UnsafeExec will execute provided callback function in the created network namespace
// from the *NetNS. It was named UnsafeExec instead of Exec as you have to be very
// cautious and remember to not spawn new goroutines inside provided callback (more
// info in warning below)
//
// WARNING!:
//  Don't spawn new goroutines inside callback functions as the one inside UnsafeExec
//  function have exclusive access to the current network namespace, and you should
//  assume, that any new goroutine will be placed in the different namespace
func (ns *NetNS) UnsafeExec(callback func()) <-chan error {
	done := make(chan error)

	go func() {
		defer ginkgo.GinkgoRecover()
		defer close(done)

		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := ns.Set(); err != nil {
			done <- fmt.Errorf("cannot set the namespace %q: %s", ns.name, err)
		}
		defer ns.Unset()

		callback()
	}()

	return done
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
	callback func(conn *net.TCPConn),
) (<-chan struct{}, <-chan error) {
	readyC := make(chan struct{})
	errorC := make(chan error)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := ns.Set(); err != nil {
			errorC <- fmt.Errorf("cannot start TCP server: %s", err)
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
				defer tcpConn.CloseWrite()

				callback(tcpConn)
			}()
		}
	}()

	return readyC, errorC
}

func (ns *NetNS) StartUDPServer(
	address string,
	setuid uintptr,
	callbacks ...func() error,
) (<-chan struct{}, <-chan error) {
	readyC := make(chan struct{})
	errorC := make(chan error)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if err := ns.Set(); err != nil {
			errorC <- fmt.Errorf("cannot switch to the namespace: %s", err)
		}
		defer ns.Unset()

		for _, callback := range callbacks {
			if err := callback(); err != nil {
				errorC <- err
			}
		}

		addr, err := net.ResolveUDPAddr("udp", address)
		if err != nil {
			errorC <- fmt.Errorf("cannot parse address and port %q: %s", address, err)
		}

		// This is very hacky way of making one process to operate under multiple UIDs,
		// which breaks POSIX semantics (ref. https://man7.org/linux/man-pages/man7/nptl.7.html)
		// The go's native syscall.Setuid is designed in a way, that all threads
		// will switch to provided UID, so we have to use the Linux's setuid() syscall
		// directly (it doesn't honor POSIX semantics).
		//
		// This logic exists to potentially run tests with DNS UDP conntrack zone splitting
		// enabled.
		//
		// ref. https://stackoverflow.com/a/66523695
		if setuid != 0 {
			if _, _, e := syscall.RawSyscall(syscall.SYS_SETUID, setuid, 0, 0); e != 0 {
				errorC <- fmt.Errorf("cannot exec syscall.SYS_SETUID (error number: %d)", e)
			}
		}

		udpConn, err := net.ListenUDP("udp", addr)
		if err != nil {
			errorC <- fmt.Errorf("cannot listen udp on address %q: %s", address, err)
		}
		defer udpConn.Close()

		// At this point we are ready for accepting UDP datagrams
		close(readyC)

		buf := make([]byte, 1024)
		n, clientAddr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			errorC <- fmt.Errorf("cannot read from udp: %s", err)
		}

		_, err = udpConn.WriteToUDP(buf[:n], clientAddr)
		if err != nil {
			errorC <- fmt.Errorf("cannot write to udp: %s", err)
		}

		if err := udpConn.Close(); err != nil {
			errorC <- fmt.Errorf("cannot close udp connection: %s", err)
		}

		close(errorC)
	}()

	return readyC, errorC
}

func (ns *NetNS) Cleanup() error {
	if ns == nil {
		return nil
	}

	done := make(chan error)

	// It's necessary to run the code in separate goroutine to lock the os thread
	// to pin the network namespaces for our purposes
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
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

		if err := netns.DeleteNamed(ns.Name()); err != nil {
			errs = append(errs, fmt.Sprintf("cannot delete network namespace: %s", err))
		}

		veth := ns.Veth().Veth()
		if err := netlink.LinkDel(veth); err != nil {
			errs = append(errs, fmt.Sprintf("cannot delete veth interface %q: %s", veth.Name, err))
		}

		if len(errs) > 0 {
			done <- fmt.Errorf("cleanup failed:\n  - %s", strings.Join(errs, "\n  - "))
		}

		close(done)
	}()

	return <-done
}
