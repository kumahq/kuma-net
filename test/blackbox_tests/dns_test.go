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

var _ = Describe("Outbound DNS/UDP traffic to port 53", func() {
	var err error
	var ns *netns.NetNS

	BeforeEach(func() {
		DeferCleanup(ns.Cleanup)

		ns, err = netns.NewNetNSBuilder().Build()
		Expect(err).To(BeNil())
	})

	DescribeTable("should be redirected to provided port",
		func(port uint16) {
			// given
			address := udp.GenRandomAddress(consts.DNSPort)
			tproxyConfig := config.Config{
				Redirect: config.Redirect{
					DNS: config.DNS{
						Enabled: true,
						Port:    port,
					},
				},
				RuntimeOutput: ioutil.Discard,
			}

			readyC, errC := ns.StartUDPServer(fmt.Sprintf("127.0.0.1:%d", port), 0)
			Consistently(errC).ShouldNot(Receive())
			Eventually(readyC).Should(BeClosed())

			// when
			Eventually(ns.Exec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// and
			Eventually(ns.Exec(func() {
				Expect(udp.DialWithHelloMsgAndGetReply(address, address)).
					To(Equal(address.String()))
			})).Should(BeClosed())

			// then
			Eventually(errC).Should(BeClosed())
		},
		func() []TableEntry {
			ports := socket.GenerateRandomPorts(50, consts.DNSPort)

			var entries []TableEntry
			for port := range ports {
				entries = append(entries, Entry(EntryDescription("to port %d, from port 53"), port))
			}

			return entries
		}(),
	)
})
