// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"strings"
)

type (
	/*
	Locale is a storage of all translated phrases for one language.

	Getting locale by Client.LC() or Client.Default()
	allows you to get (from cache) Locale object,
	using which you may transform your translated key to the desired language's phrase.

	WARNING!
	You must not instantiate this class manually!
	It's useless but safely.
	So you won't get panicked or UB.
	Manually instantiated Locale objects are considered not initialized
	and provides to you the same behaviour as if it'd be nil.
	*/
	Locale struct {
		owner        *Client
		root         *localeNode
		name         string      // in format xx_YY
		phrasesCount uint64      // not only root localeNode but all nested also
	}
)

/*
Tr tries to get translated language phrase by the specified translation key
and then tries to interpolate this phrase using passed args, if any.

Nil safe.
If this method is called on nil object, the special string is returned.

Special returned strings.
All of special returned strings has the same format:

        "i18nErr: <error_class>. Key: <translation_key>".
                <translation_key> is your translation key,
                <error_class> might be:

 - _SPTR_LOCALE_IS_NIL:                Current Locale object is nil,
 - _SPTR_TRANSLATION_KEY_IS_EMPTY:     Translation key is empty,
 - _SPTR_TRANSLATION_KEY_IS_INCORRECT: Translation key is invalid (incorrect separator),
 - _SPTR_TRANSLATION_NOT_FOUND:        Translation not found.
*/
func (l *Locale) Tr(key string, args Args) string {

	switch {
	case !l.isValid():
		return sptr(_SPTR_LOCALE_IS_NIL, key)
	case key == "":
		return sptr(_SPTR_TRANSLATION_KEY_IS_EMPTY, key)
	}

	var (
		prefix      string
		originalKey = key
	)

	for node := l.root; node != nil; {
		if idx := strings.IndexByte(key, DEFAULT_DELIMITER); idx != -1 {
			prefix, key = key[:idx], key[idx+1:]

			if len(key) == 0 || len(prefix) == 0 {
				return sptr(_SPTR_TRANSLATION_KEY_IS_INCORRECT, originalKey)
			}

			node = node.subNode(prefix, false)
			continue

		} else if translatedPhrase, found := node.content[key]; found {
			return format(translatedPhrase, args)

		} else {
			return sptr(_SPTR_TRANSLATION_NOT_FOUND, originalKey)
		}
	}

	return sptr(_SPTR_TRANSLATION_NOT_FOUND, originalKey)
}

/*
MarkAsDefault marks the current Locale object as a default Locale.
If any Locale was marked as default Locale already, the will be overwritten.

Nil safe.
If this method is called on nil object, there is no-op.
*/
func (l *Locale) MarkAsDefault() {
	if !l.isValid() {
		return
	}
	l.owner.setDefaultLocale(l)
}

/*
Name returns the current Locale's name.

Returned name is always in "xx_YY" format, where:
 - xx is a lower case chars of language name ("en", "ru", "jp"),
 - YY is a upper case chars of country name ("US", "GB", "RU").

Nil safe.
If this method is called on nil object, the empty string is returned.
*/
func (l *Locale) Name() string {
	if !l.isValid() {
		return ""
	}
	return l.name
}
