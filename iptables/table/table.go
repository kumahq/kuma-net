package table

import (
	"fmt"
	"strings"

	"github.com/kumahq/kuma-net/iptables/chain"
	. "github.com/kumahq/kuma-net/iptables/consts"
)

type TableBuilder struct {
	name string

	newChains []*chain.Chain
	chains    []*chain.Chain
}

// Build
// TODO (bartsmykla): refactor
// TODO (bartsmykla): add tests
func (b *TableBuilder) Build(verbose bool) string {
	tableLine := fmt.Sprintf("* %s", b.name)
	var newChainLines []string
	var ruleLines []string

	for _, c := range b.chains {
		rules := c.Build(verbose)
		ruleLines = append(ruleLines, rules...)
	}

	for _, c := range b.newChains {
		newChainLines = append(newChainLines, fmt.Sprintf("%s %s", Flags["new-chain"][verbose], c.Name()))
		rules := c.Build(verbose)
		ruleLines = append(ruleLines, rules...)
	}

	if verbose {
		if len(newChainLines) > 0 {
			newChainLines = append(
				[]string{"# Custom Chains:"},
				newChainLines...,
			)
		}

		if len(ruleLines) > 0 {
			ruleLines = append([]string{"# Rules:"}, ruleLines...)
		}
	}

	lines := []string{tableLine}

	newChains := strings.Join(newChainLines, "\n")
	if newChains != "" {
		lines = append(lines, newChains)
	}

	rules := strings.Join(ruleLines, "\n")
	if rules != "" {
		lines = append(lines, rules)
	}

	lines = append(lines, "COMMIT")

	if verbose {
		return strings.Join(lines, "\n\n")
	}

	return strings.Join(lines, "\n")
}

type NatTable struct {
	prerouting  *chain.Chain
	input       *chain.Chain
	output      *chain.Chain
	postrouting *chain.Chain

	// custom chains
	chains []*chain.Chain
}

func (t *NatTable) Prerouting() *chain.Chain {
	return t.prerouting
}

func (t *NatTable) Input() *chain.Chain {
	return t.input
}

func (t *NatTable) Output() *chain.Chain {
	return t.output
}

func (t *NatTable) Postrouting() *chain.Chain {
	return t.postrouting
}

func (t *NatTable) WithChain(chain *chain.Chain) *NatTable {
	t.chains = append(t.chains, chain)

	return t
}

func (t *NatTable) Build(verbose bool) string {
	table := &TableBuilder{
		name:      "nat",
		newChains: t.chains,
		chains: []*chain.Chain{
			t.prerouting,
			t.input,
			t.output,
			t.postrouting,
		},
	}

	return table.Build(verbose)
}

func Nat() *NatTable {
	return &NatTable{
		prerouting:  chain.NewChain("PREROUTING"),
		input:       chain.NewChain("INPUT"),
		output:      chain.NewChain("OUTPUT"),
		postrouting: chain.NewChain("POSTROUTING"),
	}
}
