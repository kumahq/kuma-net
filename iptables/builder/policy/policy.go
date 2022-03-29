package policy

import (
	"fmt"
)

type Policy string

const (
	ACCEPT Policy = "ACCEPT"
	DROP   Policy = "DROP"
	// NONE is used for custom chains as default policies are available only the default chains
	NONE Policy = "-"
)

type DefaultPolicy struct {
	policy        Policy
	packetCounter uint
	byteCounter   uint
}

func (p *DefaultPolicy) WithPacketCounter(counter uint) *DefaultPolicy {
	p.packetCounter = counter

	return p
}

func (p *DefaultPolicy) WithByteCounter(counter uint) *DefaultPolicy {
	p.byteCounter = counter

	return p
}

func (p *DefaultPolicy) String() string {
	return fmt.Sprintf("%s [%d:%d]", p.policy, p.packetCounter, p.byteCounter)
}

func Accept() *DefaultPolicy {
	return &DefaultPolicy{policy: ACCEPT}
}

func None() *DefaultPolicy {
	return &DefaultPolicy{policy: NONE}
}

func Drop() *DefaultPolicy {
	return &DefaultPolicy{policy: DROP}
}
