package config

import (
	"io"
	"os"
)

type Owner struct {
	UID string
	GID string
}

// TrafficFlow is a struct for Inbound/Outbound configuration
type TrafficFlow struct {
	Port          uint16
	Chain         *Chain
	RedirectChain *Chain
	ExcludePorts  []uint16
}

type DNS struct {
	Enabled            bool
	Port               uint16
	ConntrackZoneSplit bool
}

type Redirect struct {
	// NamePrefix is a prefix which will be used go generate chains name
	NamePrefix string
	Inbound    *TrafficFlow
	Outbound   *TrafficFlow
	DNS        *DNS
}

type Chain struct {
	Name string
}

func (c *Chain) GetFullName(prefix string) string {
	return prefix + c.Name
}

type Config struct {
	Owner    *Owner
	Redirect *Redirect
	// Output is the place where if any produced informational data (i.e. dump of the rules
	// which will be applied for helping user to potentially debug if something would
	// go wrong) will be placed (os.Stdout by default)
	Output io.Writer
	// Verbose when set will generate iptables configuration with longer argument/flag names,
	// additional comments etc.
	Verbose bool
}

func DefaultConfig() *Config {
	return &Config{
		Owner: &Owner{UID: "5678", GID: "5678"},
		Redirect: &Redirect{
			NamePrefix: "",
			Inbound: &TrafficFlow{
				Port:          15006,
				Chain:         &Chain{Name: "MESH_INBOUND"},
				RedirectChain: &Chain{Name: "MESH_INBOUND_REDIRECT"},
				ExcludePorts:  []uint16{},
			},
			Outbound: &TrafficFlow{
				Port:          15001,
				Chain:         &Chain{Name: "MESH_OUTBOUND"},
				RedirectChain: &Chain{Name: "MESH_OUTBOUND_REDIRECT"},
				ExcludePorts:  []uint16{},
			},
			DNS: &DNS{Port: 15053, Enabled: false, ConntrackZoneSplit: true},
		},
		Output:  os.Stdout,
		Verbose: true,
	}
}
