package chain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder/chain"
)

var _ = Describe("WithChain", func() {
	DescribeTable("WithChain",
		func(builder *chain.ChainBuilder, rules []string) {
			// when
			got := builder.Build(false)

			// then
			Expect(got).To(BeEquivalentTo(rules))
		},
		Entry("should be able to generate PREROUTING built-in chain without any rules",
			chain.Prerouting(), nil),
		Entry("should be able to generate INPUT built-in chain without any rules",
			chain.Input(), nil),
		Entry("should be able to generate OUTPUT built-in chain without any rules",
			chain.Output(), nil),
		Entry("should be able to generate POSTROUTING built-in chain without any rules",
			chain.Postrouting(), nil),
		Entry("should be able to generate new custom chain without any rules",
			chain.NewChain("FOO_BAR_BAZ"), nil),
	)
})
