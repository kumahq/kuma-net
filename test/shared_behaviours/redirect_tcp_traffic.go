package shared_behaviours

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/test/framework"
	"github.com/kumahq/kuma-net/test/framework/socket"
	"github.com/kumahq/kuma-net/test/framework/tcp"
)

func RedirectTCPTrafficDefault(config *framework.ConfigRedirectTCPTrafficDefault) {
	Describe("Inbound TCP traffic from all ports", func() {
		BeforeEach(func() {
			server, err := tcp.NewServer().
				WithPort(config.TCPServer.Port).
				WithHost(config.TCPServer.Host).
				WithConnectionHandler(tcp.ReplyWithOriginalDestination).
				Listen()
			Expect(err).To(BeNil())

			DeferCleanup(server.Close)
		})

		DescribeTable("should be redirected",
			func(port uint16) {
				client := tcp.NewClient().
					WithHost(config.TCPServer.Host).
					WithPort(port)

				Expect(client.DialAndWaitForStringReply(tcp.ReadBytes)).
					To(Equal(client.Address().String()))
			},
			EntryDescription(fmt.Sprintf(
				"to port %d, from port %%d",
				config.TCPServer.Port,
			)),
			func() []TableEntry {
				var entries []TableEntry

				ports := socket.GenerateRandomPorts(
					config.AmountOfPortsToTest,
					config.TCPServer.Port,
				)

				for port := range ports {
					entries = append(entries, Entry(nil, port))
				}

				return entries
			}(),
		)
	})
}
