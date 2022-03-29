package chain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder/chain"
)

// TODO (bartsmykla): implement some validation logic as custom chains cannot have default policies
//  they can have packet and byte counters specified though

var _ = Describe("NewChain", func() {
	DescribeTable("NewChain",
		func(builder *chain.ChainBuilder, wantDefaultPolicy string, wantRules []string) {
			// when
			gotDefaultPolicy, gotRules := builder.Build(false)

			// then
			Expect(gotDefaultPolicy).To(Equal(wantDefaultPolicy))
			Expect(gotRules).To(BeEquivalentTo(wantRules))
		},
		Entry("should be able to generate PREROUTING built-in chain, with default policy: "+
			"':PREROUTING ACCEPT [0:0]' and without any rules",
			chain.Prerouting(), ":PREROUTING ACCEPT [0:0]", nil),
		Entry("should be able to generate INPUT built-in chain, with default policy: "+
			"':INPUT ACCEPT [0:0]' and without any rules",
			chain.Input(), ":INPUT ACCEPT [0:0]", nil),
		Entry("should be able to generate OUTPUT built-in chain, with default policy: "+
			"':OUTPUT ACCEPT [0:0]' and without any rules",
			chain.Output(), ":OUTPUT ACCEPT [0:0]", nil),
		Entry("should be able to generate POSTROUTING built-in chain, with default policy: "+
			"':POSTROUTING ACCEPT [0:0]' and without any rules",
			chain.Postrouting(), ":POSTROUTING ACCEPT [0:0]", nil),
		Entry("should be able to generate new custom chain, with default policy: "+
			"':FOO_BAR_BAZ - [0:0]' and without any rules",
			chain.NewChain("FOO_BAR_BAZ"), ":FOO_BAR_BAZ - [0:0]", nil),
	)
})
