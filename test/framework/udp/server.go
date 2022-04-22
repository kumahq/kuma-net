package udp

import (
	"fmt"
	"net"
	"runtime"
	"syscall"

	"github.com/onsi/ginkgo/v2"

	"github.com/kumahq/kuma-net/test/framework/netns"
)

// UnsafeStartUDPServer will start TCP server in provided *netns.NesNS.
// Every initialized udp "connection" will be processed via provided callback
// functions. It was named UnsafeStartUDPServer instead of StartUDPServer
// because you have to be very cautious and remember to not spawn new goroutines
// inside provided callbacks (more info in warning below)
//
// WARNING!:
//  Don't spawn new goroutines inside callback functions as the goroutine inside
//  UnsafeStartUDPServer function have exclusive access to the current network
//  namespace, and you should assume, that any new goroutine will be placed
//  in a different namespace
func UnsafeStartUDPServer(
	ns *netns.NetNS,
	address string,
	setuid uintptr,
	callbacks ...func() error,
) (<-chan struct{}, <-chan error) {
	readyC := make(chan struct{})
	errorC := make(chan error)

	go func() {
		defer ginkgo.GinkgoRecover()
		defer close(errorC)

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

		// As we have to remember that when locking os thread inside goroutine,
		// any new goroutine will be spawned in different os thread,
		// our udp server is designed to handle just one connection (not one
		// at a time, but just one at all). In other case we would have to
		// accept new connections inside for loop, which would introduce huge
		// complexity to overcome locking problems as we couldn't handle
		// the connections inside different goroutines.

		buf := make([]byte, 1024)
		n, clientAddr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			errorC <- fmt.Errorf("cannot read from udp: %s", err)
		}

		if _, err := udpConn.WriteToUDP(buf[:n], clientAddr); err != nil {
			errorC <- fmt.Errorf("cannot write to udp: %s", err)
		}

		if err := udpConn.Close(); err != nil {
			errorC <- fmt.Errorf("cannot close udp connection: %s", err)
		}
	}()

	return readyC, errorC
}
