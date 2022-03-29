package parameters

import (
	"fmt"
	"strings"
)

type ParameterBuilder interface {
	Build(verbose bool) string
}

type Predicate func() bool

type Parameter[T any] struct {
	flag     string
	short    string
	value    string
	negative bool
}

type Parameters[T any] []*Parameter[T]

func newParameters[T any](long, short, value string) Parameters[T] {
	return Parameters[T]{&Parameter[T]{
		flag:  long,
		short: short,
		value: value,
	}}
}

func (p Parameters[T]) append(long, short, value string, negative bool) Parameters[T] {
	return append(p, &Parameter[T]{
		flag:     long,
		short:    short,
		value:    value,
		negative: negative,
	})
}

func (p Parameters[T]) concat(parameters ...Parameters[T]) Parameters[T] {
	result := p

	for _, params := range parameters {
		result = append(p, params...)
	}

	return result
}

func (p Parameters[T]) Build(verbose bool) string {
	var parameters []string

	for _, parameter := range p {
		flag := parameter.short

		if verbose || parameter.short == "" {
			flag = parameter.flag
		}

		param := fmt.Sprintf("%s %s", flag, parameter.value)

		if parameter.negative {
			param = fmt.Sprintf("! %s", param)
		}

		parameters = append(parameters, param)
	}

	return strings.Join(parameters, " ")
}

func (p Parameters[T]) Negate() Parameters[T] {
	var negated Parameters[T]

	for _, parameter := range p {
		negated = append(negated, &Parameter[T]{
			flag:     parameter.flag,
			short:    parameter.short,
			value:    parameter.value,
			negative: !parameter.negative,
		})
	}

	return negated
}

func (p Parameters[T]) If(predicates ...Predicate) Parameters[T] {
	for _, predicate := range predicates {
		if !predicate() {
			return nil
		}
	}

	return p
}

func Not[T interface{ Negate() T }](option T) T {
	return option.Negate()
}
