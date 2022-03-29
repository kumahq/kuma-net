package parameters_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/parameters"
)

var _ = Describe("Parameters", func() {
	Describe("Not", func() {
		DescribeTable("should return negated version of any *General",
			func(option parameters.Parameters[parameters.General]) {
				Expect(parameters.Not(option)).To(BeEquivalentTo(option.Negate()))
			},
			Entry("Source", parameters.Source("127.0.0.1/32")),
			Entry("Destination", parameters.Destination("127.0.0.1/32")),
			Entry("OutInterface", parameters.OutInterface("lo")),
		)

		DescribeTable("should return negated version of any *OwnerParameter",
			func(option parameters.Parameters[parameters.OwnerParameter]) {
				Expect(parameters.Not(option)).To(BeEquivalentTo(option.Negate()))
			},
			Entry("UID", parameters.UID(1234)),
			Entry("GID", parameters.GID(5678)),
		)

		DescribeTable("should return negated version of any *Protocol",
			func(option parameters.Parameters[parameters.Protocol]) {
				Expect(parameters.Not(option)).To(BeEquivalentTo(option.Negate()))
			},
			Entry("DestinationPort", parameters.DestinationPort(6789)),
			Entry("SourcePort", parameters.SourcePort(3467)),
		)

		// TODO (bartsmykla): Add protocols, owner
	})
})
