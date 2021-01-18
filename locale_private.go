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

/*
makeSubNode is localeNode constructor and initializer.
The caller MUST save received localeNode to some other localeNode.subNodes map
or use it as Locale.root.
*/
func (l *Locale) makeSubNode() *localeNode {
	return &localeNode{
		parent:         l,
		subNodes:       make(map[string]*localeNode),
		content:        make(map[string]string),
		contentTmp:     make(map[string]string),
		usedSourcesIdx: nil,
	}
}
