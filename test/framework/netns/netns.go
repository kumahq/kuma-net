package netns

import (
	"fmt"
	"math"
	"net"
	"runtime"
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/kumahq/kuma-net/test/framework/tcp/socket_options"
)

// CLONE_NEWNET requires Linux Kernel 3.0+

var suffixes = map[uint8]map[uint8]struct{}{}

const Loopback = "lo"

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

func newVeth(nameSeed string, suffixA, suffixB uint8) *netlink.Veth {
	suffix := fmt.Sprintf("-%d%d", suffixA, suffixB)
	la := netlink.NewLinkAttrs()
	la.Name = fmt.Sprintf("%smain%s", nameSeed, suffix)

	return &netlink.Veth{
		LinkAttrs: la,
		PeerName:  fmt.Sprintf("%speer%s", nameSeed, suffix),
	}
}

type Builder struct {
	nameSeed string
}

func (b *Builder) WithNameSeed(seed string) *Builder {
	b.nameSeed = seed

	return b
}

// we need some values which will make all names we will use to create resources
// (netns name, ip addresses, veth interface names) unique.
// I decided that the easiest way go achieve this uniqueness is to generate
// 2 uint8 values which will be representing second and third octets in the 10.0.0.0/24
// subnet, which will allow us to generate ip (v4) addresses as well as the names.
// genSuffixes will check if any network interface has already assigned subnet
// within the range we are interested in and ignore suffixes in this range
// Example of names regarding generated suffixes:
// suffixes: 123, 254
// 	netns name:			kmesh-123254
// 	veth main name:		kmesh-main-123254
// 	veth peer name: 	kmesh-peer-123254
// 	veth main address:	10.123.254.1
// 	veth main cidr:		10.123.254.1/24
// 	veth peer address:	10.123.254.2
// 	veth peer cidr:		10.123.254.2/24
func genSuffixes() (uint8, uint8, error) {
	ifaceAddresses, err := getIfaceAddresses()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get network interface addresses: %s", err)
	}

	for i := uint8(1); i < math.MaxUint8; i++ {
		var s map[uint8]struct{}
		var ok bool

		if s, ok = suffixes[i]; ok {
			if len(s) >= math.MaxUint8-1 {
				continue
			}
		} else {
			suffixes[i] = map[uint8]struct{}{
				1: {},
			}

			if ifaceContainsAddress(ifaceAddresses, net.IP{10, i, 1, 0}) {
				continue
			}

			return i, 1, nil
		}

		for j := uint8(1); j < math.MaxUint8; j++ {
			if _, ok := s[j]; !ok {
				s[j] = struct{}{}

				if !ifaceContainsAddress(ifaceAddresses, net.IP{10, i, j, 0}) {
					return i, j, nil
				}
			}
		}
	}

	return 0, 0, fmt.Errorf("out of available suffixes")
}

func getIfaceAddresses() ([]*net.IPNet, error) {
	var addresses []*net.IPNet

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("cannot list network interfaces: %s", err)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("cannot list network interface's addresses: %s", err)
		}

		for _, addr := range addrs {
			if err != nil {
				return nil, fmt.Errorf("cannot resolve tcp address: %s", err)
			}

			addresses = append(addresses, addr.(*net.IPNet))
		}
	}

	return addresses, nil
}

func ifaceContainsAddress(addresses []*net.IPNet, address net.IP) bool {
	for _, ipNet := range addresses {
		if ipNet.Contains(address) {
			return true
		}
	}

	return false
}

func genAddress(octet2, octet3, octet4 uint8) string {
	return fmt.Sprintf("10.%d.%d.%d", octet2, octet3, octet4)
}

func genCIDRAddress(octet2, octet3, octet4 uint8) string {
	return fmt.Sprintf("%s/24", genAddress(octet2, octet3, octet4))
}

func genNetNSName(nameSeed string, suffixA, suffixB uint8) string {
	return fmt.Sprintf("%s%d%d", nameSeed, suffixA, suffixB)
}

func (b *Builder) Build() (*NetNS, error) {
	suffixA, suffixB, err := genSuffixes()
	if err != nil {
		return nil, fmt.Errorf("cannot generate suffixes: %s", err)
	}

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
		veth := newVeth(b.nameSeed, suffixA, suffixB)
		if addLinkErr := netlink.LinkAdd(veth); addLinkErr != nil {
			done <- fmt.Errorf("cannot add veth interfaces: %s", addLinkErr)
		}

		mainLink, err := netlink.LinkByName(veth.Name)
		if err != nil {
			done <- fmt.Errorf("cannot get main veth interface: %s", err)
		}

		mainCIDR := genCIDRAddress(suffixA, suffixB, 1)
		mainAddr, err := netlink.ParseAddr(mainCIDR)
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
		nsName := genNetNSName(b.nameSeed, suffixA, suffixB)
		newNS, err := netns.NewNamed(nsName)
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

		peerCIDR := genCIDRAddress(suffixA, suffixB, 2)
		peerAddr, err := netlink.ParseAddr(peerCIDR)
		if err != nil {
			done <- fmt.Errorf("cannot parse peer veth interface address: %s", err)
		}

		if err := netlink.AddrAdd(peerLink, peerAddr); err != nil {
			done <- fmt.Errorf("cannot add address to peer veth interface: %s", err)
		}

		if err := netlink.LinkSetUp(peerLink); err != nil {
			done <- fmt.Errorf("cannot set peer veth interface up: %s", err)
		}

		if err := netlink.RouteAdd(&netlink.Route{Gw: mainAddr.IP}); err != nil {
			done <- fmt.Errorf("cannot set the default route: %s", err)
		}

		if err := netns.Set(originalNS); err != nil {
			done <- fmt.Errorf("cannot switch to original network namespace: %s", err)
		}

		ns = &NetNS{
			name:       nsName,
			ns:         newNS,
			originalNS: originalNS,
			veth: &Veth{
				veth:        veth,
				name:        veth.Name,
				peerName:    veth.PeerName,
				address:     genAddress(suffixA, suffixB, 1),
				cidr:        mainCIDR,
				peerAddress: genAddress(suffixA, suffixB, 2),
				peerCIDR:    peerCIDR,
			},
		}

		close(done)
	}()

	return ns, <-done
}

func NewNetNS() *Builder {
	return &Builder{
		nameSeed: "kmesh-",
	}
}
