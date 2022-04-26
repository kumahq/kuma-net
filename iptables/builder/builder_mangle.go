package builder

import (
	"github.com/kumahq/kuma-net/iptables/config"
	. "github.com/kumahq/kuma-net/iptables/parameters"
	. "github.com/kumahq/kuma-net/iptables/parameters/match/conntrack"
	"github.com/kumahq/kuma-net/iptables/table"
)

func buildMangleTable(cfg config.Config) *table.MangleTable {
	mangle := table.Mangle()

	mangle.Prerouting().
		AppendIf(cfg.ShouldDropInvalidPackets,
			Match(Conntrack(Ctstate(INVALID))),
			Jump(Drop()),
		)

	return mangle
}
