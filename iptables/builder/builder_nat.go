package builder

import (
	. "github.com/kumahq/kuma-net/iptables/chain"
	"github.com/kumahq/kuma-net/iptables/config"
	. "github.com/kumahq/kuma-net/iptables/consts"
	. "github.com/kumahq/kuma-net/iptables/parameters"
	"github.com/kumahq/kuma-net/iptables/table"
)

func buildMeshInbound(cfg config.TrafficFlow, prefix string, meshInboundRedirect string) *Chain {
	meshInbound := NewChain(cfg.Chain.GetFullName(prefix))

	// Excluded inbound ports
	for _, port := range cfg.ExcludePorts {
		meshInbound.Append(
			Protocol(Tcp(DestinationPort(port))),
			Jump(Return()),
		)
	}

	meshInbound.Append(
		Protocol(Tcp()),
		Jump(ToUserDefinedChain(meshInboundRedirect)),
	)

	return meshInbound
}

func buildMeshOutbound(cfg config.Config, loopback string, ipv6 bool) *Chain {
	prefix := cfg.Redirect.NamePrefix
	inboundRedirectChainName := cfg.Redirect.Inbound.RedirectChain.GetFullName(prefix)
	outboundChainName := cfg.Redirect.Outbound.Chain.GetFullName(prefix)
	outboundRedirectChainName := cfg.Redirect.Outbound.RedirectChain.GetFullName(prefix)
	excludePorts := cfg.Redirect.Outbound.ExcludePorts
	dnsRedirectPort := cfg.Redirect.DNS.Port
	uid := cfg.Owner.UID

	localhost := LocalhostCIDRIPv4
	inboundPassthroughSourceAddress := InboundPassthroughSourceAddressCIDRIPv4
	if ipv6 {
		inboundPassthroughSourceAddress = InboundPassthroughSourceAddressCIDRIPv6
		localhost = LocalhostCIDRIPv6
	}

	meshOutbound := NewChain(outboundChainName).
		// ipv4:
		//   when tcp_packet to 192.168.0.10:7777 arrives ⤸
		//   iptables#nat ⤸
		//     PREROUTING ⤸
		//     MESH_INBOUND ⤸
		//     MESH_INBOUND_REDIRECT ⤸
		//   envoy@15006 ⤸
		//     listener#inbound:passthrough:ipv4 ⤸
		//     cluster#inbound:passthrough:ipv4 (source_ip 127.0.0.6) ⤸
		//     listener#192.168.0.10:7777 ⤸
		//     cluster#localhost:7777 ⤸
		//   localhost:7777
		//
		// ipv6:
		//   when tcp_packet to [fd00::0:10]:7777 arrives ⤸
		//   ip6tables#nat ⤸
		//     PREROUTING ⤸
		//     MESH_INBOUND ⤸
		//     MESH_INBOUND_REDIRECT ⤸
		//   envoy@15006 ⤸
		//     listener#inbound:passthrough:ipv6 ⤸
		//     cluster#inbound:passthrough:ipv6 (source_ip ::6) ⤸
		//     listener#[fd00::0:10]:7777 ⤸
		//     cluster#localhost:7777 ⤸
		//   localhost:7777
		Append(
			Source(Address(inboundPassthroughSourceAddress)),
			OutInterface(loopback),
			Jump(Return()),
		)

	// Excluded outbound ports
	for _, port := range excludePorts {
		meshOutbound.Append(
			Protocol(Tcp(DestinationPort(port))),
			Jump(Return()),
		)
	}

	meshOutbound.
		Append(
			Protocol(Tcp(NotDestinationPortIf(cfg.ShouldRedirectDNS, DNSPort))),
			OutInterface(loopback),
			NotDestination(localhost),
			Match(Owner(Uid(uid))),
			Jump(ToUserDefinedChain(inboundRedirectChainName)),
		).
		Append(
			Protocol(Tcp(NotDestinationPortIf(cfg.ShouldRedirectDNS, DNSPort))),
			OutInterface(loopback),
			Match(Owner(NotUid(uid))),
			Jump(Return()),
		).
		Append(
			Match(Owner(Uid(uid))),
			Jump(Return()),
		).
		AppendIf(cfg.ShouldRedirectDNS,
			Protocol(Tcp(DestinationPort(DNSPort))),
			Jump(ToPort(dnsRedirectPort)),
		).
		Append(
			Destination(localhost),
			Jump(Return()),
		).
		Append(
			Jump(ToUserDefinedChain(outboundRedirectChainName)),
		)

	return meshOutbound
}

func buildMeshRedirect(cfg config.TrafficFlow, prefix string, ipv6 bool) *Chain {
	chainName := cfg.RedirectChain.GetFullName(prefix)

	redirectPort := cfg.Port
	if ipv6 {
		redirectPort = cfg.PortIPv6
	}

	return NewChain(chainName).
		Append(
			Protocol(Tcp()),
			Jump(ToPort(redirectPort)),
		)
}

func buildNatTable(cfg config.Config, loopback string, ipv6 bool) *table.NatTable {
	prefix := cfg.Redirect.NamePrefix
	inboundRedirectChainName := cfg.Redirect.Inbound.RedirectChain.GetFullName(prefix)
	inboundChainName := cfg.Redirect.Inbound.Chain.GetFullName(prefix)
	outboundChainName := cfg.Redirect.Outbound.Chain.GetFullName(prefix)
	dnsRedirectPort := cfg.Redirect.DNS.Port
	uid := cfg.Owner.UID

	nat := table.Nat()

	nat.Prerouting().Append(
		Protocol(Tcp()),
		Jump(ToUserDefinedChain(inboundChainName)),
	)

	nat.Output().
		AppendIf(cfg.ShouldRedirectDNS,
			Protocol(Udp(DestinationPort(DNSPort))),
			Match(Owner(Uid(uid))),
			Jump(Return()),
		).
		AppendIf(cfg.ShouldRedirectDNS,
			Protocol(Udp(DestinationPort(DNSPort))),
			Jump(ToPort(dnsRedirectPort)),
		).
		Append(
			Protocol(Tcp()),
			Jump(ToUserDefinedChain(outboundChainName)),
		)

	// MESH_INBOUND
	meshInbound := buildMeshInbound(cfg.Redirect.Inbound, prefix, inboundRedirectChainName)

	// MESH_INBOUND_REDIRECT
	meshInboundRedirect := buildMeshRedirect(cfg.Redirect.Inbound, prefix, ipv6)

	// MESH_OUTBOUND
	meshOutbound := buildMeshOutbound(cfg, loopback, ipv6)

	// MESH_OUTBOUND_REDIRECT
	meshOutboundRedirect := buildMeshRedirect(cfg.Redirect.Outbound, prefix, ipv6)

	return nat.
		WithChain(meshInbound).
		WithChain(meshOutbound).
		WithChain(meshInboundRedirect).
		WithChain(meshOutboundRedirect)
}
