package chain

import (
	"github.com/kumahq/kuma-net/iptables/builder/rule"
)

// TODO (bartsmykla): add validation for built-in chains as they are predefined, and cannot
//  be changed

type ChainBuilder struct {
	name  string
	rules []rule.RuleBuilder
}

func (b *ChainBuilder) String() string {
	return b.name
}

func (b *ChainBuilder) Append(ruleBuilder rule.RuleBuilder) *ChainBuilder {
	newRule := ruleBuilder.Kind(rule.Append).Chain(b.name)

	b.rules = append(b.rules, newRule)

	return b
}

func (b *ChainBuilder) Build(verbose bool) []string {
	var rules []string

	for _, r := range b.rules {
		if builtRule := r.Build(verbose); builtRule != "" {
			rules = append(rules, builtRule)
		}
	}

	return rules
}

func NewChain(name string) *ChainBuilder {
	return &ChainBuilder{
		name: name,
	}
}
