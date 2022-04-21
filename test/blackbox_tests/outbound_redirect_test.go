package blackbox_tests_test

import (
	"fmt"
	"io/ioutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder"
	"github.com/kumahq/kuma-net/iptables/config"
	"github.com/kumahq/kuma-net/test/framework/ip"
	"github.com/kumahq/kuma-net/test/framework/netns"
	"github.com/kumahq/kuma-net/test/framework/socket"
	"github.com/kumahq/kuma-net/test/framework/tcp"
)

var _ = Describe("Outbound TCP traffic to any address:port", func() {
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
					Outbound: config.TrafficFlow{
						Port: tcpServerPort,
					},
				},
				RuntimeOutput: ioutil.Discard,
			}

			tcpReadyC, tcpErrC := ns.StartTCPServer(address, tcp.ReplyWithOriginalDst)
			Eventually(tcpReadyC).Should(BeClosed())

			// when
			Eventually(ns.UnsafeExec(func() {
				Expect(builder.RestoreIPTables(tproxyConfig)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Eventually(ns.UnsafeExec(func() {
				address := ip.GenRandomIPv4().String()

				Expect(tcp.DialAndGetReply(address, port)).
					To(Equal(fmt.Sprintf("%s:%d", address, port)))
			})).Should(BeClosed())

			Consistently(tcpErrC).ShouldNot(Receive())
		},
		func() []TableEntry {
			ports := socket.GenerateRandomPorts(50, tcpServerPort)
			desc := fmt.Sprintf("to port %%d, from port %d", tcpServerPort)

			var entries []TableEntry
			for port := range ports {
				entries = append(entries, Entry(EntryDescription(desc), port))
			}

			return entries
		}(),
	)
})
