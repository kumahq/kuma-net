package parameters

import (
	"strconv"
	"strings"
)

type JumpParameter struct {
	parameters []string
}

func (p *JumpParameter) Build(bool) string {
	return strings.Join(p.parameters, " ")
}

func (p *JumpParameter) Negate() ParameterBuilder {
	return p
}

func Jump(parameter *JumpParameter) *Parameter {
	return &Parameter{
		long:       "--jump",
		short:      "-j",
		parameters: []ParameterBuilder{parameter},
	}
}

func ToUserDefinedChain(chainName string) *JumpParameter {
	return &JumpParameter{parameters: []string{chainName}}
}

func ToPort(port uint16) *JumpParameter {
	return &JumpParameter{parameters: []string{
		"REDIRECT",
		"--to-ports",
		strconv.Itoa(int(port)),
	}}
}

func Return() *JumpParameter {
	return &JumpParameter{parameters: []string{"RETURN"}}
}
