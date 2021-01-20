// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"strings"

	"github.com/qioalice/ekago/v2/ekastr"
)

type (
	/*
	interpolator is a helper tool to interpolate a string.
	It's a worker that takes a values from args by their keys and substitute
	them to the rem (it's a string as []byte) instead of the same name
	interpolation verbs using provided strings.Builder to accumulate result
	and do interpolation the most efficient way.
	*/
	interpolator struct {
		args    Args
		builder strings.Builder
		rem     []byte
	}
)

/*
cbFoundVerb is a callback for ekastr.Interpolate() function,
that is called when a interpolation verb is found.

Writes corresponding argument from args if it exists,
or keeps verb untouched and writes it as just text.
*/
func (ir *interpolator) cbFoundVerb(p []byte) {
	// guarantees that p's len >= 4
	if arg, found := ir.args[ekastr.B2S(p[2:len(p)-2])]; found {
		_, _ = ir.builder.WriteString(ekastr.ToString(arg))
	} else {
		_, _ = ir.builder.Write(p)
	}
}

/*
cbFoundText is a callback for ekastr.Interpolate() function,
that is called when a just text part found (not an interpolation verb).

Just writes it to the strings.Builder.
*/
func (ir *interpolator) cbFoundText(p []byte) {
	_, _ = ir.builder.Write(p) // always returns not-nil error
}

/*
interpolate does interpolation of translation phrase and its provided arguments.

Returns an interpolated string,
or original string if either there was no arguments or its a nil.

Ignores unused arguments.
Verbs that doesn't have associated argument remains as is.

Verbs must be in the format: "{{<name>}}",
<name> is key from Args.
*/
func (ir *interpolator) interpolate() string {
	ekastr.Interpolateb(ir.rem, ir.cbFoundVerb, ir.cbFoundText)
	return ir.builder.String()
}

/*
newInterpolator is a interpolator constructor.
Transforms phrase to []byte w/ no-copy and grows builder's internal buffer
to the phrase's len + 128 bytes.
*/
func newInterpolator(phrase string, args Args) *interpolator {
	i := &interpolator{
		args: args,
		rem:  ekastr.S2B(phrase),
	}
	i.builder.Grow(len(i.rem) + 128)
	return i
}
