// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"bytes"
	"sync/atomic"
	"unsafe"

	"github.com/qioalice/ekago/v2/ekaerr"
)

type (
	/*
	TODO: comment
	*/
	Client struct {

		/*
		state is a current state of the whole package.
		The package provides to you some promises (contracts),
		to maintain which this global variable exists.

		You may read more about promises (contracts) in the README.md file.
		Here is a some "under the hood" description of how is it works.
		 - You cannot call any Locale's getters during locale files loading and parsing.
		 - TODO:
		*/
		state uint32

		config struct {

			// C-like bool variables. 1 - true, 0 - false.
			// Protected by atomic operations.

			OverwriteExistingKey   uint32
			LCEmptyLocaleNameAsNil uint32
			LCNotFoundLocaleAsNil  uint32
		}

		defaultLocale unsafe.Pointer

		storage,
		storageTmp map[string]*Locale

		sources,
		sourcesTmp []SourceItem

		buf bytes.Buffer

		phrasesTotal uint64
		localesTotal uint32
	}
)

/*
Source allows you to add locale content or path to it.
Multiple calls are allowed or you can pass all arguments into one call.

        Source(<content1>)                                   // (1)
        Source(<content2>)                                   // (2)

is the same as

        Source(<content1>, <content2>)                       // (3)

You can even combine the arrays of content and single data.

        Source(<content1>, []{<content2>, <content3>})       // (4)

As you noticed, the examples above written at the near Golang pseudocode.
That's because there is different entities may be used as locale source
depends on type of argument.

First thing first, all arguments are passed and combined into one single array.
Passed arrays also too.

Then the formed array's items are analysed independently,
the whole set of sources is generating, step by step,
moving forward to being ready to loaded by Load() call.

Argument's types and their meanings:
(Keep in mind, that ALL OTHER ARGUMENT TYPES which not listed below ARE PROHIBITED
and will lead to fast return error).

As we know from the above, there is only some "base" types and arrays are allowed.
Base types are:

 - string (treated as path to either locale's directory or locale's one file),
 - []byte (treated as the content of locale's file).

Adding arrays to the list above and we've also get:

 - []string (treated as the array of either locale's directories, files or even mixed),
 - [][]byte (treated as the array of locale's files content).

The passed paths are analyzed as soon as possible during this call
(thus you will get an error if path is not exist, access denied or something else)
but the content will not be loaded until Load() call.
Also for passed []byte (or [][]byte).

Keep in mind, Load() call flushes all pending sources.
Thus you will need to re-register sources you want to use as source again
after Load() call,
if you need to call Load() more than one times (dunno why you even need that).

Path to directory or file might be absolute or relative.
If relative it will converted to absolutely starting from the current work directory.

You may provide path to directory that contains sub-directories.
In that case all these sub-directories will be scanned too recursively,
till locale files found.
*/
func (c *Client) Source(args ...interface{}) *ekaerr.Error {
	return c.source(args).Throw()
}

/*
TODO: comment
*/
func (c *Client) Load() *ekaerr.Error {
	return c.load().Throw()
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
func (c *Client) LC(name string) *Locale {

	if !c.isValid() {
		return nil
	}

	if name == "" {
		if atomic.LoadUint32(&c.config.LCEmptyLocaleNameAsNil) == 1 {
			return nil
		} else {
			return c.getDefaultLocale()
		}
	}

	if loc := c.getLocale(name); loc != nil {
		return loc

	} else if atomic.LoadUint32(&c.config.LCNotFoundLocaleAsNil) == 0 {
		return c.getDefaultLocale()

	} else {
		return nil
	}
}

/*
Default returns a Locale object that is marked as default Locale.
If no Locale marked as default, nil is returned.

Reminder:
It's safe to call Locale.Tr() of nil Locale.
You just will get an error string message. Not panic, not UB.
*/
func (c *Client) Default() *Locale {
	if !c.isValid() {
		return nil
	}
	return c.getDefaultLocale()
}

/*
Tr is an alias for Client.LC(localeName).Tr(key, args).
See LC() function and Locale.Tr() method for more details.
*/
func (c *Client) Tr(localeName, key string, args Args) string {
	return c.LC(localeName).Tr(key, args)
}
