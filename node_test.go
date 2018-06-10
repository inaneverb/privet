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
func TestLocnodeTr(t *testing.T) {
	node := makenode()
	assertBool(t, node != nil, "node is nil")

	subnode := makenode()
	assertBool(t, subnode != nil, "subnode is nil")

	assertErr(t, subnode.appendValue("key", "value", true))
	assertErr(t, node.appendNode("prefix", subnode, true))

	v, ok := node.tr("prefix.key")
	assertBool(t, ok && v == "value", fmt.Sprintf("tr(): %t, '%s'", ok, v))

	v, ok = node.tr("prefix.key2")
	assertBool(t, !ok && v == "", fmt.Sprintf("tr(): %t, '%s'", ok, v))
}

//
func BenchmarkLocnodeTr(b *testing.B) {
	node := makenode()
	subnode := makenode()
	subnode.appendValue("key", "value", true)
	node.appendNode("prefix", subnode, true)
	for i := 0; i < b.N; i++ {
		node.tr("prefix.key")
	}
}