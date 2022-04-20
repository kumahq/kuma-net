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

		ns, err = netns.NewNetNS().Build()
		Expect(err).To(BeNil())
	})

	DescribeTable("should be redirected to outbound port",
		func(port uint16) {
			// given
			tcpReadyC, tcpErrC := ns.StartTCPServer(fmt.Sprintf(":%d", tcpServerPort))
			Eventually(tcpReadyC).Should(BeClosed())

			// when
			Eventually(ns.Exec(func() {
				cfg := config.DefaultConfig()
				cfg.Redirect.Inbound.Port = tcpServerPort
				cfg.Output = ioutil.Discard

				Expect(builder.RestoreIPTables(cfg)).Error().To(Succeed())
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
