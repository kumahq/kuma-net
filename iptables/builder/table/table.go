package table

import (
	"fmt"
	"strings"

	"github.com/kumahq/kuma-net/iptables/builder/chain"
)

type TableBuilder struct {
	name string

	newChains []*chain.ChainBuilder
	chains    []*chain.ChainBuilder
}

func Table(name string) *TableBuilder {
	return &TableBuilder{
		name: name,
	}
}

// Build
// TODO (bartsmykla): refactor
// TODO (bartsmykla): add tests
func (b *TableBuilder) Build(verbose bool) string {
	tableLine := fmt.Sprintf("*%s", b.name)
	var newChainLines []string
	var newChainDefaultPolicyLines []string
	var defaultPolicyLines []string
	var ruleLines []string

	for _, c := range b.chains {
		defaultPolicy, rules := c.Build(verbose)
		defaultPolicyLines = append(defaultPolicyLines, defaultPolicy)
		ruleLines = append(ruleLines, rules...)
	}

	for _, c := range b.newChains {
		newChainLines = append(newChainLines, fmt.Sprintf("-N %s", c.String()))
		defaultPolicy, rules := c.Build(verbose)
		newChainDefaultPolicyLines = append(newChainDefaultPolicyLines, defaultPolicy)
		ruleLines = append(ruleLines, rules...)
	}

	if verbose {
		if len(defaultPolicyLines) > 0 {
			defaultPolicyLines = append(
				[]string{"# Builtin Chains Default Policies:"},
				defaultPolicyLines...,
			)
		}

		if len(newChainLines) > 0 {
			newChainLines = append(
				[]string{"# Custom Chains:"},
				newChainLines...,
			)
		}

		if len(newChainDefaultPolicyLines) > 0 {
			newChainDefaultPolicyLines = append(
				[]string{"# Custom Chains Default Policies:"},
				newChainDefaultPolicyLines...,
			)
		}

		if len(ruleLines) > 0 {
			ruleLines = append([]string{"# Rules:"}, ruleLines...)
		}
	}

	lines := []string{tableLine}

	defaultPolicies := strings.Join(defaultPolicyLines, "\n")
	if defaultPolicies != "" {
		lines = append(lines, defaultPolicies)
	}

	newChains := strings.Join(newChainLines, "\n")
	if newChains != "" {
		lines = append(lines, newChains)
	}

	newChainsDefaultPolicies := strings.Join(newChainDefaultPolicyLines, "\n")
	if newChainsDefaultPolicies != "" {
		lines = append(lines, newChainsDefaultPolicies)
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
	prerouting  *chain.ChainBuilder
	input       *chain.ChainBuilder
	output      *chain.ChainBuilder
	postrouting *chain.ChainBuilder

	// custom chains
	chains []*chain.ChainBuilder
}

func (t *NatTable) Prerouting() *chain.ChainBuilder {
	return t.prerouting
}

func (t *NatTable) Input() *chain.ChainBuilder {
	return t.input
}

func (t *NatTable) Output() *chain.ChainBuilder {
	return t.output
}

func (t *NatTable) Postrouting() *chain.ChainBuilder {
	return t.postrouting
}

func (t *NatTable) Chain(chain *chain.ChainBuilder) *NatTable {
	t.chains = append(t.chains, chain)

	return t
}

func (t *NatTable) Build(verbose bool) string {
	table := &TableBuilder{
		name:      "nat",
		newChains: t.chains,
		chains: []*chain.ChainBuilder{
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
		prerouting:  chain.Prerouting(),
		input:       chain.Input(),
		output:      chain.Output(),
		postrouting: chain.Postrouting(),
	}
}
