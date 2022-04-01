package parameters_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/kumahq/kuma-net/iptables/parameters"
)

var _ = Describe("ProtocolParameter", func() {
	Describe("Protocol", func() {
		DescribeTable("DestinationPort",
			func(port int, verbose bool, want string) {
				// when
				got := DestinationPort(uint16(port)).Build(verbose)

				// then
				Expect(got).To(Equal(want))
			},
			Entry(nil, 22, false, "--dport 22"),
			Entry(nil, 22, true, "--destination-port 22"),
			Entry(nil, 7777, false, "--dport 7777"),
			Entry(nil, 7777, true, "--destination-port 7777"),
		)

		DescribeTable("NotDestinationPort",
			func(port int, verbose bool, want string) {
				// when
				got := NotDestinationPort(uint16(port)).Build(verbose)

				// then
				Expect(got).To(Equal(want))
			},
			Entry(nil, 22, false, "! --dport 22"),
			Entry(nil, 22, true, "! --destination-port 22"),
			Entry(nil, 7777, false, "! --dport 7777"),
			Entry(nil, 7777, true, "! --destination-port 7777"),
		)

		Describe("NotDestinationPortIf", func() {
			It("should return nil, when predicate returns false", func() {
				Expect(NotDestinationPortIf(func() bool {
					return false
				}, 22)).To(BeNil())
			})

			// TODO (bartsmykla): add cases when predicate returns true
		})

		DescribeTable("Tcp",
			func(parameters []*TcpUdpParameter, verbose bool, want string) {
				// when
				got := Tcp(parameters...).Build(verbose)

				// then
				Expect(got).To(Equal(want))
			},
			Entry(nil, nil, false, "tcp"),
			Entry(nil, nil, true, "tcp"),
			Entry(nil, []*TcpUdpParameter{DestinationPort(22)}, false, "tcp --dport 22"),
			Entry(nil,
				[]*TcpUdpParameter{DestinationPort(22)},
				true,
				"tcp --destination-port 22",
			),
			Entry(nil,
				[]*TcpUdpParameter{NotDestinationPort(22)},
				false,
				"tcp ! --dport 22",
			),
			Entry(nil,
				[]*TcpUdpParameter{NotDestinationPort(22)},
				true,
				"tcp ! --destination-port 22",
			),
		)
	})
})
