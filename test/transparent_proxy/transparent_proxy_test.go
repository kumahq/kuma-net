package transparent_proxy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

var _ = Describe("TransparentProxy", func() {
	// shared_behaviours.RedirectTCPTrafficDefault(&framework.ConfigRedirectTCPTrafficDefault{
	// 	AmountOfPortsToTest: 20,
	// 	TCPServer: &framework.ConfigTCPServer{
	// 		Host: "localhost",
	// 		Port: 7878,
	// 	},
	// })

	It("should do something", func() {
		la := netlink.NewLinkAttrs()
		la.Name = "veth0"

		veth := &netlink.Veth{
			LinkAttrs: la,
			PeerName:  "veth1",
		}

		Expect(netlink.LinkAdd(veth)).To(Succeed())

		DeferCleanup(netlink.LinkDel, veth)
	})
})
