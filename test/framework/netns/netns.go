package netns

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
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
	name string
	ns   netns.NsHandle
	veth *netlink.Veth
}

func (ns *NetNS) Name() string {
	return ns.name
}

func (ns *NetNS) Cleanup() error {
	var errs []string

	if err := ns.ns.Close(); err != nil {
		errs = append(errs, fmt.Sprintf("cannot close network namespace fd: %s", err))
	}

	if err := netns.DeleteNamed(ns.name); err != nil {
		errs = append(errs, fmt.Sprintf("cannot delete network namespace: %s", err))
	}

	if err := netlink.LinkDel(ns.veth); err != nil {
		errs = append(errs, fmt.Sprintf("cannot delete veth interfaces: %s", err))
	}

	return fmt.Errorf("cleanup failed:\n- %s", strings.Join(errs, "\n -"))
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
	// Lock the OS Thread, so we don't accidentally switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	originalNS, err := netns.Get()
	if err != nil {
		return nil, fmt.Errorf("cannot get the original network namespace: %s", err)
	}
	defer originalNS.Close()

	// Create a pair of veth interfaces
	veth := newVeth(config.Veth)
	if err := netlink.LinkAdd(veth); err != nil {
		return nil, fmt.Errorf("cannot add veth interfaces: %s", err)
	}

	mainLink, err := netlink.LinkByName(veth.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot get main veth interface: %s", err)
	}

	mainAddr, err := netlink.ParseAddr(config.Veth.Address)
	if err != nil {
		return nil, fmt.Errorf("cannot parse main veth interface address: %s", err)
	}

	if err := netlink.AddrAdd(mainLink, mainAddr); err != nil {
		return nil, fmt.Errorf("cannot add address to main veth interface: %s", err)
	}

	if err := netlink.LinkSetUp(mainLink); err != nil {
		return nil, fmt.Errorf("cannot set main veth interface up: %s", err)
	}

	// peer link - interface which will be moved to the custom network namespace
	peerLink, err := netlink.LinkByName(veth.PeerName)
	if err != nil {
		return nil, fmt.Errorf("cannot get peer veth interface: %s", err)
	}

	// Create a new network namespace (when creating new namespace,
	// we are automatically switching into it)
	newNS, err := netns.NewNamed(config.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot create new network namespace: %s", err)
	}

	// set the loopback interface up
	lo, err := netlink.LinkByName(Loopback)
	if err != nil {
		return nil, fmt.Errorf("cannot get loopback interface: %s", err)
	}

	if err := netlink.LinkSetUp(lo); err != nil {
		return nil, fmt.Errorf("cannot set loopback interface up: %s", err)
	}

	// switch to the original namespace to assign veth peer interface
	// to our freshly made namespace
	if err := netns.Set(originalNS); err != nil {
		return nil, fmt.Errorf("cannot switch to original network namespace: %s", err)
	}

	// Adding an interface to a network namespace will cause the interface
	// to lose its existing IP address, so we cannot assign it earlier.
	if err := netlink.LinkSetNsFd(peerLink, int(newNS)); err != nil {
		return nil, fmt.Errorf("cannot put peer veth interface inside new network interface: %s", err)
	}

	if err := netns.Set(newNS); err != nil {
		return nil, fmt.Errorf("cannot switch to new network interface: %s", err)
	}

	peerAddr, err := netlink.ParseAddr(config.Veth.PeerAddress)
	if err != nil {
		return nil, fmt.Errorf("cannot parse peer veth interface address: %s", err)
	}

	if err := netlink.AddrAdd(peerLink, peerAddr); err != nil {
		return nil, fmt.Errorf("cannot add address to peer veth interface: %s", err)
	}

	if err := netlink.LinkSetUp(peerLink); err != nil {
		return nil, fmt.Errorf("cannot set peer veth interface up: %s", err)
	}

	if err := netns.Set(originalNS); err != nil {
		return nil, fmt.Errorf("cannot switch to original network namespace: %s", err)
	}

	return &NetNS{
		name: config.Name,
		ns:   newNS,
		veth: veth,
	}, nil
}
