package commands

import (
	"strings"

	"github.com/kumahq/kuma-net/iptables/parameters"
)

type Command struct {
	long       string
	short      string
	chainName  string
	parameters []*parameters.Parameter
}

func (c *Command) Build(verbose bool) string {
	flag := c.short

	if verbose {
		flag = c.long
	}

	cmd := []string{flag}

	if c.chainName != "" {
		cmd = append(cmd, c.chainName)
	}

	for _, parameter := range c.parameters {
		if parameter != nil {
			cmd = append(cmd, parameter.Build(verbose))
		}
	}

	return strings.Join(cmd, " ")
}

func Append(chainName string, parameters []*parameters.Parameter) *Command {
	return &Command{
		long:       "--append",
		short:      "-A",
		chainName:  chainName,
		parameters: parameters,
	}
}
