package builder

import (
	"fmt"

	. "github.com/kumahq/kuma-net/iptables/chain"
	. "github.com/kumahq/kuma-net/iptables/consts"
	. "github.com/kumahq/kuma-net/iptables/parameters"
	"github.com/kumahq/kuma-net/iptables/table"
	"github.com/kumahq/kuma-net/transparent-proxy/config"
)

func buildMeshInbound(cfg config.TrafficFlow, prefix string, meshInboundRedirect string) *Chain {
	meshInbound := NewChain(cfg.Chain.GetFullName(prefix))
	if !cfg.Enabled {
		meshInbound.Append(
			Protocol(Tcp()),
			Jump(Return()),
		)
		return meshInbound
	}

	// Include inbound ports
	for _, port := range cfg.IncludePorts {
		meshInbound.Append(
			Protocol(Tcp(DestinationPort(port))),
			Jump(ToUserDefinedChain(meshInboundRedirect)),
		)
	}

	if len(cfg.IncludePorts) == 0 {
		// Excluded outbound ports
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
	}

	return meshInbound
}

func buildMeshOutbound(
	cfg config.Config,
	dnsServers []string,
	loopback string,
	ipv6 bool,
) *Chain {
	prefix := cfg.Redirect.NamePrefix
	inboundRedirectChainName := cfg.Redirect.Inbound.RedirectChain.GetFullName(prefix)
	outboundChainName := cfg.Redirect.Outbound.Chain.GetFullName(prefix)
	outboundRedirectChainName := cfg.Redirect.Outbound.RedirectChain.GetFullName(prefix)
	excludePorts := cfg.Redirect.Outbound.ExcludePorts
	includePorts := cfg.Redirect.Outbound.IncludePorts
	hasIncludedPorts := len(includePorts) > 0
	dnsRedirectPort := cfg.Redirect.DNS.Port
	uid := cfg.Owner.UID

	localhost := LocalhostCIDRIPv4
	inboundPassthroughSourceAddress := InboundPassthroughSourceAddressCIDRIPv4
	if ipv6 {
		inboundPassthroughSourceAddress = InboundPassthroughSourceAddressCIDRIPv6
		localhost = LocalhostCIDRIPv6
	}

	meshOutbound := NewChain(outboundChainName)
	if !cfg.Redirect.Outbound.Enabled {
		meshOutbound.Append(
			Protocol(Tcp()),
			Jump(Return()),
		)
		return meshOutbound
	}

	// Excluded outbound ports
	if !hasIncludedPorts {
		for _, port := range excludePorts {
			meshOutbound.Append(
				Protocol(Tcp(DestinationPort(port))),
				Jump(Return()),
			)
		}
	}
	meshOutbound.
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
		).
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
		)
	if cfg.ShouldRedirectDNS() {
		if cfg.ShouldCaptureAllDNS() {
			meshOutbound.Append(
				Protocol(Tcp(DestinationPort(DNSPort))),
				Jump(ToPort(dnsRedirectPort)),
			)
		} else {
			for _, dnsIp := range dnsServers {
				meshOutbound.Append(
					Destination(dnsIp),
					Protocol(Tcp(DestinationPort(DNSPort))),
					Jump(ToPort(dnsRedirectPort)),
				)
			}
		}
	}
	meshOutbound.
		Append(
			Destination(localhost),
			Jump(Return()),
		)

	if hasIncludedPorts {
		for _, port := range includePorts {
			meshOutbound.Append(
				Protocol(Tcp(DestinationPort(port))),
				Jump(ToUserDefinedChain(outboundRedirectChainName)),
			)
		}
	} else {
		meshOutbound.Append(
			Jump(ToUserDefinedChain(outboundRedirectChainName)),
		)
	}

	return meshOutbound
}

func buildMeshRedirect(cfg config.TrafficFlow, prefix string, ipv6 bool) *Chain {
	chainName := cfg.RedirectChain.GetFullName(prefix)

	redirectPort := cfg.Port
	if ipv6 && cfg.PortIPv6 != 0 {
		redirectPort = cfg.PortIPv6
	}

	return NewChain(chainName).
		Append(
			Protocol(Tcp()),
			Jump(ToPort(redirectPort)),
		)
}

