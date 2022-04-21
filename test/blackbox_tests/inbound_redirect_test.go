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
			tcpServerAddress := fmt.Sprintf(":%d", tcpServerPort)
			peerAddress := ns.Veth().PeerAddress()
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Inbound: config.TrafficFlow{
						Port: tcpServerPort,
					},
				},
				RuntimeOutput: ioutil.Discard,
			}

			tcpReadyC, tcpErrC := ns.StartTCPServer(tcpServerAddress, tcp.ReplyWithOriginalDst)
			Eventually(tcpReadyC).Should(BeClosed())

			// when
			Eventually(ns.Exec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Expect(tcp.DialAndGetReply(peerAddress, port)).
				To(Equal(fmt.Sprintf("%s:%d", peerAddress, port)))

			// and, then
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
				RuntimeOutput: ioutil.Discard,
			}
			peerAddress := ns.Veth().PeerAddress()

			redirectReadyC, redirectErrC := ns.StartTCPServer(
				fmt.Sprintf(":%d", redirectTCPServerPort),
				tcp.ReplyWithOriginalDst,
			)
			Eventually(redirectReadyC).Should(BeClosed())

			excludedReadyC, excludedErrC := ns.StartTCPServer(
				fmt.Sprintf(":%d", excludedPort),
				tcp.ReplyWith("foobar"),
			)
			Eventually(excludedReadyC).Should(BeClosed())

			// when
			Eventually(ns.Exec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Expect(tcp.DialAndGetReply(peerAddress, excludedPort)).To(Equal("foobar"))

			Expect(tcp.DialAndGetReply(peerAddress, port)).
				To(Equal(fmt.Sprintf("%s:%d", peerAddress, port)))

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
