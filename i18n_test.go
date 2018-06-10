// Copyright © 2018. All rights reserved.
// Author: Alice Qio.
// Contacts: <qioalice@gmail.com>.
// License: https://opensource.org/licenses/MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom
// the Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
// IN THE SOFTWARE.

package i18n

import "testing"
import "fmt"

// todo: Comment

//
func assertBool(t *testing.T, b bool, descr string) {
	if !b { t.Fatal(descr) }
}

//
func assertErr(t *testing.T, err error) {
	if err != nil { t.Fatal(err) }
}

//
func TestExtractArgs(t *testing.T) {
	m1 := Args{"Argskey1": 1234, "Argskey2": "1235", "Argskey3": nil}
	m2 := map[string]int{"SI1": 1, "SI2": 2}
	m3 := map[string]string{"SS1": "string1", "SS2": "string2"}
	m4 := map[int]string{12: "IS12", 24: "IS24"}
	resargs := extractargs([]interface{}{
		m1, m2, m3, m4, "str", 9919, 92.114e+1, true, nil,
	})
	assertBool(t, len(resargs) == len(m1)+len(m2)+len(m3),
		"Incompatible lenghts of result map and predefined")
}

//
func BenchmarkExtractArgs(b *testing.B) {
	m1 := Args{"Argskey1": 1234, "Argskey2": "1235", "Argskey3": nil}
	m2 := map[string]int{"SI1": 1, "SI2": 2}
	m3 := map[string]string{"SS1": "string1", "SS2": "string2"}
	m4 := map[int]string{12: "IS12", 24: "IS24"}
	for i := 0; i < b.N; i++ {
		extractargs([]interface{}{
			m1, m2, m3, m4, "str", 9919, 92.114e+1, true, nil,
		})
	}
}

//
func TestFormat(t *testing.T) {
	tf := func(have, want string) {
		assertBool(t, want == have,
			fmt.Sprintf("\nwant: %s\nhave: %s\n", want, have))
	}

	f1 := format("test string", nil)
	tf(f1, "test string")

	f2 := format("test {{", nil)
	tf(f2, "test {{")

	f3 := format("test {{as}}", nil)
	tf(f3, "test {{as}}")

	f4 := format("test {{key}}", Args{
		"key": "string",
	})
	tf(f4, "test string")

	f5 := format("{{s1}} {{i_}} {{unexisted}} {{", Args{
		"s1": "string",
		"i_": 124,
	})
	tf(f5, "string 124 {{unexisted}} {{")
}

//
func BenchmarkFormat(b *testing.B) {
	args := Args{
		"s1": "string",
		"i_": 124,
	}
	for i := 0; i < b.N; i++ {
		format("{{s1}} {{i_}} {{unexisted}} {{", args)
	}
}