func addOutputRules(cfg config.Config, dnsServers []string, nat *table.NatTable) error {
	outboundChainName := cfg.Redirect.Outbound.Chain.GetFullName(cfg.Redirect.NamePrefix)
	dnsRedirectPort := cfg.Redirect.DNS.Port
	uid := cfg.Owner.UID
	if cfg.Log.Enabled {
		nat.Output().Append(
			Jump(Log(OutputLogPrefix, cfg.Log.Level)),
		)
	}

	// Excluded outbound ports for UIDs
	for _, uIDsToPorts := range cfg.Redirect.Outbound.ExcludePortsForUIDs {
		var protocol *Parameter

		switch uIDsToPorts.Protocol {
		case TCP:
			protocol = Protocol(Tcp(DestinationPortRangeOrValue(uIDsToPorts)))
		case UDP:
			protocol = Protocol(Udp(DestinationPortRangeOrValue(uIDsToPorts)))
		default:
			return fmt.Errorf("unknown protocol %s, only 'tcp' or 'udp' allowed", uIDsToPorts.Protocol)
		}

		nat.Output().Append(
			protocol,
			Match(Owner(UidRangeOrValue(uIDsToPorts))),
			Jump(Return()),
		)
	}

	if cfg.ShouldRedirectDNS() {
		nat.Output().Append(
			Protocol(Udp(DestinationPort(DNSPort))),
			Match(Owner(Uid(uid))),
			Jump(Return()),
		)
		if cfg.ShouldCaptureAllDNS() {
			nat.Output().Append(
				Protocol(Udp(DestinationPort(DNSPort))),
				Jump(ToPort(dnsRedirectPort)),
			)
		} else {
			for _, dnsIp := range dnsServers {
				nat.Output().Append(
					Destination(dnsIp),
					Protocol(Udp(DestinationPort(DNSPort))),
					Jump(ToPort(dnsRedirectPort)),
				)
			}
		}
	}
	nat.Output().
		Append(
			Protocol(Tcp()),
			Jump(ToUserDefinedChain(outboundChainName)),
		)
	return nil
}

func buildNatTable(
	cfg config.Config,
	dnsServers []string,
	loopback string,
	ipv6 bool,
) (*table.NatTable, error) {
	prefix := cfg.Redirect.NamePrefix
	inboundRedirectChainName := cfg.Redirect.Inbound.RedirectChain.GetFullName(prefix)
	inboundChainName := cfg.Redirect.Inbound.Chain.GetFullName(prefix)
	nat := table.Nat()

	if cfg.Log.Enabled {
		nat.Prerouting().Append(
			Jump(Log(PreroutingLogPrefix, cfg.Log.Level)),
		)
	}
	nat.Prerouting().Append(
		Protocol(Tcp()),
		Jump(ToUserDefinedChain(inboundChainName)),
	)

	if err := addOutputRules(cfg, dnsServers, nat); err != nil {
		return nil, fmt.Errorf("could not add output rules %s", err)
	}

	// MESH_INBOUND
	meshInbound := buildMeshInbound(cfg.Redirect.Inbound, prefix, inboundRedirectChainName)

	// MESH_INBOUND_REDIRECT
	meshInboundRedirect := buildMeshRedirect(cfg.Redirect.Inbound, prefix, ipv6)

	// MESH_OUTBOUND
	meshOutbound := buildMeshOutbound(cfg, dnsServers, loopback, ipv6)

	// MESH_OUTBOUND_REDIRECT
	meshOutboundRedirect := buildMeshRedirect(cfg.Redirect.Outbound, prefix, ipv6)

	return nat.
		WithChain(meshInbound).
		WithChain(meshOutbound).
		WithChain(meshInboundRedirect).
		WithChain(meshOutboundRedirect), nil
}
