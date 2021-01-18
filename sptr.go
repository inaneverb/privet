// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

type (
	_SpecialTranslationClass string
)

//goland:noinspection GoSnakeCaseUsage
const (
	__SPTR_PREFIX = _SpecialTranslationClass("i18nErr: ")
	__SPTR_SUFFIX = _SpecialTranslationClass(". Key: ")

	_SPTR_TRANSLATION_NOT_FOUND = __SPTR_PREFIX +
		_SpecialTranslationClass("TranslationNotFound") + __SPTR_SUFFIX

	_SPTR_LOCALE_IS_NIL = __SPTR_PREFIX +
		_SpecialTranslationClass("LocaleIsNil") + __SPTR_SUFFIX

	_SPTR_TRANSLATION_KEY_IS_EMPTY = __SPTR_PREFIX +
		_SpecialTranslationClass("TranslationKeyIsEmpty") + __SPTR_SUFFIX

	_SPTR_TRANSLATION_KEY_IS_INCORRECT = __SPTR_PREFIX +
		_SpecialTranslationClass("TranslationKeyIsIncorrect") + __SPTR_SUFFIX
)

/*
Trivia:
Locale.Tr() or Client.Tr() may have an error.
Not existed or empty translation key, not initialized Client,
an errors of interpolation of language phrase with arguments, and others.

We need to way to say caller that there was an error.
I do not want to use *ekaerr.Error
as a 2nd return argument of Locale.Tr() or Client.Tr() methods.
Caller's checks will be too hard to read.

There is another way.
A special strings. It's OK. Users will say:
"Ha, bad translations. Found an easter egg. Or visual translation bug."
And it's ok. It will not lead to some bad consequences. I mean, very bad.

So, sptr() is just a generator of that "easter egg" - a special string
that you (as a caller) may get instead of language phrase. If something went wrong.

And "_SPTR_" starts constants are classes for that generator.
*/
func sptr(class _SpecialTranslationClass, originalKey string) string {
	return string(class) + originalKey
}
