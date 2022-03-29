package policy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder/policy"
)

var _ = Describe("Policy", func() {
	Describe("Accept", func() {
		It("should generate 'ACCEPT [0:0]' by default", func() {
			Expect(policy.Accept().String()).To(Equal("ACCEPT [0:0]"))
		})

		It("should generate 'ACCEPT' with provided packet count", func() {
			Expect(policy.Accept().WithPacketCounter(12345).String()).To(Equal("ACCEPT [12345:0]"))
		})

		It("should generate 'ACCEPT' with provided byte count", func() {
			Expect(policy.Accept().WithByteCounter(67890).String()).To(Equal("ACCEPT [0:67890]"))
		})

		It("should generate 'ACCEPT' with provided packet and byte counts", func() {
			// when
			got := policy.Accept().
				WithPacketCounter(12345).
				WithByteCounter(67890).
				String()

			// then
			Expect(got).To(Equal("ACCEPT [12345:67890]"))
		})
	})

	Describe("Drop", func() {
		It("should generate 'DROP [0:0]' by default", func() {
			Expect(policy.Drop().String()).To(Equal("DROP [0:0]"))
		})

		It("should generate 'DROP' with provided packet count", func() {
			Expect(policy.Drop().WithPacketCounter(12345).String()).To(Equal("DROP [12345:0]"))
		})

		It("should generate 'DROP' with provided byte count", func() {
			Expect(policy.Drop().WithByteCounter(67890).String()).To(Equal("DROP [0:67890]"))
		})

		It("should generate 'DROP' with provided packet and byte counts", func() {
			// when
			got := policy.Drop().
				WithPacketCounter(12345).
				WithByteCounter(67890).
				String()

			// then
			Expect(got).To(Equal("DROP [12345:67890]"))
		})
	})

	Describe("None", func() {
		It("should generate '- [0:0]' by default", func() {
			Expect(policy.None().String()).To(Equal("- [0:0]"))
		})

		It("should generate '-' with provided packet count", func() {
			Expect(policy.None().WithPacketCounter(12345).String()).To(Equal("- [12345:0]"))
		})

		It("should generate '-' with provided byte count", func() {
			Expect(policy.None().WithByteCounter(67890).String()).To(Equal("- [0:67890]"))
		})

		It("should generate '-' with provided packet and byte counts", func() {
			// when
			got := policy.None().
				WithPacketCounter(12345).
				WithByteCounter(67890).
				String()

			// then
			Expect(got).To(Equal("- [12345:67890]"))
		})
	})
})
