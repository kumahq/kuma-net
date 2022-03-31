package parameters

type General struct{}

// Source will generate arguments for the "-s, --source address[/mask]" flag
// Address can be either a network name, a hostname, a network IP address (with /mask),
// or a plain IP address. Hostnames will be resolved once only, before the rule is submitted
// to the kernel. Please note that specifying any name to be resolved with a remote query such as
// DNS is a horrible idea. The mask can be either an ipv4 network mask (for iptables) or
// a plain number, specifying the number of 1's on the left side of the network mask.
// Thus, an iptables mask of 24 is equivalent to 255.255.255.0
//
// ref. iptables(8) > PARAMETERS
func Source(address string) Parameters[General] {
	return newParameters[General]("--source", "-s", address)
}

// Destination will generate arguments for the "-d, --destination address[/mask]" flag
// See the description of the -s (source) flag for a detailed description of the syntax
//
// ref. iptables(8) > PARAMETERS
func Destination(address string) Parameters[General] {
	return newParameters[General]("--destination", "-d", address)
}

// OutInterface will generate arguments for the "-o, --out-interface name" flag
// Name of an interface via which a packet is going to be sent (for packets entering the FORWARD,
// OUTPUT and POSTROUTING chains). If the interface name ends in a "+", then any interface
// which begins with this name will match
//
// ref. iptables(8) > PARAMETERS
func OutInterface(name string) Parameters[General] {
	return newParameters[General]("--out-interface", "-o", name)
}
