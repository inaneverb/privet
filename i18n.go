// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"github.com/qioalice/ekago/v2/ekaerr"
)

//goland:noinspection GoSnakeCaseUsage
const (
	DEFAULT_DELIMITER byte = '/'
)

/*

*/
func Source(args ...interface{}) *ekaerr.Error {
	return defaultClient.source(args).Throw()
}

/*

*/
func Load() *ekaerr.Error {
	return defaultClient.load().Throw()
}

/*
LC returns the requested Locale by its name.

If the Locale with the specified name doesn't exists (or if name is empty):
 - Default Locale is returned if any locale marked as default;
 - nil is returned if no locale is marked as default.

You may change that behaviour using:

 1. Config.LCEmptyLocaleNameAsNil set to true (false by default)
    if you want to get nil Locale if name is empty
    (even if any Locale is marked as default).

 2. Config.LCNotFoundLocaleAsNil set to true (false by default)
    if you want to get nil Locale if Locale with requested name not found
    (even if any Locale is marked as default).
*/
func LC(name string) *Locale {
	return defaultClient.LC(name)
}

func Default() *Locale {
	return defaultClient.Default()
}

/*
Tr is an alias for LC(localeName).Tr(key, args).
See LC() function and Locale.Tr() method for more details.
*/
func Tr(localeName, key string, args Args) string {
	return defaultClient.LC(localeName).Tr(key, args)
}
