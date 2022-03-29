package builder

import (
	. "github.com/kumahq/kuma-net/iptables/builder/chain"
	"github.com/kumahq/kuma-net/iptables/builder/config"
	. "github.com/kumahq/kuma-net/iptables/builder/rule"
	"github.com/kumahq/kuma-net/iptables/builder/table"
	. "github.com/kumahq/kuma-net/iptables/builder/target"
	. "github.com/kumahq/kuma-net/iptables/parameters"
)

// TODO (bartsmykla): move it to some better suited place (consts.go maybe?)
const (
	DNSPort   uint16 = 53
	Localhost        = "127.0.0.1/32"
	Loopback         = "lo"
	// InboundPassthroughIPv4SourceAddress
	// TODO (bartsmykla): add some description
	InboundPassthroughIPv4SourceAddress = "127.0.0.6/32"
)

type IPTables struct {
	tables []*table.TableBuilder
}

//goland:noinspection GoSnakeCaseUsage
func Build(config *config.Config) string {
	inbound := config.Tables.Nat.MeshInbound.Name
	outbound := config.Tables.Nat.MeshOutbound.Name
	inboundRedirect := config.Tables.Nat.MeshInboundRedirect.Name
	outboundRedirect := config.Tables.Nat.MeshOutboundRedirect.Name

	inboundRedirectPort := config.Redirect.Inbound.Port
	outboundRedirectPort := config.Redirect.Outbound.Port
	dnsRedirectPort := config.Redirect.DNS.Port
	uid := config.Owner.UID
	gid := config.Owner.GID

	shouldRedirectInbound := config.Redirect.Inbound.Enabled
	shouldRedirectOutbound := config.Redirect.Outbound.Enabled
	shouldRedirectDNS := config.Redirect.DNS.Enabled

	nat := table.Nat()

	MESH_INBOUND_REDIRECT := NewChain(inboundRedirect).
		Append(Match(TCP()).Then(Redirect(To(Ports(inboundRedirectPort))))).
		If(shouldRedirectInbound)

	MESH_INBOUND := NewChain(inbound).
		Append(Match(TCP()).Then(Redirect(To(MESH_INBOUND_REDIRECT)))).
		If(shouldRedirectInbound)

	MESH_OUTBOUND_REDIRECT := NewChain(outboundRedirect).
		Append(Match(TCP()).
			Then(Redirect(To(Ports(outboundRedirectPort))))).
		If(shouldRedirectOutbound)

	MESH_OUTBOUND := NewChain(outbound).
		// when tcp_packet to 192.168.0.10:7777 arrives ⤸
		// iptables#nat
		//   PREROUTING ⤸
		//   MESH_INBOUND ⤸
		//   MESH_INBOUND_REDIRECT ⤸
		// envoy@15006 ⤸
		//   listener#inbound:passthrough:ipv4 ⤸
		//   cluster#inbound:passthrough:ipv4 (source_ip 127.0.0.6) ⤸
		//   listener#192.168.0.10:7777 ⤸
		//   cluster#localhost:7777 ⤸
		// localhost:7777
		Append(Match(Source(InboundPassthroughIPv4SourceAddress),
			OutInterface(Loopback)).
			Then(Return)).
		Append(Match(TCP(Not(DestinationPort(DNSPort))).If(shouldRedirectDNS),
			Not(Destination(Localhost)),
			Owner(UID(uid)),
			OutInterface(Loopback)).
			Then(Redirect(To(MESH_INBOUND_REDIRECT))).
			If(shouldRedirectInbound)).
		Append(Match(TCP(Not(DestinationPort(DNSPort))).If(shouldRedirectDNS),
			OutInterface(Loopback),
			Owner(Not(UID(uid)))).
			Then(Return)).
		Append(Match(Owner(UID(uid))).
			Then(Return)).
		Append(Match(Not(Destination(Localhost)),
			OutInterface(Loopback),
			Owner(GID(gid))).
			Then(Redirect(To(MESH_INBOUND_REDIRECT))).
			If(shouldRedirectInbound)).
		Append(Match(TCP(Not(DestinationPort(DNSPort))).If(shouldRedirectDNS),
			OutInterface(Loopback),
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
			Then(Redirect(To(MESH_OUTBOUND_REDIRECT))).
			If(shouldRedirectOutbound))

	nat.Prerouting().
		Append(Match(TCP()).Then(Redirect(To(MESH_INBOUND)))).
		If(shouldRedirectInbound)

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
			Then(Redirect(To(MESH_OUTBOUND))).
			If(shouldRedirectOutbound))

	// TODO (bartsmykla): currently it assumes that outbound redirection is enabled,
	//  but maybe we should allow to redirect DNS traffic even if we don't want to
	//  redirect any other traffic?

	// TODO (bartsmykla): some parameters (--source for example) should result in
	//  multiple rules when provided more addresses instead of doing it in one
	//  make sure to handle it properly

	return nat.
		Chain(MESH_INBOUND).
		Chain(MESH_OUTBOUND).
		Chain(MESH_INBOUND_REDIRECT).
		Chain(MESH_OUTBOUND_REDIRECT).
		Build(config.Verbose)
}
