package target_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder/target"
)

type dummyChain struct {
	name string
}

func (c *dummyChain) ChainName() string {
	return c.name
}

func newDummyChain(name string) *dummyChain {
	return &dummyChain{name: name}
}

var _ = Describe("RedirectParameter", func() {
	Describe("Ports", func() {
		It("should generate 'REDIRECT --to-ports' with provided port", func() {
			Expect(target.Ports(8888).String()).To(Equal("REDIRECT --to-ports 8888"))
		})

		It("should generate 'REDIRECT --to-ports' with comma-separated ports when provided more "+
			"than one", func() {
			// when
			got := target.Ports(7777, 8888, 7878).String()

			// then
			Expect(got).To(Equal("REDIRECT --to-ports 7777,8888,7878"))
		})
	})

	Describe("To", func() {
		It("should generate '<CHAIN_NAME>'", func() {
			// given
			chainName := "FOO_BAR"
			chain := newDummyChain(chainName)

			// when
			got := target.To(chain).String()

			// then
			Expect(got).To(Equal(chainName))
		})
	})
})

var _ = Describe("Func", func() {
	Describe("Return", func() {
		It("should generate string slice with only 'RETURN' inside", func() {
			Expect(target.Return()).To(BeEquivalentTo([]string{"RETURN"}))
		})
	})

	Describe("Redirect", func() {
		It("should generate string slice with only '<CHAIN_NAME>' inside when To option "+
			"provided", func() {
			// given
			chainName := "BAZ_FUZ"
			chain := newDummyChain(chainName)

			// when
			got := target.Redirect(target.To(chain))()

			// then
			Expect(got).To(BeEquivalentTo([]string{chainName}))
		})
	})
})
