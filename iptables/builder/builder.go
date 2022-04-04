package builder

import (
	"fmt"
	"strings"

	"github.com/kumahq/kuma-net/iptables/config"
	"github.com/kumahq/kuma-net/iptables/table"
)

type IPTables struct {
	raw *table.RawTable
	nat *table.NatTable
}

func newIPTables(raw *table.RawTable, nat *table.NatTable) *IPTables {
	return &IPTables{
		raw: raw,
		nat: nat,
	}
}

func (t *IPTables) Build(verbose bool) string {
	var tables []string

	raw := t.raw.Build(verbose)
	if raw != "" {
		tables = append(tables, raw)
	}

	nat := t.nat.Build(verbose)
	if nat != "" {
		tables = append(tables, nat)
	}

	if verbose {
		return strings.Join(tables, "\n\n")
	}

	return strings.Join(tables, "\n")
}

func BuildIPTables(config *config.Config) (string, error) {
	loopbackIface, err := getLoopback()
	if err != nil {
		return "", fmt.Errorf("cannot obtain loopback interface: %s", err)
	}

	return newIPTables(
		buildRawTable(config),
		buildNatTable(config, loopbackIface.Name),
	).Build(config.Verbose), nil
}
