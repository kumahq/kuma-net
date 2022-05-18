package builder

import (
	"github.com/kumahq/kuma-net/iptables/config"
	. "github.com/kumahq/kuma-net/iptables/consts"
	. "github.com/kumahq/kuma-net/iptables/parameters"
	"github.com/kumahq/kuma-net/iptables/table"
)

func buildRawTable(cfg config.Config) *table.RawTable {
	raw := table.Raw()

	if cfg.Redirect.DNS.Enabled && cfg.Redirect.DNS.ConntrackZoneSplit {
		raw.Output().
			Append(
				Protocol(Udp(DestinationPort(DNSPort))),
				Match(Owner(Uid(cfg.Owner.UID))),
				Jump(Ct(Zone("1"))),
			).
			Append(
				Protocol(Udp(SourcePort(cfg.Redirect.DNS.Port))),
				Match(Owner(Uid(cfg.Owner.UID))),
				Jump(Ct(Zone("2"))),
			).
			Append(
				Protocol(Udp(DestinationPort(DNSPort))),
				Jump(Ct(Zone("2"))),
			)

		raw.Prerouting().
			Append(
				Protocol(Udp(SourcePort(DNSPort))),
				Jump(Ct(Zone("1"))),
			)
	}

	return raw
}
