package rule_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder/rule"
)

type dummyMatch string

func (m dummyMatch) String() string {
	return string(m)
}

var _ = Describe("RuleBuilder", func() {
	It("should generate valid --append rule", func() {
		// given
		targetReturn := func() []string {
			return []string{"RETURN"}
		}

		matchProtocol := dummyMatch("--protocol tcp --match tcp --destination-port 2345")
		matchOwner := dummyMatch("--match owner --uid-owner 1234")

		// when
		got := rule.Match(matchProtocol, matchOwner).
			Chain("BAZ_BAR_FOO").
			Kind(rule.Append).
			Then(targetReturn).
			Build()

		// then
		want := fmt.Sprintf("--append BAZ_BAR_FOO %s %s --jump RETURN", matchProtocol, matchOwner)
		Expect(got).To(Equal(want))
	})
})
