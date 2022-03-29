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
// ref. iptables-extensions(8)#owner

package parameters

import (
	"strconv"
)

type OwnerParameter struct{}

// Owner attempts to match various characteristics of the packet creator,for locally generated
// packets. This match is only valid in the OUTPUT and POSTROUTING chains. Forwarded packets
// do not have any socket associated with them. Packets from kernel threads do have a socket,
// but usually no owner
//
// Note: at least one option is required thus split of parameters
func Owner(
	option Parameters[OwnerParameter],
	options ...Parameters[OwnerParameter],
) Parameters[OwnerParameter] {
	return newParameters[OwnerParameter]("--match", "-m", "owner").concat(option).concat(options...)
}

// UID matches if the packet socket's file structure (if it has one) is owned by the user
// with given UID
func UID(uid uint16) Parameters[OwnerParameter] {
	return newParameters[OwnerParameter]("--uid-owner", "", strconv.Itoa(int(uid)))
}

// GID Matches if the packet socket's file structure is owned by the given group
func GID(gid uint16) Parameters[OwnerParameter] {
	return newParameters[OwnerParameter]("--gid-owner", "", strconv.Itoa(int(gid)))
}
