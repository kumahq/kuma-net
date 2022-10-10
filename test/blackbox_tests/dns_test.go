package blackbox_tests_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder"
	"github.com/kumahq/kuma-net/iptables/consts"
	"github.com/kumahq/kuma-net/test/blackbox_tests"
	"github.com/kumahq/kuma-net/test/framework/netns"
	"github.com/kumahq/kuma-net/test/framework/socket"
	"github.com/kumahq/kuma-net/test/framework/syscall"
	"github.com/kumahq/kuma-net/test/framework/sysctl"
	"github.com/kumahq/kuma-net/test/framework/tcp"
	"github.com/kumahq/kuma-net/test/framework/udp"
	"github.com/kumahq/kuma-net/transparent-proxy/config"
)

var _ = Describe("Outbound IPv4 DNS/UDP traffic to port 53", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(randomPort uint16) {
			// given
			address := udp.GenRandomAddressIPv4(consts.DNSPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:    true,
						Port:       randomPort,
						CaptureAll: true,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
					Outbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				RuntimeStdout: ioutil.Discard,
			}
			serverAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv4, randomPort)

			readyC, errC := udp.UnsafeStartUDPServer(ns, serverAddress, udp.ReplyWithReceivedMsg)
			Consistently(errC).ShouldNot(Receive())
			Eventually(readyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExec(func() {
				Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(address, address)).
					To(Equal(address.String()))
			})).Should(BeClosed())

			// then
			Consistently(errC).ShouldNot(Receive())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(1, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %d", consts.DNSPort)
				entry := Entry(EntryDescription(desc), randomPorts[0])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv4 DNS/UDP traffic to port 53", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port except for traffic excluded by uid",
		func(randomPort uint16) {
			dnsUserUid := uintptr(4201) // see /.github/workflows/tests.yaml:76
			// given
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:    true,
						Port:       randomPort,
						CaptureAll: true,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
					Outbound: config.TrafficFlow{
						Enabled: true,
						ExcludePortsForUIDs: []config.UIDsToPorts{{
							UIDs:     config.ValueOrRangeList(strconv.Itoa(int(dnsUserUid))),
							Ports:    config.ValueOrRangeList(strconv.Itoa(int(consts.DNSPort))),
							Protocol: "udp",
						}},
					},
				},
				RuntimeStdout: ioutil.Discard,
			}
			originalAddress := &net.UDPAddr{IP: net.ParseIP(consts.LocalhostIPv4), Port: int(consts.DNSPort)}
			redirectedToAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv4, randomPort)

			redirectedC, redirectedErr := udp.UnsafeStartUDPServer(ns, redirectedToAddress, udp.ReplyWithReceivedMsg)
			Consistently(redirectedErr).ShouldNot(Receive())
			Eventually(redirectedC).Should(BeClosed())

			originalC, originalErr := udp.UnsafeStartUDPServer(ns, originalAddress.String(), udp.ReplyWithMsg("excluded"))
			Consistently(originalErr).ShouldNot(Receive())
			Eventually(originalC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExec(func() {
				Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(originalAddress, originalAddress)).
					To(Equal(originalAddress.String()))
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExecInLoop(1, 0, func() {
				Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(originalAddress, originalAddress)).
					To(Equal("excluded"))
			}, syscall.SetUID(dnsUserUid))).Should(BeClosed())

			// then
			Consistently(redirectedErr).ShouldNot(Receive())
			Consistently(originalErr).ShouldNot(Receive())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(2, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %d", consts.DNSPort)
				entry := Entry(EntryDescription(desc), randomPorts[0])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv6 DNS/UDP traffic to port 53", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().WithIPv6(true).Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port except for traffic excluded by uid",
		func(randomPort uint16) {
			// given
			dnsUserUid := uintptr(4201) // see /.github/workflows/tests.yaml:76
			address := udp.GenRandomAddressIPv6(consts.DNSPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:    true,
						Port:       randomPort,
						CaptureAll: true,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
					Outbound: config.TrafficFlow{
						Enabled: true,
						ExcludePortsForUIDs: []config.UIDsToPorts{{
							UIDs:     config.ValueOrRangeList(strconv.Itoa(int(dnsUserUid))),
							Ports:    config.ValueOrRangeList(strconv.Itoa(int(consts.DNSPort))),
							Protocol: "udp",
						}},
					},
				},
				IPv6:          true,
				RuntimeStdout: ioutil.Discard,
			}
			redirectedAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv6, randomPort)
			originalAddress := &net.UDPAddr{IP: net.ParseIP(consts.LocalhostIPv6), Port: int(consts.DNSPort)}

			redirectedC, redirectedErr := udp.UnsafeStartUDPServer(ns, redirectedAddress, udp.ReplyWithReceivedMsg)
			Consistently(redirectedErr).ShouldNot(Receive())
			Eventually(redirectedC).Should(BeClosed())

			originalC, originalErr := udp.UnsafeStartUDPServer(ns, originalAddress.String(), udp.ReplyWithMsg("excluded"))
			Consistently(originalErr).ShouldNot(Receive())
			Eventually(originalC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExec(func() {
				Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(address, address)).
					To(Equal(address.String()))
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExecInLoop(1, 0, func() {
				Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(originalAddress, originalAddress)).
					To(Equal("excluded"))
			}, syscall.SetUID(dnsUserUid))).Should(BeClosed())

			// then
			Consistently(redirectedErr).ShouldNot(Receive())
			Consistently(originalErr).ShouldNot(Receive())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(1, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %d", consts.DNSPort)
				entry := Entry(EntryDescription(desc), randomPorts[0])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv4 DNS/TCP traffic to port 53", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(dnsPort, outboundPort uint16) {
			// given
			address := tcp.GenRandomAddressIPv4(consts.DNSPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:    true,
						Port:       dnsPort,
						CaptureAll: true,
					},
					Outbound: config.TrafficFlow{
						Port:    outboundPort,
						Enabled: true,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				RuntimeStdout: ioutil.Discard,
			}
			serverAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv4, dnsPort)

			readyC, errC := tcp.UnsafeStartTCPServer(
				ns,
				serverAddress,
				tcp.ReplyWithOriginalDstIPv4,
				tcp.CloseConn,
			)
			Consistently(errC).ShouldNot(Receive())
			Eventually(readyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExec(func() {
				Expect(tcp.DialTCPAddrAndGetReply(address)).To(Equal(address.String()))
			})).Should(BeClosed())

			// then
			Eventually(errC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				// We are drawing two ports instead of one as the first one will be used
				// to expose TCP server inside the namespace, which will be pretending
				// a DNS server which should intercept all DNS traffic on port TCP#53,
				// and the second one will be set as an outbound redirection port,
				// which wound intercept the packet if no DNS redirection would be set,
				// and we don't want them to be the same
				randomPorts := socket.GenerateRandomPortsSlice(2, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf(
					"to port %d, from port %d",
					randomPorts[0],
					consts.DNSPort,
				)
				entry := Entry(EntryDescription(desc), randomPorts[0], randomPorts[1])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv6 DNS/UDP traffic to port 53", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().WithIPv6(true).Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(randomPort uint16) {
			// given
			address := udp.GenRandomAddressIPv6(consts.DNSPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:    true,
						Port:       randomPort,
						CaptureAll: true,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
					Outbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				IPv6:          true,
				RuntimeStdout: ioutil.Discard,
			}
			serverAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv6, randomPort)

			readyC, errC := udp.UnsafeStartUDPServer(ns, serverAddress, udp.ReplyWithReceivedMsg)
			Consistently(errC).ShouldNot(Receive())
			Eventually(readyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExec(func() {
				Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(address, address)).
					To(Equal(address.String()))
			})).Should(BeClosed())

			// then
			Consistently(errC).ShouldNot(Receive())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(1, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %d", consts.DNSPort)
				entry := Entry(EntryDescription(desc), randomPorts[0])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv6 DNS/TCP traffic to port 53", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().WithIPv6(true).Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(dnsPort, outboundPort uint16) {
			// given
			address := tcp.GenRandomAddressIPv6(consts.DNSPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:    true,
						Port:       dnsPort,
						CaptureAll: true,
					},
					Outbound: config.TrafficFlow{
						Port:    outboundPort,
						Enabled: true,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				IPv6:          true,
				RuntimeStdout: ioutil.Discard,
			}
			serverAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv6, dnsPort)

			readyC, errC := tcp.UnsafeStartTCPServer(
				ns,
				serverAddress,
				tcp.ReplyWithOriginalDstIPv6,
				tcp.CloseConn,
			)
			Consistently(errC).ShouldNot(Receive())
			Eventually(readyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExec(func() {
				Expect(tcp.DialTCPAddrAndGetReply(address)).To(Equal(address.String()))
			})).Should(BeClosed())

			// then
			Eventually(errC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				// We are drawing two ports instead of one as the first one will be used
				// to expose TCP server inside the namespace, which will be pretending
				// a DNS server which should intercept all DNS traffic on port TCP#53,
				// and the second one will be set as an outbound redirection port,
				// which wound intercept the packet if no DNS redirection would be set,
				// and we don't want them to be the same
				randomPorts := socket.GenerateRandomPortsSlice(2, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf(
					"to port %d, from port %d",
					randomPorts[0],
					consts.DNSPort,
				)
				entry := Entry(EntryDescription(desc), randomPorts[0], randomPorts[1])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv4 DNS/UDP conntrack zone splitting", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().
			WithBeforeExecFuncs(sysctl.SetLocalPortRange(32768, 32770)).
			Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(port uint16) {
			// given
			uid := uintptr(5678)
			s1Address := fmt.Sprintf("%s:%d", ns.Veth().PeerAddress(), consts.DNSPort)
			s2Address := fmt.Sprintf("%s:%d", consts.LocalhostIPv4, port)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:            true,
						Port:               port,
						ConntrackZoneSplit: true,
						CaptureAll:         true,
					},
					Outbound: config.TrafficFlow{
						Enabled: true,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				Owner:         config.Owner{UID: strconv.Itoa(int(uid))},
				RuntimeStdout: ioutil.Discard,
			}
			want := map[string]uint{
				s1Address: blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				s2Address: blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
			}

			s1ReadyC, s1ErrC := udp.UnsafeStartUDPServer(
				ns,
				s1Address,
				udp.ReplyWithLocalAddr,
			)
			Consistently(s1ErrC).ShouldNot(Receive())
			Eventually(s1ReadyC).Should(BeClosed())

			s2ReadyC, s2ErrC := udp.UnsafeStartUDPServer(
				ns,
				s2Address,
				udp.ReplyWithLocalAddr,
				sysctl.SetUnprivilegedPortStart(0),
				syscall.SetUID(uid),
			)
			Consistently(s2ErrC).ShouldNot(Receive())
			Eventually(s2ReadyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			results := udp.NewResultMap()

			exec1ErrC := ns.UnsafeExecInLoop(
				blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				time.Millisecond,
				func() {
					Expect(udp.DialAddrAndIncreaseResultMap(s1Address, results)).To(Succeed())
				},
				syscall.SetUID(uid),
			)

			exec2ErrC := ns.UnsafeExecInLoop(
				blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				time.Millisecond,
				func() {
					Expect(udp.DialAddrAndIncreaseResultMap(s1Address, results)).To(Succeed())
				},
			)

			Consistently(exec1ErrC).ShouldNot(Receive())
			Consistently(exec2ErrC).ShouldNot(Receive())
			Eventually(exec1ErrC, blackbox_tests.DNSConntrackZoneSplittingTestTimeout).
				Should(BeClosed())
			Eventually(exec2ErrC, blackbox_tests.DNSConntrackZoneSplittingTestTimeout).
				Should(BeClosed())

			Expect(results.GetFinalResults()).To(BeEquivalentTo(want))
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				ports := socket.GenerateRandomPortsSlice(1, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, ports...)
				desc := fmt.Sprintf("to port %%d, from port %d", consts.DNSPort)
				entry := Entry(EntryDescription(desc), ports[0])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv6 DNS/UDP conntrack zone splitting", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().
			WithIPv6(true).
			WithBeforeExecFuncs(sysctl.SetLocalPortRange(32768, 32770)).
			Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(port uint16) {
			// given
			uid := uintptr(5678)
			s1Address := fmt.Sprintf("%s:%d", consts.LocalhostIPv6, consts.DNSPort)
			s2Address := fmt.Sprintf("%s:%d", consts.LocalhostIPv6, port)
			tproxyConfig := config.Config{

				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:            true,
						Port:               port,
						ConntrackZoneSplit: true,
						CaptureAll:         true,
					},
					Outbound: config.TrafficFlow{
						Enabled: true,
					},
					Inbound: config.TrafficFlow{
						Enabled: true,
					},
				},
				IPv6:          true,
				Owner:         config.Owner{UID: strconv.Itoa(int(uid))},
				RuntimeStdout: ioutil.Discard,
			}
			want := map[string]uint{
				s1Address: blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				s2Address: blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
			}

			s1ReadyC, s1ErrC := udp.UnsafeStartUDPServer(
				ns,
				s1Address,
				udp.ReplyWithLocalAddr,
			)
			Consistently(s1ErrC).ShouldNot(Receive())
			Eventually(s1ReadyC).Should(BeClosed())

			s2ReadyC, s2ErrC := udp.UnsafeStartUDPServer(
				ns,
				s2Address,
				udp.ReplyWithLocalAddr,
				sysctl.SetUnprivilegedPortStart(0),
				syscall.SetUID(uid),
			)
			Consistently(s2ErrC).ShouldNot(Receive())
			Eventually(s2ReadyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			results := udp.NewResultMap()

			exec1ErrC := ns.UnsafeExecInLoop(
				blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				time.Millisecond,
				func() {
					Expect(udp.DialAddrAndIncreaseResultMap(s1Address, results)).To(Succeed())
				},
				syscall.SetUID(uid),
			)

			exec2ErrC := ns.UnsafeExecInLoop(
				blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				time.Millisecond,
				func() {
					Expect(udp.DialAddrAndIncreaseResultMap(s1Address, results)).To(Succeed())
				},
			)

			Consistently(exec1ErrC).ShouldNot(Receive())
			Consistently(exec2ErrC).ShouldNot(Receive())
			Eventually(exec1ErrC, blackbox_tests.DNSConntrackZoneSplittingTestTimeout).
				Should(BeClosed())
			Eventually(exec2ErrC, blackbox_tests.DNSConntrackZoneSplittingTestTimeout).
				Should(BeClosed())

			Expect(results.GetFinalResults()).To(BeEquivalentTo(want))
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				ports := socket.GenerateRandomPortsSlice(1, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, ports...)
				desc := fmt.Sprintf("to port %%d, from port %d", consts.DNSPort)
				entry := Entry(EntryDescription(desc), ports[0])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv4 DNS/UDP traffic to port 53 only for addresses in configuration ", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(randomPort uint16) {
			// given
			dnsServers := getDnsServers("testdata/resolv4.conf", 2, false)
			randomAddressDnsRequest := udp.GenRandomAddressIPv4(consts.DNSPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:          true,
						CaptureAll:       false,
						Port:             randomPort,
						ResolvConfigPath: "testdata/resolv4.conf",
					},
				},
				RuntimeStdout: ioutil.Discard,
			}
			serverAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv4, randomPort)

			readyC, errC := udp.UnsafeStartUDPServer(ns, serverAddress, udp.ReplyWithReceivedMsg)
			Consistently(errC).ShouldNot(Receive())
			Eventually(readyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			for _, dnsServer := range dnsServers {
				Eventually(ns.UnsafeExec(func() {
					Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(dnsServer, dnsServer)).
						To(Equal(dnsServer.String()))
				})).Should(BeClosed())
			}

			// and do not redirect any dns request
			Eventually(ns.UnsafeExec(func() {
				Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(randomAddressDnsRequest, randomAddressDnsRequest))
			})).ShouldNot(BeClosed())

			// then
			Consistently(errC).ShouldNot(Receive())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(1, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %d", consts.DNSPort)
				entry := Entry(EntryDescription(desc), randomPorts[0])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv6 DNS/UDP traffic to port 53 only for addresses in configuration ", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().WithIPv6(true).Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(randomPort uint16) {
			// given
			dnsServers := getDnsServers("testdata/resolv6.conf", 2, true)
			randomAddressDnsRequest := udp.GenRandomAddressIPv6(consts.DNSPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:          true,
						CaptureAll:       false,
						Port:             randomPort,
						ResolvConfigPath: "testdata/resolv6.conf",
					},
				},
				RuntimeStdout: ioutil.Discard,
				IPv6:          true,
			}
			serverAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv6, randomPort)

			readyC, errC := udp.UnsafeStartUDPServer(ns, serverAddress, udp.ReplyWithReceivedMsg)
			Consistently(errC).ShouldNot(Receive())
			Eventually(readyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			for _, dnsServer := range dnsServers {
				Eventually(ns.UnsafeExec(func() {
					Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(dnsServer, dnsServer)).
						To(Equal(dnsServer.String()))
				})).Should(BeClosed())
			}

			// and do not redirect any dns request
			Eventually(ns.UnsafeExec(func() {
				Expect(udp.DialUDPAddrWithHelloMsgAndGetReply(randomAddressDnsRequest, randomAddressDnsRequest))
			})).ShouldNot(BeClosed())

			// then
			Consistently(errC).ShouldNot(Receive())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				randomPorts := socket.GenerateRandomPortsSlice(1, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, randomPorts...)
				desc := fmt.Sprintf("to port %%d, from port %d", consts.DNSPort)
				entry := Entry(EntryDescription(desc), randomPorts[0])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

