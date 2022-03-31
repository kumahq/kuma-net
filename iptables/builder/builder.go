package builder

import (
	"fmt"
	"net"

	. "github.com/kumahq/kuma-net/iptables/builder/chain"
	"github.com/kumahq/kuma-net/iptables/builder/config"
	. "github.com/kumahq/kuma-net/iptables/builder/rule"
	"github.com/kumahq/kuma-net/iptables/builder/table"
	. "github.com/kumahq/kuma-net/iptables/builder/target"
	. "github.com/kumahq/kuma-net/iptables/consts"
	. "github.com/kumahq/kuma-net/iptables/parameters"
)

type IPTables struct {
	tables []*table.TableBuilder
}

func getLoopback() (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("listig network interfaces failed: %s", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			return &iface, nil
		}
	}

	return nil, fmt.Errorf("it appears there is no loopback interface")
}

func getInterfaceIPv4Address(iface *net.Interface) (string, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("cannot obtain interface addresses: %s", err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			return "", fmt.Errorf(
				"ip appears at leas one of the interface addresses is non IP: %s\t%T",
				addr,
				addr,
			)
		}

		if ipNet.IP.To4() != nil {
			return ipNet.IP.Mask(net.CIDRMask(32, 32)).String(), nil
		}
	}

	return "", fmt.Errorf("it appears interface has no IPv4 address")
}

func Build(config *config.Config) (string, error) {
	loopbackIface, err := getLoopback()
	if err != nil {
		return "", fmt.Errorf("cannot obtain loopback interface: %s", err)
	}

	loopback := loopbackIface.Name

	inboundRedirectPort := config.Redirect.Inbound.Port
	outboundRedirectPort := config.Redirect.Outbound.Port
	dnsRedirectPort := config.Redirect.DNS.Port
	uid := config.Owner.UID
	gid := config.Owner.GID

	shouldRedirectDNS := config.Redirect.DNS.Enabled

	nat := table.Nat()

	// KUMA_MESH_INBOUND_REDIRECT
	meshInboundRedirect := NewChain(config.Chains.MeshInboundRedirect.GetFullName()).
		Append(Match(TCP()).
			Then(Redirect(To(Ports(inboundRedirectPort)))))

	// KUMA_MESH_INBOUND
	meshInbound := NewChain(config.Chains.MeshInbound.GetFullName()).
		Append(Match(TCP()).
			Then(Redirect(To(meshInboundRedirect))))

	// KUMA_MESH_OUTBOUND_REDIRECT
	meshOutboundRedirect := NewChain(config.Chains.MeshOutboundRedirect.GetFullName()).
		Append(Match(TCP()).
			Then(Redirect(To(Ports(outboundRedirectPort)))))

	// KUMA_MESH_OUTBOUND
	meshOutbound := NewChain(config.Chains.MeshOutbound.GetFullName()).
		// when tcp_packet to 192.168.0.10:7777 arrives ⤸
		// iptables#nat
		//   PREROUTING ⤸
		//   KUMA_MESH_INBOUND ⤸
		//   KUMA_MESH_INBOUND_REDIRECT ⤸
		// envoy@15006 ⤸
		//   listener#inbound:passthrough:ipv4 ⤸
		//   cluster#inbound:passthrough:ipv4 (source_ip 127.0.0.6) ⤸
		//   listener#192.168.0.10:7777 ⤸
		//   cluster#localhost:7777 ⤸
		// localhost:7777
		Append(Match(Source(InboundPassthroughIPv4SourceAddress),
			OutInterface(loopback)).
			Then(Return)).
		Append(Match(TCP(Not(DestinationPort(DNSPort))).If(shouldRedirectDNS),
			Not(Destination(Localhost)),
			Owner(UID(uid)),
			OutInterface(loopback)).
			Then(Redirect(To(meshInboundRedirect)))).
		Append(Match(TCP(Not(DestinationPort(DNSPort))).If(shouldRedirectDNS),
			OutInterface(loopback),
			Owner(Not(UID(uid)))).
			Then(Return)).
		Append(Match(Owner(UID(uid))).
			Then(Return)).
		Append(Match(Not(Destination(Localhost)),
			OutInterface(loopback),
			Owner(GID(gid))).
			Then(Redirect(To(meshInboundRedirect)))).
		Append(Match(TCP(Not(DestinationPort(DNSPort))).If(shouldRedirectDNS),
			OutInterface(loopback),
			Owner(Not(GID(gid)))).
			Then(Return)).
		Append(Match(Owner(GID(gid))).
			Then(Return)).
		Append(Match(TCP(DestinationPort(DNSPort))).
			Then(Redirect(To(Ports(dnsRedirectPort)))).
			If(shouldRedirectDNS)).
		Append(Match(Destination(Localhost)).
			Then(Return)).
		Append(Match().
			Then(Redirect(To(meshOutboundRedirect))))

	nat.Prerouting().
		Append(Match(TCP()).
			Then(Redirect(To(meshInbound))))

	nat.Output().
		Append(Match(UDP(DestinationPort(DNSPort)), Owner(UID(uid))).
			Then(Return).
			If(shouldRedirectDNS)).
		Append(Match(UDP(DestinationPort(DNSPort)), Owner(GID(gid))).
			Then(Return).
			If(shouldRedirectDNS)).
		Append(Match(UDP(DestinationPort(DNSPort))).
			Then(Redirect(Ports(dnsRedirectPort))).
			If(shouldRedirectDNS)).
		Append(Match(TCP()).
			Then(Redirect(To(meshOutbound))))

	return nat.
		AddChain(meshInbound).
		AddChain(meshOutbound).
		AddChain(meshInboundRedirect).
		AddChain(meshOutboundRedirect).
		Build(config.Verbose), nil
}
