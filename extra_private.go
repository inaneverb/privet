// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"github.com/qioalice/ekago/v2/ekastr"
)

/*
isValidLocaleName reports whether passed s is a valid locale name
that is in the following format "xx_YY", where:
 - xx is a lower case chars of language name ("en", "ru", "jp"),
 - YY is a upper case chars of country name ("US", "GB", "RU").
*/
func isValidLocaleName(s string) bool {
	return len(s) == 5 &&
		ekastr.CharIsLowerCaseLetter(s[0]) &&
		ekastr.CharIsLowerCaseLetter(s[1]) &&
		ekastr.CharIsUpperCaseLetter(s[3]) &&
		ekastr.CharIsUpperCaseLetter(s[4]) &&
		s[2] == '_'
}
