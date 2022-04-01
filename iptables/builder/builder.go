package builder

import (
	"fmt"
	"net"

	. "github.com/kumahq/kuma-net/iptables/chain"
	"github.com/kumahq/kuma-net/iptables/config"
	. "github.com/kumahq/kuma-net/iptables/consts"
	. "github.com/kumahq/kuma-net/iptables/parameters"
	"github.com/kumahq/kuma-net/iptables/table"
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

func buildMeshInbound(cfg *config.TrafficFlow, meshInboundRedirect string) *Chain {
	meshInbound := NewChain(cfg.Chain.GetFullName())

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

func buildMeshOutbound(
	cfg *config.Config,
	loopback string,
	meshInboundRedirect string,
	meshOutboundRedirect string,
) *Chain {
	outboundChainName := cfg.Redirect.Outbound.Chain.GetFullName()
	excludePorts := cfg.Redirect.Outbound.ExcludePorts
	shouldRedirectDNS := cfg.Redirect.DNS.Enabled
	dnsRedirectPort := cfg.Redirect.DNS.Port
	uid := cfg.Owner.UID
	gid := cfg.Owner.GID

	meshOutbound := NewChain(outboundChainName).
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
		Append(
			Source(Address(InboundPassthroughIPv4SourceAddress)),
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
		// UID
		Append(
			Protocol(Tcp(NotDestinationPortIf(shouldRedirectDNS, DNSPort))),
			OutInterface(loopback),
			NotDestination(Localhost),
			Match(Owner(Uid(uid))),
			Jump(ToUserDefinedChain(meshInboundRedirect)),
		).
		Append(
			Protocol(Tcp(NotDestinationPortIf(shouldRedirectDNS, DNSPort))),
			OutInterface(loopback),
			Match(Owner(NotUid(uid))),
			Jump(Return()),
		).
		Append(
			Match(Owner(Uid(uid))),
			Jump(Return()),
		).
		// GID
		Append(
			Protocol(Tcp(NotDestinationPortIf(shouldRedirectDNS, DNSPort))),
			OutInterface(loopback),
			NotDestination(Localhost),
			Match(Owner(Gid(gid))),
			Jump(ToUserDefinedChain(meshInboundRedirect)),
		).
		Append(
			Protocol(Tcp(NotDestinationPortIf(shouldRedirectDNS, DNSPort))),
			OutInterface(loopback),
			Match(Owner(NotGid(gid))),
			Jump(Return()),
		).
		Append(
			Match(Owner(Gid(gid))),
			Jump(Return()),
		).
		AppendIf(shouldRedirectDNS,
			Protocol(Tcp(DestinationPort(DNSPort))),
			Jump(ToPort(dnsRedirectPort)),
		).
		Append(
			Destination(Localhost),
			Jump(Return()),
		).
		Append(
			Jump(ToUserDefinedChain(meshOutboundRedirect)),
		)

	return meshOutbound
}

func Build(config *config.Config) (string, error) {
	loopbackIface, err := getLoopback()
	if err != nil {
		return "", fmt.Errorf("cannot obtain loopback interface: %s", err)
	}

	loopback := loopbackIface.Name

	inboundRedirectChainName := config.Redirect.Inbound.RedirectChain.GetFullName()
	inboundChainName := config.Redirect.Inbound.Chain.GetFullName()
	outboundChainName := config.Redirect.Outbound.Chain.GetFullName()
	outboundRedirectChainName := config.Redirect.Outbound.RedirectChain.GetFullName()
	inboundRedirectPort := config.Redirect.Inbound.Port
	outboundRedirectPort := config.Redirect.Outbound.Port
	dnsRedirectPort := config.Redirect.DNS.Port
	uid := config.Owner.UID
	gid := config.Owner.GID
	shouldRedirectDNS := config.Redirect.DNS.Enabled

	nat := table.Nat()

	nat.Prerouting().Append(
		Protocol(Tcp()),
		Jump(ToUserDefinedChain(inboundChainName)),
	)

	nat.Output().
		AppendIf(shouldRedirectDNS,
			Protocol(Udp(DestinationPort(DNSPort))),
			Match(Owner(Uid(uid))),
			Jump(Return()),
		).
		AppendIf(shouldRedirectDNS,
			Protocol(Udp(DestinationPort(DNSPort))),
			Match(Owner(Gid(gid))),
			Jump(Return()),
		).
		AppendIf(shouldRedirectDNS,
			Protocol(Udp(DestinationPort(DNSPort))),
			Jump(ToPort(dnsRedirectPort)),
		).
		Append(
			Protocol(Tcp()),
			Jump(ToUserDefinedChain(outboundChainName)),
		)

	// MESH_INBOUND
	meshInbound := buildMeshInbound(config.Redirect.Inbound, inboundRedirectChainName)

	// MESH_INBOUND_REDIRECT
	meshInboundRedirect := NewChain(inboundRedirectChainName).
		Append(
			Protocol(Tcp()),
			Jump(ToPort(inboundRedirectPort)),
		)

	// MESH_OUTBOUND
	meshOutbound := buildMeshOutbound(
		config,
		loopback,
		inboundRedirectChainName,
		outboundRedirectChainName,
	)

	// MESH_OUTBOUND_REDIRECT
	meshOutboundRedirect := NewChain(outboundRedirectChainName).
		Append(
			Protocol(Tcp()),
			Jump(ToPort(outboundRedirectPort)),
		)

	return nat.
		WithChain(meshInbound).
		WithChain(meshOutbound).
		WithChain(meshInboundRedirect).
		WithChain(meshOutboundRedirect).
		Build(config.Verbose), nil
}
