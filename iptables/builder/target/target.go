package target

import (
	"fmt"
	"strings"
)

type RedirectParameter []string

func (o RedirectParameter) String() string {
	return strings.Join(o, " ")
}

func Ports(ports ...uint16) RedirectParameter {
	return RedirectParameter{"REDIRECT", "--to-ports", JoinUints16(ports, ",")}
}

func To(target fmt.Stringer) RedirectParameter {
	return RedirectParameter{target.String()}
}

type Func func() []string

func Return() []string {
	return []string{"RETURN"}
}

// Redirect will return function which should return slice of strings from combined
// RedirectParameters
// As it requires at least one parameter, it was divided in parameters for "parameter" and the rest
// "parameters"
func Redirect(parameter RedirectParameter, parameters ...RedirectParameter) Func {
	return func() []string {
		var params []string

		for _, param := range append([]RedirectParameter{parameter}, parameters...) {
			params = append(params, param.String())
		}

		return params
	}
}
