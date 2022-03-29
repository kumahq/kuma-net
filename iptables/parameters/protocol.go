package parameters

import (
	"strconv"
)

type Protocol struct{}

func protocolMatcher(protocol string, parameters ...Parameters[Protocol]) Parameters[Protocol] {
	// As with introduction of If() methods option can be nil, when if predicate returns false,
	// we have to filter out parameters
	var filtered []Parameters[Protocol]
	for _, params := range parameters {
		filtered = append(filtered, params)
	}

	finalParameters := newParameters[Protocol]("--protocol", "-p", protocol)

	// If we want to specify --destination-port | --dport, --source-port | --sport
	// or other tcp, or udp specific configuration we have to add --match | -m tcp flag
	// ref. iptables-extensions(8) > tcp
	// ref. iptables-extensions(8) > udp
	if len(filtered) > 0 {
		finalParameters = finalParameters.append("--match", "-m", protocol, false)
	}

	for _, params := range filtered {
		finalParameters = finalParameters.concat(params)
	}

	return finalParameters
}

func TCP(opts ...Parameters[Protocol]) Parameters[Protocol] {
	return protocolMatcher("tcp", opts...)
}

func UDP(opts ...Parameters[Protocol]) Parameters[Protocol] {
	return protocolMatcher("udp", opts...)
}

func SourcePort(port uint16) Parameters[Protocol] {
	return newParameters[Protocol]("--source-port", "--sport", strconv.Itoa(int(port)))
}

func DestinationPort(port uint16) Parameters[Protocol] {
	return newParameters[Protocol]("--destination-port", "--dport", strconv.Itoa(int(port)))
}
