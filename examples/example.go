// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/qioalice/privet/v2"
)

func main() {

	// Scan starting from directory,
	// the current example.go file is placed in.

	_, source, _, _ := runtime.Caller(0)
	source = filepath.Dir(source)
	source = filepath.Join(source, ".") // in JS: source = source | "."

	privet.Source(source).LogAsFatal()
	privet.Load().LogAsFatal()

	privet.LC("en_US").MarkAsDefault()

	for _, localeName := range []string {
		"en_US",
		"ru_RU",
	} {
		fmt.Println(privet.Tr(localeName, "Main/Greetings", privet.Args{
			"name": "Alice",
		}))
	}
}
