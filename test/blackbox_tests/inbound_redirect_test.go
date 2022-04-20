package blackbox_tests_test

import (
	"fmt"
	"io/ioutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder"
	"github.com/kumahq/kuma-net/iptables/config"
	"github.com/kumahq/kuma-net/test/framework/netns"
	"github.com/kumahq/kuma-net/test/framework/socket"
	"github.com/kumahq/kuma-net/test/framework/tcp"
)

var _ = Describe("Inbound TCP traffic from all ports", func() {
	var err error
	var ns *netns.NetNS
	var tcpServerPort uint16

	BeforeEach(func() {
		DeferCleanup(ns.Cleanup)

		tcpServerPort = socket.GenFreeRandomPort()

		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	DescribeTable("should be redirected to outbound port",
		func(port uint16) {
			// given
			address := fmt.Sprintf(":%d", tcpServerPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Inbound: config.TrafficFlow{
						Port: tcpServerPort,
					},
				},
				Output: ioutil.Discard,
			}

			tcpReadyC, tcpErrC := ns.StartTCPServer(address, tcp.ReplyWithOriginalDst)
			Eventually(tcpReadyC).Should(BeClosed())

			// when
			Eventually(ns.Exec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Expect(tcp.DialAndGetReply(ns.Veth().PeerAddress(), port)).
				To(Equal([]byte(fmt.Sprintf("%s:%d", ns.Veth().PeerAddress(), port))))

			Consistently(tcpErrC).ShouldNot(Receive())
		},
		func() []TableEntry {
			ports := socket.GenerateRandomPorts(50, tcpServerPort)
			desc := fmt.Sprintf("to port %d, from port %%d", tcpServerPort)

			var entries []TableEntry
			for port := range ports {
				entries = append(entries, Entry(EntryDescription(desc), port))
			}

			return entries
		}(),
	)
})

var _ = Describe("Inbound TCP traffic from all ports except excluded ones", func() {
	var err error
	var ns *netns.NetNS
	var redirectTCPServerPort, excludedPort uint16

	BeforeEach(func() {
		DeferCleanup(ns.Cleanup)

		redirectTCPServerPort = socket.GenFreeRandomPort()
		excludedPort = socket.GenFreeRandomPort()

		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	DescribeTable("should be redirected to outbound port",
		func(port uint16) {
			// given
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Inbound: config.TrafficFlow{
						Port:         redirectTCPServerPort,
						ExcludePorts: []uint16{excludedPort},
					},
				},
				Output: ioutil.Discard,
			}
			want := []byte("foobar")

			redirectReadyC, redirectErrC := ns.StartTCPServer(
				fmt.Sprintf(":%d", redirectTCPServerPort),
				tcp.ReplyWithOriginalDst,
			)
			Eventually(redirectReadyC).Should(BeClosed())

			excludedReadyC, excludedErrC := ns.StartTCPServer(
				fmt.Sprintf(":%d", excludedPort),
				tcp.ReplyWith(want),
			)
			Eventually(excludedReadyC).Should(BeClosed())

			// when
			Eventually(ns.Exec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Expect(tcp.DialAndGetReply(ns.Veth().PeerAddress(), excludedPort)).
				To(Equal(want))

			Expect(tcp.DialAndGetReply(ns.Veth().PeerAddress(), port)).
				To(Equal([]byte(fmt.Sprintf("%s:%d", ns.Veth().PeerAddress(), port))))

			Consistently(redirectErrC).ShouldNot(Receive())
			Consistently(excludedErrC).ShouldNot(Receive())
		},
		func() []TableEntry {
			ports := socket.GenerateRandomPorts(50, redirectTCPServerPort)
			desc := fmt.Sprintf("to port %d, from port %%d", redirectTCPServerPort)

			var entries []TableEntry
			for port := range ports {
				entries = append(entries, Entry(EntryDescription(desc), port))
			}

			return entries
		}(),
	)
})
