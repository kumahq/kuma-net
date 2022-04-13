package blackbox_tests_test

import (
	"fmt"

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
	tcpServerPort := socket.GenerateRandomPort()
	peerAddress := "192.168.111.2"

	buildTableEntries := func() []TableEntry {
		ports := socket.GenerateRandomPorts(50, tcpServerPort)
		desc := fmt.Sprintf("to port %d, from port %%d", tcpServerPort)

		var entries []TableEntry
		for port := range ports {
			entries = append(entries, Entry(EntryDescription(desc), port))
		}

		return entries
	}

	BeforeEach(func() {
		ns, err = netns.NewNetNS(&netns.Config{
			Name: "foo",
			Veth: &netns.Veth{
				Name:        "main0",
				PeerName:    "peer0",
				Address:     "192.168.111.1/24",
				PeerAddress: fmt.Sprintf("%s/24", peerAddress),
			},
		})
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

				_, err := builder.RestoreIPTables(cfg)

				return err
			})

			Eventually(ready).Should(BeClosed())

			// then
			Expect(tcp.DialAndGetReply(peerAddress, port)).
				To(Equal([]byte(fmt.Sprintf("%s:%d", peerAddress, port))))

			Consistently(err).ShouldNot(Receive())
		},
		buildTableEntries(),
	)
})
