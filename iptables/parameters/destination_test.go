package parameters_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/kumahq/kuma-net/iptables/parameters"
)

var _ = Describe("DestinationParameter", func() {
	Describe("Destination", func() {
		DescribeTable("should build valid destination parameter with, provided address",
			func(address, want, wantVerbose string) {
				// when
				got := Destination(address)

				// then
				Expect(got.Build(false)).To(Equal(want))
				Expect(got.Build(true)).To(Equal(wantVerbose))
			},
			Entry("IPv4 IP address", "254.254.254.254",
				"-d 254.254.254.254",
				"--destination 254.254.254.254",
			),
			Entry("IPv4 IP address with CIDR mask", "127.0.0.1/32",
				"-d 127.0.0.1/32",
				"--destination 127.0.0.1/32",
			),
			Entry("IPv6 IP address", "::1",
				"-d ::1",
				"--destination ::1",
			),
			Entry("IPv6 IP address with CIDR mask", "f675:e763:3d02:e43c:4734:c8db:481e:295a/128",
				"-d f675:e763:3d02:e43c:4734:c8db:481e:295a/128",
				"--destination f675:e763:3d02:e43c:4734:c8db:481e:295a/128",
			),
		)

		DescribeTable("should build valid negated destination parameter with, provided address, "+
			"when negated",
			func(address, want, wantVerbose string) {
				// when
				got := Destination(address).Negate()

				// then
				Expect(got.Build(false)).To(Equal(want))
				Expect(got.Build(true)).To(Equal(wantVerbose))
			},
			Entry("IPv4 IP address", "254.254.254.254",
				"! -d 254.254.254.254",
				"! --destination 254.254.254.254",
			),
			Entry("IPv4 IP address with CIDR mask", "127.0.0.1/32",
				"! -d 127.0.0.1/32",
				"! --destination 127.0.0.1/32",
			),
			Entry("IPv6 IP address", "::1",
				"! -d ::1",
				"! --destination ::1",
			),
			Entry("IPv6 IP address with CIDR mask", "f675:e763:3d02:e43c:4734:c8db:481e:295a/128",
				"! -d f675:e763:3d02:e43c:4734:c8db:481e:295a/128",
				"! --destination f675:e763:3d02:e43c:4734:c8db:481e:295a/128",
			),
		)
	})

	Describe("NotDestination", func() {
		DescribeTable("should return the result of Destination(...).Negate()",
			func(address string) {
				// given
				want := Destination(address).Negate()

				// when
				got := NotDestination(address)

				// then
				Expect(got.Build(false)).To(BeEquivalentTo(want.Build(false)))
				Expect(got.Build(true)).To(BeEquivalentTo(want.Build(true)))
			},
			Entry("IPv4 IP address", "254.254.254.254"),
			Entry("IPv4 IP address with CIDR mask", "127.0.0.1/32"),
			Entry("IPv6 IP address", "::1"),
			Entry("IPv6 IP address with CIDR mask", "f675:e763:3d02:e43c:4734:c8db:481e:295a/128"),
		)
	})
})
