package blackbox_tests_test

import (
	"fmt"
	"io/ioutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder"
	"github.com/kumahq/kuma-net/iptables/config"
	"github.com/kumahq/kuma-net/iptables/consts"
	"github.com/kumahq/kuma-net/test/framework/netns"
	"github.com/kumahq/kuma-net/test/framework/socket"
	"github.com/kumahq/kuma-net/test/framework/udp"
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
						Enabled: true,
						Port:    randomPort,
					},
				},
				RuntimeOutput: ioutil.Discard,
			}
			serverAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv4, randomPort)

			readyC, errC := udp.UnsafeStartUDPServer(ns, serverAddress, 0)
			Consistently(errC).ShouldNot(Receive())
			Eventually(readyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExec(func() {
				Expect(udp.DialWithHelloMsgAndGetReply(address, address)).
					To(Equal(address.String()))
			})).Should(BeClosed())

			// then
			Eventually(errC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < 50; i++ {
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
						Enabled: true,
						Port:    randomPort,
					},
				},
				IPv6:          true,
				RuntimeOutput: ioutil.Discard,
			}
			serverAddress := fmt.Sprintf("%s:%d", consts.LocalhostIPv6, randomPort)

			readyC, errC := udp.UnsafeStartUDPServer(ns, serverAddress, 0)
			Consistently(errC).ShouldNot(Receive())
			Eventually(readyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			Eventually(ns.UnsafeExec(func() {
				Expect(udp.DialWithHelloMsgAndGetReply(address, address)).
					To(Equal(address.String()))
			})).Should(BeClosed())

			// then
			Eventually(errC).Should(BeClosed())
		},
		func() []TableEntry {
			var entries []TableEntry
			lockedPorts := []uint16{consts.DNSPort}

			for i := 0; i < 50; i++ {
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
