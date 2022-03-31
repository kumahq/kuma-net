package config

type Owner struct {
	UID uint16
	GID uint16
}

type Inbound struct {
	Port         uint16
	ExcludePorts []uint16
}

type Outbound struct {
	Port         uint16
	ExcludePorts []uint16
}

type DNS struct {
	enabled bool
	Port    uint16
}

func (r *DNS) Enabled() bool {
	return r.enabled
}

type Redirect struct {
	Inbound  *Inbound
	Outbound *Outbound
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

type Chains struct {
	MeshInbound          *Chain
	MeshInboundRedirect  *Chain
	MeshOutbound         *Chain
	MeshOutboundRedirect *Chain
}

func (c *Chains) WithPrefix(prefix string) *Chains {
	return &Chains{
		MeshInbound:          c.MeshInbound.WithPrefix(prefix),
		MeshInboundRedirect:  c.MeshInboundRedirect.WithPrefix(prefix),
		MeshOutbound:         c.MeshOutbound.WithPrefix(prefix),
		MeshOutboundRedirect: c.MeshOutboundRedirect.WithPrefix(prefix),
	}
}

type Config struct {
	Owner    *Owner
	Redirect *Redirect
	Chains   *Chains
	// Verbose when set will generate iptables configuration with longer argument/flag names,
	// additional comments etc.
	Verbose bool
}

func DefaultConfig() *Config {
	return &Config{
		Owner: &Owner{UID: 5678, GID: 5678},
		Redirect: &Redirect{
			Inbound:  &Inbound{Port: 15006},
			Outbound: &Outbound{Port: 15001},
			DNS:      &DNS{Port: 15053, enabled: false},
		},
		Chains: &Chains{
			MeshInbound:          &Chain{Name: "MESH_INBOUND"},
			MeshInboundRedirect:  &Chain{Name: "MESH_INBOUND_REDIRECT"},
			MeshOutbound:         &Chain{Name: "MESH_OUTBOUND"},
			MeshOutboundRedirect: &Chain{Name: "MESH_OUTBOUND_REDIRECT"},
		},
		Verbose: true,
	}
}
