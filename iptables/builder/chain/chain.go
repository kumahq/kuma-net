package chain

import (
	"fmt"

	"github.com/kumahq/kuma-net/iptables/builder/policy"
	"github.com/kumahq/kuma-net/iptables/builder/rule"
	"github.com/kumahq/kuma-net/iptables/parameters"
)

// TODO (bartsmykla): add validation for built-in chains as they are predefined, and cannot
//  be changed

type ChainBuilder struct {
	name          string
	rules         []rule.RuleBuilder
	defaultPolicy *policy.DefaultPolicy
}

func (b *ChainBuilder) String() string {
	return b.name
}

func (b *ChainBuilder) Append(ruleBuilder rule.RuleBuilder) *ChainBuilder {
	newRule := ruleBuilder.Kind(rule.Append).Chain(b.name)

	b.rules = append(b.rules, newRule)

	return b
}

func (b *ChainBuilder) DefaultPolicy(policy *policy.DefaultPolicy) *ChainBuilder {
	b.defaultPolicy = policy

	return b
}

func (b *ChainBuilder) If(predicates ...parameters.Predicate) *ChainBuilder {
	for _, predicate := range predicates {
		if !predicate() {
			return nil
		}
	}

	return b
}

func (b *ChainBuilder) Build(verbose bool) (string, []string) {
	defaultPolicy := fmt.Sprintf(":%s %s", b.name, b.defaultPolicy)

	var rules []string
	for _, r := range b.rules {
		if builtRule := r.Build(verbose); builtRule != "" {
			rules = append(rules, builtRule)
		}
	}

	return defaultPolicy, rules
}

func Prerouting() *ChainBuilder {
	return &ChainBuilder{
		name:          "PREROUTING",
		defaultPolicy: policy.Accept(),
	}
}

func Input() *ChainBuilder {
	return &ChainBuilder{
		name:          "INPUT",
		defaultPolicy: policy.Accept(),
	}
}

func Output() *ChainBuilder {
	return &ChainBuilder{
		name:          "OUTPUT",
		defaultPolicy: policy.Accept(),
	}
}

func Postrouting() *ChainBuilder {
	return &ChainBuilder{
		name:          "POSTROUTING",
		defaultPolicy: policy.Accept(),
	}
}

func NewChain(name string) *ChainBuilder {
	return &ChainBuilder{
		name:          name,
		defaultPolicy: policy.None(),
	}
}
