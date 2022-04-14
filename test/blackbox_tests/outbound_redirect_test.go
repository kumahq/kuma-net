package blackbox_tests_test

import (
	"fmt"
	"io/ioutil"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder"
	"github.com/kumahq/kuma-net/iptables/config"
	"github.com/kumahq/kuma-net/test/framework/ip"
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
			// given
			done := make(chan struct{})

			// when
			address := fmt.Sprintf(":%d", tcpServerPort)
			ready, err := ns.StartTCPServer(address, func() error {
				cfg := config.DefaultConfig()
				cfg.Redirect.Outbound.Port = tcpServerPort
				cfg.Output = ioutil.Discard

				_, err := builder.RestoreIPTables(cfg)

				return err
			})
			Eventually(ready).Should(BeClosed())

			// then
			go func() {
				defer GinkgoRecover()
				defer close(done)

				runtime.LockOSThread()
				defer runtime.UnlockOSThread()

				Expect(ns.Set()).To(Succeed())
				defer ns.Unset()

				address := ip.GenRandomIPv4()

				Expect(tcp.DialAndGetReply(address, port)).
					To(Equal([]byte(fmt.Sprintf("%s:%d", address, port))))
			}()

			Eventually(done).Should(BeClosed())

			Consistently(err).ShouldNot(Receive())
		},
		func() []TableEntry {
			ports := socket.GenerateRandomPorts(howManyPortsToTest, tcpServerPort)
			desc := fmt.Sprintf("to port %%d, from port %d", tcpServerPort)

			var entries []TableEntry
			for port := range ports {
				entries = append(entries, Entry(EntryDescription(desc), port))
			}

			return entries
		}(),
	)
})
