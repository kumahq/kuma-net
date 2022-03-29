package parameters_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/parameters"
)

var _ = Describe("TCP", func() {
	verbose := true

	It("should generate '--protocol tcp' when no protocol options provided", func() {
		Expect(parameters.TCP().Build(verbose)).To(Equal("--protocol tcp"))
	})

	It("should generate '! --protocol tcp' when negated and no options provided", func() {
		Expect(parameters.TCP().Negate().Build(verbose)).To(Equal("! --protocol tcp"))
	})

	It("should generate '--protocol tcp' and '--match tcp' with provided options", func() {
		// given
		destinationPort := parameters.DestinationPort(1234)
		sourcePort := parameters.SourcePort(5678)

		// when
		got := parameters.TCP(destinationPort, sourcePort).Build(verbose)

		// then
		want := fmt.Sprintf(
			"--protocol tcp --match tcp %s %s",
			destinationPort.Build(verbose),
			sourcePort.Build(verbose),
		)

		Expect(got).To(Equal(want))
	})
})

var _ = Describe("Protocol", func() {
	verbose := true

	Describe("DestinationPort", func() {
		It("should generate '--destination-port' with provided port", func() {
			// when
			got := parameters.DestinationPort(7777).Build(verbose)

			// then
			Expect(got).To(Equal("--destination-port 7777"))
		})

		It("should generate '! --destination-port' with provided port when negated", func() {
			// when
			got := parameters.DestinationPort(7777).Negate().Build(verbose)

			// then
			Expect(got).To(Equal("! --destination-port 7777"))
		})
	})

	Describe("SourcePort", func() {
		It("should generate '--source-port' with provided port", func() {
			// when
			got := parameters.SourcePort(7777).Build(verbose)

			// then
			Expect(got).To(Equal("--source-port 7777"))
		})

		It("should generate '! --source-port' with provided port when negated", func() {
			// when
			got := parameters.SourcePort(7777).Negate().Build(verbose)

			// then
			Expect(got).To(Equal("! --source-port 7777"))
		})
	})
})
