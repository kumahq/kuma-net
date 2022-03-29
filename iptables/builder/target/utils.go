package target

import (
	"strconv"
	"strings"
)

func JoinUints16(numbers []uint16, sep string) string {
	var numberAsStrings []string
	for _, number := range numbers {
		numberAsStrings = append(numberAsStrings, strconv.Itoa(int(number)))
	}

	return strings.Join(numberAsStrings, sep)
}
