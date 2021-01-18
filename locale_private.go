// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

/*
isValid ensures that the current Locale object is not nil and initialized correctly
(not manually instantiated by the caller). Returns true if this is correct object.
*/
func (l *Locale) isValid() bool {
	return l != nil && l.owner != nil && l.root != nil
}
