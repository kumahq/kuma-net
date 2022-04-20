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
	"github.com/kumahq/kuma-net/test/framework/ip"
	"github.com/kumahq/kuma-net/test/framework/netns"
	"github.com/kumahq/kuma-net/test/framework/socket"
)

var _ = Describe("Outbound DNS traffic to port 53", func() {
	var err error
	var ns *netns.NetNS
	howManyPortsToTest := uint(50)

	BeforeEach(func() {
		DeferCleanup(ns.Cleanup)

		ns, err = netns.NewNetNS().Build()
		Expect(err).To(BeNil())
	})

	DescribeTable("should be redirected to provided port",
		func(port uint16) {
			// given
			udpReadyC, udpErrC := ns.StartUDPServer(fmt.Sprintf("127.0.0.1:%d", port), 0)
			Consistently(udpErrC).ShouldNot(Receive())
			Eventually(udpReadyC).Should(BeClosed())

			// when
			Eventually(ns.Exec(func() {
				cfg := config.DefaultConfig()
				cfg.Redirect.DNS.Enabled = true
				cfg.Redirect.DNS.Port = port
				cfg.Output = ioutil.Discard

				Expect(builder.RestoreIPTables(cfg)).Error().To(Succeed())
			})).Should(BeClosed())

			// then
			Eventually(ns.Exec(func() {
				udpAddress := ip.GenRandomUDPAddress(consts.DNSPort)

				socket, err := net.DialUDP("udp", nil, udpAddress)
				Expect(err).To(Succeed())
				defer socket.Close()

				sendData := []byte(udpAddress.String())
				Expect(socket.Write(sendData)).Error().To(Succeed())

				buf := make([]byte, 1024)
				n, err := socket.Read(buf)
				Expect(err).To(Succeed())

				Expect(buf[:n]).To(Equal(sendData))
			})).Should(BeClosed())

			Eventually(udpErrC).Should(BeClosed())
		},
		func() []TableEntry {
			ports := socket.GenerateRandomPorts(howManyPortsToTest, consts.DNSPort)

			var entries []TableEntry
			for port := range ports {
				entries = append(entries, Entry(EntryDescription("to port %d, from port 53"), port))
			}

			return entries
		}(),
	)
})
