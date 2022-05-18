package blackbox_tests_test

import (
	"fmt"
	"io/ioutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder"
	"github.com/kumahq/kuma-net/iptables/config"
	"github.com/kumahq/kuma-net/test/blackbox_tests"
	"github.com/kumahq/kuma-net/test/framework/netns"
	"github.com/kumahq/kuma-net/test/framework/socket"
	"github.com/kumahq/kuma-net/test/framework/tcp"
)

var _ = Describe("Inbound IPv4 TCP traffic from any ports", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to the inbound_redirection port",
		func(serverPort, randomPort uint16) {
			// given
			tcpServerAddress := fmt.Sprintf(":%d", serverPort)
			peerAddress := ns.Veth().PeerAddress()
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Inbound: config.TrafficFlow{
						Port: serverPort,
					},
				},
				RuntimeOutput: ioutil.Discard,
			}

			tcpReadyC, tcpErrC := tcp.UnsafeStartTCPServer(
				ns,
				tcpServerAddress,
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
			Expect(tcp.DialIPWithPortAndGetReply(peerAddress, randomPort)).
				To(Equal(fmt.Sprintf("%s:%d", peerAddress, randomPort)))

			// and, then
			Consistently(tcpErrC).ShouldNot(Receive())
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

var _ = Describe("Inbound IPv6 TCP traffic from any ports", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().WithIPv6(true).Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to the inbound_redirection port",
		func(serverPort, randomPort uint16) {
			// given
			tcpServerAddress := fmt.Sprintf(":%d", serverPort)
			peerAddress := ns.Veth().PeerAddress()
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Inbound: config.TrafficFlow{
						Port: serverPort,
					},
				},
				IPv6:          true,
				RuntimeOutput: ioutil.Discard,
			}

			tcpReadyC, tcpErrC := tcp.UnsafeStartTCPServer(
				ns,
				tcpServerAddress,
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
			Expect(tcp.DialIPWithPortAndGetReply(peerAddress, randomPort)).
				To(Equal(fmt.Sprintf("[%s]:%d", peerAddress, randomPort)))

			// and, then
			Consistently(tcpErrC).ShouldNot(Receive())
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

var _ = Describe("Inbound IPv4 TCP traffic from any ports except excluded ones", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to the inbound_redirection port",
		func(serverPort, randomPort, excludedPort uint16) {
			// given
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Inbound: config.TrafficFlow{
						Port:         serverPort,
						ExcludePorts: []uint16{excludedPort},
					},
				},
				RuntimeOutput: ioutil.Discard,
			}
			peerAddress := ns.Veth().PeerAddress()

			redirectReadyC, redirectErrC := tcp.UnsafeStartTCPServer(
				ns,
				fmt.Sprintf(":%d", serverPort),
				tcp.ReplyWithOriginalDstIPv4,
				tcp.CloseConn,
			)
			Eventually(redirectReadyC).Should(BeClosed())
			Consistently(redirectErrC).ShouldNot(Receive())

			excludedReadyC, excludedErrC := tcp.UnsafeStartTCPServer(
				ns,
				fmt.Sprintf(":%d", excludedPort),
				tcp.ReplyWith("foobar"),
				tcp.CloseConn,
			)
			Eventually(excludedReadyC).Should(BeClosed())
			Consistently(excludedErrC).ShouldNot(Receive())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Expect(tcp.DialIPWithPortAndGetReply(peerAddress, excludedPort)).To(Equal("foobar"))

			// then
			Expect(tcp.DialIPWithPortAndGetReply(peerAddress, randomPort)).
				To(Equal(fmt.Sprintf("%s:%d", peerAddress, randomPort)))

			// then
			Eventually(redirectErrC).Should(BeClosed())
			Eventually(excludedErrC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			var lockedPorts []uint16

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(3, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %%d (excluded: %%d)")
				entry := Entry(
					EntryDescription(desc),
					randomPorts[0],
					randomPorts[1],
					randomPorts[2],
				)
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Inbound IPv6 TCP traffic from any ports except excluded ones", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().WithIPv6(true).Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to the inbound_redirection port",
		func(serverPort, randomPort, excludedPort uint16) {
			// given
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					Inbound: config.TrafficFlow{
						Port:         serverPort,
						ExcludePorts: []uint16{excludedPort},
					},
				},
				IPv6:          true,
				RuntimeOutput: ioutil.Discard,
			}
			peerAddress := ns.Veth().PeerAddress()

			redirectReadyC, redirectErrC := tcp.UnsafeStartTCPServer(
				ns,
				fmt.Sprintf(":%d", serverPort),
				tcp.ReplyWithOriginalDstIPv6,
				tcp.CloseConn,
			)
			Eventually(redirectReadyC).Should(BeClosed())
			Consistently(redirectErrC).ShouldNot(Receive())

			excludedReadyC, excludedErrC := tcp.UnsafeStartTCPServer(
				ns,
				fmt.Sprintf(":%d", excludedPort),
				tcp.ReplyWith("foobar"),
				tcp.CloseConn,
			)
			Eventually(excludedReadyC).Should(BeClosed())
			Consistently(excludedErrC).ShouldNot(Receive())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Expect(tcp.DialIPWithPortAndGetReply(peerAddress, excludedPort)).To(Equal("foobar"))

			// then
			Expect(tcp.DialIPWithPortAndGetReply(peerAddress, randomPort)).
				To(Equal(fmt.Sprintf("[%s]:%d", peerAddress, randomPort)))

			// then
			Eventually(redirectErrC).Should(BeClosed())
			Eventually(excludedErrC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			var lockedPorts []uint16

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(3, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %%d (excluded: %%d)")
				entry := Entry(
					EntryDescription(desc),
					randomPorts[0],
					randomPorts[1],
					randomPorts[2],
				)
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})
