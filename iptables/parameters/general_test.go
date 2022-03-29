package parameters_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/parameters"
)

var _ = Describe("General", func() {
	Describe("Source", func() {
		It("should generate '--source' with provided value", func() {
			Expect(parameters.Source("127.0.0.1/32").Build(true)).To(Equal("--source 127.0.0.1/32"))
		})

		It("should generate '! --uid-owner' with provided value", func() {
			// when
			got := parameters.Source("127.0.0.2/32").Negate().Build(true)

			// then
			Expect(got).To(Equal("! --source 127.0.0.2/32"))
		})
	})

	Describe("Destination", func() {
		It("should generate '--destination' with provided value", func() {
			// when
			got := parameters.Destination("127.0.0.1/32").Build(true)

			// then
			Expect(got).To(Equal("--destination 127.0.0.1/32"))
		})

		It("should generate '! --uid-owner' with provided value when negated", func() {
			// when
			got := parameters.Destination("127.0.0.2/32").Negate().Build(true)

			// then
			Expect(got).To(Equal("! --destination 127.0.0.2/32"))
		})
	})

	Describe("OutInterface", func() {
		It("should generate '--out-interface' with provided value", func() {
			Expect(parameters.OutInterface("lo").Build(true)).To(Equal("--out-interface lo"))
		})

		It("should generate '! --out-interface' with provided value when negated", func() {
			// when
			got := parameters.OutInterface("eth1").Negate().Build(true)

			// then
			Expect(got).To(Equal("! --out-interface eth1"))
		})
	})
})
