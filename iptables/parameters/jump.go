package parameters

import (
	"strconv"
	"strings"
)

type NewJumpParameter struct {
	parameters []string
}

func (p *NewJumpParameter) Build(bool) string {
	return strings.Join(p.parameters, " ")
}

func (p *NewJumpParameter) Negate() ParameterBuilder {
	return p
}

func Jump(parameter *NewJumpParameter) *Parameter {
	return &Parameter{
		long:       "--jump",
		short:      "-j",
		parameters: []ParameterBuilder{parameter},
	}
}

func ToUserDefinedChain(chainName string) *NewJumpParameter {
	return &NewJumpParameter{parameters: []string{chainName}}
}

func ToPort(port uint16) *NewJumpParameter {
	return &NewJumpParameter{parameters: []string{
		"REDIRECT",
		"--to-ports",
		strconv.Itoa(int(port)),
	}}
}

func Return() *NewJumpParameter {
	return &NewJumpParameter{parameters: []string{"RETURN"}}
}