var _ = Describe("Outbound IPv4 DNS/UDP conntrack zone splitting with specific IP", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		ns, err = netns.NewNetNSBuilder().
			WithBeforeExecFuncs(sysctl.SetLocalPortRange(32768, 32770)).
			Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(port uint16) {
			// given
			uid := uintptr(5678)
			dnsServers := getDnsServers("testdata/resolv4-conntrack.conf", 1, false)
			s1Address := fmt.Sprintf("%s:%d", dnsServers[0].IP.String(), consts.DNSPort)
			s2Address := fmt.Sprintf("%s:%d", consts.LocalhostIPv4, port)
			notRedirected := udp.GenRandomAddressIPv4(consts.DNSPort).AddrPort().String()
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled:            true,
						Port:               port,
						ConntrackZoneSplit: true,
						CaptureAll:         false,
						ResolvConfigPath:   "testdata/resolv4.conf",
					},
				},
				Owner:         config.Owner{UID: strconv.Itoa(int(uid))},
				RuntimeStdout: ioutil.Discard,
			}
			want := map[string]uint{
				s1Address: blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				s2Address: blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
			}

			s1ReadyC, s1ErrC := udp.UnsafeStartUDPServer(
				ns,
				s1Address,
				udp.ReplyWithLocalAddr,
			)
			Consistently(s1ErrC).ShouldNot(Receive())
			Eventually(s1ReadyC).Should(BeClosed())

			s2ReadyC, s2ErrC := udp.UnsafeStartUDPServer(
				ns,
				s2Address,
				udp.ReplyWithLocalAddr,
				sysctl.SetUnprivilegedPortStart(0),
				syscall.SetUID(uid),
			)
			Consistently(s2ErrC).ShouldNot(Receive())
			Eventually(s2ReadyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			results := udp.NewResultMap()

			exec1ErrC := ns.UnsafeExecInLoop(
				blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				time.Millisecond,
				func() {
					Expect(udp.DialAddrAndIncreaseResultMap(s1Address, results)).To(Succeed())
				},
				syscall.SetUID(uid),
			)

			exec2ErrC := ns.UnsafeExecInLoop(
				blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				time.Millisecond,
				func() {
					Expect(udp.DialAddrAndIncreaseResultMap(s1Address, results)).To(Succeed())
				},
			)

			exec3ErrC := ns.UnsafeExecInLoop(
				blackbox_tests.DNSConntrackZoneSplittingStressCallsAmount,
				time.Millisecond,
				func() {
					Expect(udp.DialAddrAndIncreaseResultMap(notRedirected, results)).ToNot(Succeed())
				},
			)

			Consistently(exec1ErrC).ShouldNot(Receive())
			Consistently(exec2ErrC).ShouldNot(Receive())
			Consistently(exec3ErrC).ShouldNot(Receive())
			Eventually(exec1ErrC, blackbox_tests.DNSConntrackZoneSplittingTestTimeout).
				Should(BeClosed())
			Eventually(exec2ErrC, blackbox_tests.DNSConntrackZoneSplittingTestTimeout).
				Should(BeClosed())
			Eventually(exec3ErrC, blackbox_tests.DNSConntrackZoneSplittingTestTimeout).
				ShouldNot(BeClosed())

			Expect(results.GetFinalResults()).To(BeEquivalentTo(want))
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < blackbox_tests.TestCasesAmount; i++ {
				ports := socket.GenerateRandomPortsSlice(1, lockedPorts...)
				// This gives us more entropy as all generated ports will be
				// different from each other
				lockedPorts = append(lockedPorts, ports...)
				desc := fmt.Sprintf("to port %%d, from port %d", consts.DNSPort)
				entry := Entry(EntryDescription(desc), ports[0])
				entries = append(entries, entry)
			}

			return entries
		}(),
	)
})

func getDnsServers(configPath string, expectedServers int, isIpv6 bool) []*net.UDPAddr {
	var dnsServers []*net.UDPAddr
	configPath, err := filepath.Abs(configPath)
	Expect(err).ToNot(HaveOccurred())

	ipv4, ipv6, err := builder.GetDnsServers(configPath)
	Expect(err).ToNot(HaveOccurred())

	dnsAddresses := ipv4
	if isIpv6 {
		dnsAddresses = ipv6
	}
	Expect(dnsAddresses).To(HaveLen(expectedServers))
	for _, dnsServer := range dnsAddresses {
		dnsServers = append(dnsServers, &net.UDPAddr{
			IP:   net.ParseIP(dnsServer),
			Port: int(consts.DNSPort),
		})
	}
	return dnsServers
}
