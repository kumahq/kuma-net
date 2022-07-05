package blackbox_tests_test

import (
	"fmt"
	"io/ioutil"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder"
	"github.com/kumahq/kuma-net/iptables/config"
	"github.com/kumahq/kuma-net/iptables/consts"
	"github.com/kumahq/kuma-net/test/blackbox_tests"
	"github.com/kumahq/kuma-net/test/framework/ip"
	"github.com/kumahq/kuma-net/test/framework/netns"
	"github.com/kumahq/kuma-net/test/framework/socket"
	"github.com/kumahq/kuma-net/test/framework/tcp"
)

var _ = Describe("Outbound IPv4 TCP traffic to any address:port", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to outbound port",
		func(serverPort, randomPort uint16) {
			// given
			address := fmt.Sprintf(":%d", serverPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Outbound: config.TrafficFlow{
						Enabled: true,
						Port:    serverPort,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				RuntimeOutput: ioutil.Discard,
			}

			tcpReadyC, tcpErrC := tcp.UnsafeStartTCPServer(
				ns,
				address,
				tcp.ReplyWithOriginalDstIPv4,
				tcp.CloseConn,
			)
			Eventually(tcpReadyC).Should(BeClosed())
			Consistently(tcpErrC).ShouldNot(Receive())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Eventually(ns.UnsafeExec(func() {
				address := ip.GenRandomIPv4()

				Expect(tcp.DialIPWithPortAndGetReply(address, randomPort)).
					To(Equal(fmt.Sprintf("%s:%d", address, randomPort)))
			})).Should(BeClosed())

			// then
			Eventually(tcpErrC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			var lockedPorts []uint16

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(2, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %%d")
				entry := Entry(
					EntryDescription(desc),
					randomPorts[0],
					randomPorts[1],
				)
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv6 TCP traffic to any address:port", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().WithIPv6(true).Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to outbound port",
		func(serverPort, randomPort uint16) {
			// given
			address := fmt.Sprintf(":%d", serverPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Outbound: config.TrafficFlow{
						Enabled: true,
						Port:    serverPort,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				IPv6:          true,
				RuntimeOutput: ioutil.Discard,
			}

			tcpReadyC, tcpErrC := tcp.UnsafeStartTCPServer(
				ns,
				address,
				tcp.ReplyWithOriginalDstIPv6,
				tcp.CloseConn,
			)
			Eventually(tcpReadyC).Should(BeClosed())
			Consistently(tcpErrC).ShouldNot(Receive())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Eventually(ns.UnsafeExec(func() {
				address := ip.GenRandomIPv6()

				Expect(tcp.DialIPWithPortAndGetReply(address, randomPort)).
					To(Equal(fmt.Sprintf("[%s]:%d", address, randomPort)))
			})).Should(BeClosed())

			// then
			Eventually(tcpErrC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			var lockedPorts []uint16

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(2, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %%d")
				entry := Entry(
					EntryDescription(desc),
					randomPorts[0],
					randomPorts[1],
				)
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv4 TCP traffic to any address:port", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should not be redirected to outbound port",
		func(serverPort, randomPort uint16) {
			// given
			address := fmt.Sprintf(":%d", randomPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Outbound: config.TrafficFlow{
						Enabled: false,
						Port:    serverPort,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				RuntimeOutput: ioutil.Discard,
			}

			tcpReadyC, tcpErrC := tcp.UnsafeStartTCPServer(
				ns,
				address,
				tcp.ReplyWith("randomPort"),
				tcp.CloseConn,
			)
			Eventually(tcpReadyC).Should(BeClosed())
			Consistently(tcpErrC).ShouldNot(Receive())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Eventually(ns.UnsafeExec(func() {
				Expect(tcp.DialIPWithPortAndGetReply(net.ParseIP(consts.LocalhostIPv4), randomPort)).
					To(Equal("randomPort"))
			})).Should(BeClosed())

			// then
			Eventually(tcpErrC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			var lockedPorts []uint16

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(2, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %%d")
				entry := Entry(
					EntryDescription(desc),
					randomPorts[0],
					randomPorts[1],
				)
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv6 TCP traffic to any address:port", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().WithIPv6(true).Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should not be redirected to outbound port",
		func(serverPort, randomPort uint16) {
			// given
			address := fmt.Sprintf(":%d", randomPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Outbound: config.TrafficFlow{
						Enabled: false,
						Port:    serverPort,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				IPv6:          true,
				RuntimeOutput: ioutil.Discard,
			}

			tcpReadyC, tcpErrC := tcp.UnsafeStartTCPServer(
				ns,
				address,
				tcp.ReplyWith("randomPort"),
				tcp.CloseConn,
			)
			Eventually(tcpReadyC).Should(BeClosed())
			Consistently(tcpErrC).ShouldNot(Receive())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Eventually(ns.UnsafeExec(func() {
				Expect(tcp.DialIPWithPortAndGetReply(net.ParseIP(consts.LocalhostIPv6), randomPort)).
					To(Equal("randomPort"))
			})).Should(BeClosed())

			// then
			Eventually(tcpErrC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			var lockedPorts []uint16

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(2, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %%d")
				entry := Entry(
					EntryDescription(desc),
					randomPorts[0],
					randomPorts[1],
				)
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})
