package target_test

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kumahq/kuma-net/iptables/builder/target"
)

var _ = Describe("JoinUints16", func() {
	DescribeTable("should generate string with combined uint16 parameters",
		func(genParams func() ([]uint16, string, string)) {
			// given
			numbers, separator, want := genParams()

			// when
			got := target.JoinUints16(numbers, separator)

			// then
			Expect(got).To(Equal(want))
		},
		Entry(
			nil,
			func() ([]uint16, string, string) {
				return []uint16{1, 2, 3, 15, 7777, 8888, math.MaxUint16},
					" ",
					fmt.Sprintf("1 2 3 15 7777 8888 %d", math.MaxUint16)
			},
		),
		Entry(
			nil,
			func() ([]uint16, string, string) {
				separators := []string{" ", "\n", "\t", "0", ":", ",", ".", ", "}
				separator := separators[rand.Intn(len(separators)-1)]

				howManyPorts := rand.Intn(10) + 5
				var ports []uint16
				for i := 0; i < howManyPorts; i++ {
					ports = append(ports, uint16(rand.Intn(math.MaxUint16-1)+1))
				}

				var want []string
				for _, port := range ports {
					want = append(want, strconv.Itoa(int(port)))
				}

				return ports, separator, strings.Join(want, separator)
			},
		))
})
