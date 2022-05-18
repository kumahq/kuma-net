package netns

import (
	"fmt"
	"math"
	"net"
	"runtime"
	"strconv"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const Loopback = "lo"

var suffixes = map[uint8]map[uint8]struct{}{}

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
	ipv6     bool
	// beforeExecFuncs are functions which should be run whenever we want to execute
	// anything inside the network namespace. For example if we want to test
	// the dns conntrack zone splitting we have to reduce the amount of available
	// local ports by writing to /proc/sys/net/ipv4/ip_local_port_range
	// (equivalent of `echo "32768   32770" > /proc/sys/net/ipv4/ip_local_port_range`).
	// By doing so we have to remember that this change is ephemeral and will be
	// applied only for the locked goroutine which it was invoked from
	beforeExecFuncs []func() error
}

func (b *Builder) WithNameSeed(seed string) *Builder {
	b.nameSeed = seed

	return b
}

func (b *Builder) WithIPv6(value bool) *Builder {
	b.ipv6 = value

	return b
}

func (b *Builder) WithBeforeExecFuncs(fns ...func() error) *Builder {
	b.beforeExecFuncs = append(b.beforeExecFuncs, fns...)

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
// 	veth peer name:		kmesh-peer-123254
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

func genIPv4IPNet(octet2, octet3, octet4 uint8) *net.IPNet {
	return &net.IPNet{
		IP:   net.IP{10, octet2, octet3, octet4},
		Mask: net.CIDRMask(24, 32),
	}
}

func genIPv6IPNet(octet1, octet2, octet3 uint8) *net.IPNet {
	hex6 := strconv.FormatInt(int64(octet1), 16)
	hex7 := strconv.FormatInt(int64(octet2), 16)
	hex8 := strconv.FormatInt(int64(octet3), 16)

	address := fmt.Sprintf("fd00::%s:%s:%s", hex6, hex7, hex8)

	return &net.IPNet{
		IP:   net.ParseIP(address),
		Mask: net.CIDRMask(64, 128),
	}
}

func genIPNet(ipv6 bool, octet1, octet2, octet3 uint8) *net.IPNet {
	if ipv6 {
		return genIPv6IPNet(octet1, octet2, octet3)
	}

	return genIPv4IPNet(octet1, octet2, octet3)
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

		mainIPNet := genIPNet(b.ipv6, suffixA, suffixB, 1)
		mainAddr, err := netlink.ParseAddr(mainIPNet.String())
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

		peerIPNet := genIPNet(b.ipv6, suffixA, suffixB, 2)
		peerAddr, err := netlink.ParseAddr(peerIPNet.String())
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
				veth:      veth,
				name:      veth.Name,
				peerName:  veth.PeerName,
				ipNet:     mainIPNet,
				peerIPNet: peerIPNet,
			},
			beforeExecFuncs: b.beforeExecFuncs,
		}

		close(done)
	}()

	return ns, <-done
}

func NewNetNSBuilder() *Builder {
	return &Builder{
		nameSeed: "kmesh-",
	}
}
