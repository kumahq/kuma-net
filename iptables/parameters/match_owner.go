package parameters

// Owner
//       This module attempts to match various characteristics of the packet creator,
//       for locally generated packets. This match is only valid in the OUTPUT and POSTROUTING
//       chains. Forwarded packets do not have any socket associated with them.
//       Packets from kernel threads do have a socket, but usually no owner.
//
//       [!] --uid-owner userid
//              Matches if the packet socket's file structure (if it has one) is owned by
//              the given user.
//
//       [!] --gid-owner groupid
//              Matches if the packet socket's file structure is owned by the given group.
//
// ref. iptables-extensions(8) > owner

import (
	"fmt"
	"strconv"
)

type OwnerParameter struct {
	flag     string
	value    string
	negative bool
}

func (p *OwnerParameter) Negate() ParameterBuilder {
	p.negative = !p.negative

	return p
}

func (p *OwnerParameter) Build(bool) string {
	if p.negative {
		return fmt.Sprintf("! %s %s", p.flag, p.value)
	}

	return fmt.Sprintf("%s %s", p.flag, p.value)
}

func uid(id uint16, negative bool) *OwnerParameter {
	return &OwnerParameter{
		flag:     "--uid-owner",
		value:    strconv.Itoa(int(id)),
		negative: negative,
	}
}

// Uid matches if the packet socket's file structure (if it has one) is owned by the user
// with given UID
func Uid(id uint16) *OwnerParameter {
	return uid(id, false)
}

func NotUid(id uint16) *OwnerParameter {
	return uid(id, true)
}

func gid(id uint16, negative bool) *OwnerParameter {
	return &OwnerParameter{
		flag:     "--gid-owner",
		value:    strconv.Itoa(int(id)),
		negative: negative,
	}
}

// Gid Matches if the packet socket's file structure is owned by the given group
func Gid(id uint16) *OwnerParameter {
	return gid(id, false)
}

func NotGid(id uint16) *OwnerParameter {
	return gid(id, true)
}

// Owner attempts to match various characteristics of the packet creator,for locally generated
// packets. This match is only valid in the OUTPUT and POSTROUTING chains. Forwarded packets
// do not have any socket associated with them. Packets from kernel threads do have a socket,
// but usually no owner
func Owner(ownerParameters ...*OwnerParameter) *MatchParameter {
	var parameters []ParameterBuilder

	for _, parameter := range ownerParameters {
		parameters = append(parameters, parameter)
	}

	return &MatchParameter{
		name:       "owner",
		parameters: parameters,
	}
}
