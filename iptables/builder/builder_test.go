package builder_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder"
	"github.com/kumahq/kuma-net/iptables/builder/config"
)

var _ = Describe("IPTables Builder", func() {
	It("", func() {
		Expect(builder.Build(config.DefaultConfig())).To(BeEmpty())
	})
})
