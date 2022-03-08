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
		buildTableEntries := func() []TableEntry {
			ports := socket.GenerateRandomPorts(
				config.AmountOfPortsToTest,
				config.TCPServer.Port,
			)

			desc := fmt.Sprintf(
				"to port %d, from port %%d",
				config.TCPServer.Port,
			)

			var entries []TableEntry
			for port := range ports {
				entries = append(entries, Entry(EntryDescription(desc), port))
			}

			return entries
		}

		BeforeEach(func() {
			server, err := tcp.NewServer().
				WithPort(config.TCPServer.Port).
				WithHost(config.TCPServer.Host).
				WithConnHandler(tcp.ReplyWithOriginalDst).
				Listen()
			Expect(err).To(BeNil())
			DeferCleanup(server.Close)
		})

		DescribeTable("should be redirected",
			func(port uint16) {
				// given
				addr, err := tcp.ResolveAddress(config.TCPServer.Host, port)
				Expect(err).To(BeNil())

				// As we expect traffic to all ports to be redirected to one,
				// defined port, we are creating TCP server which will be
				// listening on the port on which we are trying to establish,
				// the connection with the expectancy that it will never get our
				// packets, as they should be redirected to the different port
				server, err := tcp.NewServer().
					WithAddress(addr).
					WithConnHandler(tcp.UnexpectedConn).
					// As this server should never receive any connection,
					// and as we are blocking goroutine inside the Listen()
					// waiting for connection, it will return error as we are
					// closing the connection at the end of the test.
					// It's fine, and we can safely ignore this error
					WithConnErrHandler(tcp.IgnoreUseClosedNetworkConnection).
					Listen()
				Expect(err).To(BeNil())
				DeferCleanup(server.Close)

				// when
				reply, err := tcp.NewClient().
					WithAddress(addr).
					DialAndGetReply(tcp.ReadTCPAddr)
				Expect(err).To(BeNil())

				// then
				Expect(reply).To(Equal(addr))
			},
			buildTableEntries(),
		)
	})
}
