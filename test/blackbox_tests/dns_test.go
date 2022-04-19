package blackbox_tests_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"runtime"

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
		ns, err = netns.NewNetNS().Build()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(ns.Cleanup()).To(Succeed())
	})

	DescribeTable("should be redirected to provided port",
		func(port uint16) {
			// given
			done := make(chan struct{})

			// when
			ready, err := ns.StartUDPServer(fmt.Sprintf("127.0.0.1:%d", port), 0,
				func() error {
					cfg := config.DefaultConfig()
					cfg.Redirect.DNS.Enabled = true
					cfg.Redirect.DNS.Port = port
					cfg.Output = ioutil.Discard

					_, err := builder.RestoreIPTables(cfg)

					return err
				})

			Consistently(err).ShouldNot(Receive())
			Eventually(ready).Should(BeClosed())

			// then
			go func() {
				defer GinkgoRecover()
				defer close(done)

				runtime.LockOSThread()
				defer runtime.UnlockOSThread()

				Expect(ns.Set()).To(Succeed())
				defer ns.Unset()

				udpAddress := ip.GenRandomUDPAddress(consts.DNSPort)

				socket, err := net.DialUDP("udp", nil, udpAddress)
				Expect(err).To(BeNil())
				defer socket.Close()

				sendData := []byte(udpAddress.String())
				Expect(socket.Write(sendData)).Error().To(BeNil())

				buf := make([]byte, 1024)
				n, err := socket.Read(buf)
				Expect(err).To(Succeed())

				Expect(buf[:n]).To(Equal(sendData))
			}()

			Eventually(err).Should(BeClosed())
			Eventually(done).Should(BeClosed())
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
