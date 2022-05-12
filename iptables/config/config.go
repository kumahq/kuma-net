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
	Chain         Chain
	RedirectChain Chain
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
	Inbound    TrafficFlow
	Outbound   TrafficFlow
	DNS        DNS
}

type Chain struct {
	Name string
}

func (c Chain) GetFullName(prefix string) string {
	return prefix + c.Name
}

type Config struct {
	Owner    Owner
	Redirect Redirect
	// DropInvalidPackets when set will enable configuration which should drop
	// packets in invalid states
	DropInvalidPackets bool
	// IPv6 when set will be used to configure iptables as well as ip6tables
	IPv6 bool
	// RuntimeOutput is the place where Any debugging, runtime information
	// will be placed (os.Stdout by default)
	RuntimeOutput io.Writer
	// Verbose when set will generate iptables configuration with longer
	// argument/flag names, additional comments etc.
	Verbose bool
}

// ShouldDropInvalidPackets is just a convenience function which can be used in
// iptables conditional command generations instead of inlining anonymous functions
// i.e. AppendIf(ShouldDropInvalidPackets(), Match(...), Jump(Drop()))
func (c Config) ShouldDropInvalidPackets() bool {
	return c.DropInvalidPackets
}

func defaultConfig() Config {
	return Config{
		Owner: Owner{UID: "5678", GID: "5678"},
		Redirect: Redirect{
			NamePrefix: "",
			Inbound: TrafficFlow{
				Port:          15006,
				Chain:         Chain{Name: "MESH_INBOUND"},
				RedirectChain: Chain{Name: "MESH_INBOUND_REDIRECT"},
				ExcludePorts:  []uint16{},
			},
			Outbound: TrafficFlow{
				Port:          15001,
				Chain:         Chain{Name: "MESH_OUTBOUND"},
				RedirectChain: Chain{Name: "MESH_OUTBOUND_REDIRECT"},
				ExcludePorts:  []uint16{},
			},
			DNS: DNS{Port: 15053, Enabled: false, ConntrackZoneSplit: false},
		},
		DropInvalidPackets: false,
		IPv6:               false,
		RuntimeOutput:      os.Stdout,
		Verbose:            true,
	}
}

func MergeConfigWithDefaults(cfg Config) Config {
	result := defaultConfig()

	// .Owner
	if cfg.Owner.UID != "" {
		result.Owner.UID = cfg.Owner.UID
	}

	if cfg.Owner.GID != "" {
		result.Owner.GID = cfg.Owner.GID
	}

	// .Redirect
	if cfg.Redirect.NamePrefix != "" {
		result.Redirect.NamePrefix = cfg.Redirect.NamePrefix
	}

	// .Redirect.Inbound
	if cfg.Redirect.Inbound.Port != 0 {
		result.Redirect.Inbound.Port = cfg.Redirect.Inbound.Port
	}

	if cfg.Redirect.Inbound.Chain.Name != "" {
		result.Redirect.Inbound.Chain.Name = cfg.Redirect.Inbound.Chain.Name
	}

	if cfg.Redirect.Inbound.RedirectChain.Name != "" {
		result.Redirect.Inbound.RedirectChain.Name = cfg.Redirect.Inbound.RedirectChain.Name
	}

	if len(cfg.Redirect.Inbound.ExcludePorts) > 0 {
		result.Redirect.Inbound.ExcludePorts = cfg.Redirect.Inbound.ExcludePorts
	}

	// .Redirect.Outbound
	if cfg.Redirect.Outbound.Port != 0 {
		result.Redirect.Outbound.Port = cfg.Redirect.Outbound.Port
	}

	if cfg.Redirect.Outbound.Chain.Name != "" {
		result.Redirect.Outbound.Chain.Name = cfg.Redirect.Outbound.Chain.Name
	}

	if cfg.Redirect.Outbound.RedirectChain.Name != "" {
		result.Redirect.Outbound.RedirectChain.Name = cfg.Redirect.Outbound.RedirectChain.Name
	}

	if len(cfg.Redirect.Outbound.ExcludePorts) > 0 {
		result.Redirect.Outbound.ExcludePorts = cfg.Redirect.Outbound.ExcludePorts
	}

	// .Redirect.DNS
	result.Redirect.DNS.Enabled = cfg.Redirect.DNS.Enabled
	result.Redirect.DNS.ConntrackZoneSplit = cfg.Redirect.DNS.ConntrackZoneSplit

	if cfg.Redirect.DNS.Port != 0 {
		result.Redirect.DNS.Port = cfg.Redirect.DNS.Port
	}

	// .DropInvalidPackets
	result.DropInvalidPackets = cfg.DropInvalidPackets

	// .IPv6
	result.IPv6 = cfg.IPv6

	// .RuntimeOutput
	if cfg.RuntimeOutput != nil {
		result.RuntimeOutput = cfg.RuntimeOutput
	}

	// .Verbose
	result.Verbose = cfg.Verbose

	return result
}
