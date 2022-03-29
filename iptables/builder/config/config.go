package config

type Owner struct {
	UID uint16
	GID uint16
}

func DefaultOwner() *Owner {
	return &Owner{
		UID: 5678,
		GID: 5678,
	}
}

type Inbound struct {
	Port uint16
}

func (r *Inbound) Enabled() bool {
	return r != nil
}

func DefaultInbound() *Inbound {
	return &Inbound{Port: 15006}
}

type Outbound struct {
	Port uint16
}

func (r *Outbound) Enabled() bool {
	return r != nil
}

func DefaultOutbound() *Outbound {
	return &Outbound{Port: 15001}
}

type DNS struct {
	enabled bool
	Port    uint16
}

func (r *DNS) Enabled() bool {
	return r.enabled
}

func DefaultDNS() *DNS {
	return &DNS{Port: 15053, enabled: false}
}

type Redirect struct {
	// If Inbound is not nil, we assume we want to redirect inbound traffic, if it's nil,
	// then no iptables rules for inbound redirection will be generated
	Inbound *Inbound
	// If Outbound is not nil, we assume we want to redirect outbound traffic, if it's nil,
	// then no iptables rules for outbound redirection will be generated
	Outbound *Outbound
	// If DNS is not nil, we assume we want to redirect DNS traffic, if it's nil,
	// then no iptables rules for DNS redirection will be generated
	DNS *DNS
}

func DefaultRedirect() *Redirect {
	return &Redirect{
		Inbound:  DefaultInbound(),
		Outbound: DefaultOutbound(),
		DNS:      DefaultDNS(),
	}
}

type MeshInbound struct {
	Name string
}

func DefaultMeshInbound() *MeshInbound {
	return &MeshInbound{Name: "MESH_INBOUND"}
}

type MeshInboundRedirect struct {
	Name string
}

func DefaultMeshInboundRedirect() *MeshInboundRedirect {
	return &MeshInboundRedirect{Name: "MESH_INBOUND_REDIRECT"}
}

type MeshOutbound struct {
	Name string
}

func DefaultMeshOutbound() *MeshOutbound {
	return &MeshOutbound{Name: "MESH_OUTBOUND"}
}

type MeshOutboundRedirect struct {
	Name string
}

func DefaultMeshOutboundRedirect() *MeshOutboundRedirect {
	return &MeshOutboundRedirect{Name: "MESH_OUTBOUND_REDIRECT"}
}

type NAT struct {
	MeshInbound          *MeshInbound
	MeshInboundRedirect  *MeshInboundRedirect
	MeshOutbound         *MeshOutbound
	MeshOutboundRedirect *MeshOutboundRedirect
}

func DefaultNAT() *NAT {
	return &NAT{
		MeshInbound:          DefaultMeshInbound(),
		MeshInboundRedirect:  DefaultMeshInboundRedirect(),
		MeshOutbound:         DefaultMeshOutbound(),
		MeshOutboundRedirect: DefaultMeshOutboundRedirect(),
	}
}

type Tables struct {
	Nat *NAT
}

func DefaultTables() *Tables {
	return &Tables{
		Nat: DefaultNAT(),
	}
}

type Config struct {
	Owner    *Owner
	Redirect *Redirect
	Tables   *Tables
	// Verbose when set will generate iptables configuration with longer argument/flag names,
	// additional comments etc.
	Verbose bool
}

func DefaultConfig() *Config {
	return &Config{
		Owner:    DefaultOwner(),
		Redirect: DefaultRedirect(),
		Tables:   DefaultTables(),
		Verbose:  true,
	}
}
