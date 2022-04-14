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
	howManyPortsToTest := uint(50)
	tcpServerPort := socket.GenFreeRandomPort()

	BeforeEach(func() {
		ns, err = netns.NewNetNS().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to outbound port",
		func(port uint16) {
			// when
			address := fmt.Sprintf(":%d", tcpServerPort)
			ready, err := ns.StartTCPServer(address, func() error {
				cfg := config.DefaultConfig()
				cfg.Redirect.Inbound.Port = tcpServerPort
				cfg.Output = ioutil.Discard

				_, err := builder.RestoreIPTables(cfg)

				return err
			})

			Eventually(ready).Should(BeClosed())

			// then
			Expect(tcp.DialAndGetReply(ns.Veth().PeerAddress(), port)).
				To(Equal([]byte(fmt.Sprintf("%s:%d", ns.Veth().PeerAddress(), port))))

			Consistently(err).ShouldNot(Receive())
		},
		func() []TableEntry {
			ports := socket.GenerateRandomPorts(howManyPortsToTest, tcpServerPort)
			desc := fmt.Sprintf("to port %d, from port %%d", tcpServerPort)

			var entries []TableEntry
			for port := range ports {
				entries = append(entries, Entry(EntryDescription(desc), port))
			}

			return entries
		}(),
	)
})
