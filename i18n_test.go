// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/qioalice/privet/v2"
)

func TestFull(t *testing.T) {
	var c privet.Client


	fmt.Println(os.Getwd())
	_ = c
	_ = require.Error
}
