package parameters_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/parameters"
)

var _ = Describe("Owner", func() {
	It("should generate '--match owner' with provided options", func() {
		// given
		uid := parameters.UID(1234)
		gid := parameters.GID(5678)

		// when
		got := parameters.Owner(uid, gid).Build(true)

		// then
		want := fmt.Sprintf("--match owner %s %s", uid.Build(true), gid.Build(true))

		Expect(got).To(Equal(want))
	})

	// TODO (bartsmykla): ! --match is invalid, so behaviour when negated should be
	//  different here probably
	// It("should generate '! --match owner' with provided options when negated", func() {
	// 	// given
	// 	uid := options.UID(2345)
	// 	gid := options.GID(6789)
	//
	// 	// when
	// 	got := options.Owner(uid, gid).Negate().Build()
	//
	// 	// then
	// 	want := fmt.Sprintf("--match owner %s %s", uid, gid)
	//
	// 	Expect(got).To(Equal(want))
	// })
})

var _ = Describe("OwnerParameter", func() {
	Describe("UID", func() {
		It("should generate '--uid-owner' with provided UID", func() {
			Expect(parameters.UID(8888).Build(true)).To(Equal("--uid-owner 8888"))
		})

		It("should generate '! --uid-owner' with provided UID when negated", func() {
			Expect(parameters.UID(9999).Negate().Build(true)).To(Equal("! --uid-owner 9999"))
		})
	})

	Describe("GID", func() {
		It("should generate '--gid-owner' with provided GID", func() {
			Expect(parameters.GID(8888).Build(true)).To(Equal("--gid-owner 8888"))
		})

		It("should generate '! --gid-owner' with provided GID when negated", func() {
			Expect(parameters.GID(9999).Negate().Build(true)).To(Equal("! --gid-owner 9999"))
		})
	})
})
