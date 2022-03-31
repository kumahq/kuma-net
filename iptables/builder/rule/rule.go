package rule

import (
	"strings"

	"github.com/kumahq/kuma-net/iptables/builder/target"
	. "github.com/kumahq/kuma-net/iptables/consts"
	"github.com/kumahq/kuma-net/iptables/parameters"
)

type Kind uint

func (k Kind) String() string {
	switch k {
	case Append:
		return "append"
	default:
		panic("unsupported rule")
	}
}

const (
	Append Kind = iota
)

func (k Kind) Build(verbose bool) string {
	return Flags[k.String()][verbose]
}

type RuleBuilder struct {
	kind   Kind
	chain  string
	match  []parameters.ParameterBuilder
	target target.Func
}

func (b RuleBuilder) Build(verbose bool) string {
	// TODO (bartsmykla): definitely refactor
	if b.kind == Append && b.target == nil {
		return ""
	}

	rule := []string{b.kind.Build(verbose)}

	if b.chain != "" {
		rule = append(rule, b.chain)
	}

	if len(b.match) > 0 {
		for _, option := range b.match {
			if opt := option.Build(verbose); opt != "" {
				rule = append(rule, opt)
			}
		}
	}

	if b.target != nil {
		rule = append(rule, Flags["jump"][verbose])
		rule = append(rule, b.target()...)
	}

	return strings.Join(rule, " ")
}

func (b RuleBuilder) Then(target target.Func) RuleBuilder {
	return RuleBuilder{
		kind:   b.kind,
		chain:  b.chain,
		match:  b.match,
		target: target,
	}
}

func (b RuleBuilder) If(predicates ...parameters.Predicate) RuleBuilder {
	for _, predicate := range predicates {
		if !predicate() {
			return RuleBuilder{}
		}
	}

	return b
}

func (b RuleBuilder) Kind(kind Kind) RuleBuilder {
	return RuleBuilder{
		kind:   kind,
		chain:  b.chain,
		match:  b.match,
		target: b.target,
	}
}

func (b RuleBuilder) Chain(chain string) RuleBuilder {
	return RuleBuilder{
		kind:   b.kind,
		chain:  chain,
		match:  b.match,
		target: b.target,
	}
}

func Match(match ...parameters.ParameterBuilder) RuleBuilder {
	return RuleBuilder{match: match}
}
