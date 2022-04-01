package config

type Owner struct {
	UID uint16
	GID uint16
}

// TrafficFlow is a struct for Inbound/Outbound configuration
type TrafficFlow struct {
	Port          uint16
	Chain         *Chain
	RedirectChain *Chain
	ExcludePorts  []uint16
}

type DNS struct {
	enabled bool
	Port    uint16
}

func (r *DNS) Enabled() bool {
	return r.enabled
}

type Redirect struct {
	Inbound  *TrafficFlow
	Outbound *TrafficFlow
	DNS      *DNS
}

type MeshInbound struct {
	Prefix string
	Name   string
}

func (c *MeshInbound) GetFullName() string {
	return c.Prefix + c.Name
}

type Chain struct {
	Prefix string
	Name   string
}

func (c *Chain) WithPrefix(prefix string) *Chain {
	c.Prefix = prefix

	return c
}

func (c *Chain) GetFullName() string {
	return c.Prefix + c.Name
}

type Config struct {
	Owner    *Owner
	Redirect *Redirect
	// Verbose when set will generate iptables configuration with longer argument/flag names,
	// additional comments etc.
	Verbose bool
}

func DefaultConfig() *Config {
	return &Config{
		Owner: &Owner{UID: 5678, GID: 5678},
		Redirect: &Redirect{
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
			DNS: &DNS{Port: 15053, enabled: false},
		},
		Verbose: true,
	}
}
