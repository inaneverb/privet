// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"bytes"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/qioalice/ekago/v2/ekaerr"
	"github.com/qioalice/ekago/v2/ekaunsafe"

	"github.com/modern-go/reflect2"
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
	const s = "Failed to count one or many locale sources. "
	switch {

	case !c.isValid():
		return ekaerr.IllegalState.
			New(s + "Client is not valid.").
			Throw()

	case len(args) == 0:
		return ekaerr.IllegalArgument.
			New(s + "There are no sources.").
			Throw()
	}

	if !(c.changeState(_LLS_STANDBY, _LLS_SOURCE_PENDING) ||
		c.changeState(_LLS_READY, _LLS_SOURCE_PENDING)) {

		allowedStates := []string{
			strState(_LLS_STANDBY),
			strState(_LLS_READY),
		}

		return ekaerr.IllegalState.
			New(s + "Another Source() or Load() called.").
			AddFields("privet_allowed_states", strings.Join(allowedStates, ", ")).
			Throw()
	}

	// We got "lock" of c.state as _LLS_SOURCE_PENDING.
	// We need to change it to _LLS_STANDBY or _LLS_READY when this func is over
	// depends on HOW this func is over.
	//
	// _LLS_STANDBY, when:
	//  - All new sources has been counted, nil is returned to the caller;
	//  - New sources has not been counted, not nil error is returned to the caller,
	//    AND there was already counted NEW sources (previous call of Source()).
	//
	// _LLS_READY, when:
	//  - New sources has not been counted, not nil error is returned to the caller,
	//    AND there was NO already counted NEW sources (was no previous calls of Source()),
	//    AND there was previous successful call of Load().
	defer func(c *Client){
		if len(c.sourcesTmp) == 0 && c.storage != nil {
			c.changeStateForce(_LLS_READY)
		} else {
			c.changeStateForce(_LLS_STANDBY)
		}
	}(c)

	var (
		sources = make([]SourceItem, 0, len(args))
		err     *ekaerr.Error
	)

	//goland:noinspection GoNilness
	for _, arg := range args {

		switch argType := reflect2.TypeOf(arg); argType.RType() {

		case ekaunsafe.RTypeString():
			err = c.sourceString(&sources, arg.(string), 0)

		case ekaunsafe.RTypeStringArray():
			arr := arg.([]string)
			for i, n := 0, len(arr); i < n && err.IsNil(); i ++ {
				err = c.sourceString(&sources, arr[i], 0)
			}

		case ekaunsafe.RTypeBytes():
			err = c.sourceBytes(&sources, arg.([]byte))

		case ekaunsafe.RTypeBytesArray():
			arr := arg.([][]byte)
			for i, n := 0, len(arr); i < n && err.IsNil(); i++ {
				err = c.sourceBytes(&sources, arr[i])
			}

		default:
			return ekaerr.IllegalArgument.
				New(s + "Unexpected type of source.").
				AddFields("privet_source_type", argType.String()).
				Throw()
		}

		if err.IsNotNil() {
			return err.
				AddMessage(s).
				Throw()
		}
	}

	// There are two MD5 checks.
	// First that there is no the same sources just counted.
	// Second that there is no the same sources in just counted sources
	// and already counted.

	var (
		i, j int
		// se2[j] (if not nil) contains the SourceItem with the same content,
		// as sources[i] contains. Used to catch an error.
		se2 []SourceItem
	)

	//goland:noinspection GoNilness
	for n := len(sources); i < n && se2 == nil; i++ {

		//goland:noinspection GoNilness
		for j = i+1; j < n && se2 == nil; j++ {

			if sources[i].md5 == sources[j].md5 {
				se2 = sources
			}
		}

		for j, m := 0, len(c.sourcesTmp); j < m && se2 == nil; j++ {
			if sources[i].md5 == c.sourcesTmp[j].md5 {
				se2 = c.sourcesTmp
			}
		}
	}

	if se2 != nil {
		return ekaerr.IllegalArgument.
			New(s + "Two sources with the same content detected.").
			AddFields(
				"privet_source_1", sources[i].Path,
				"privet_source_2", se2[j].Path).
			Throw()
	}

	if len(sources) == 0 {
		return ekaerr.IllegalArgument.
			New(s + "There are no valid sources.").
			Throw()
	}

	if len(c.sourcesTmp) != 0 {
		c.sourcesTmp = append(c.sourcesTmp, sources...)
	} else {
		c.sourcesTmp = sources
	}

	return nil
}

/*
TODO: comment
*/
func (c *Client) Load() *ekaerr.Error {
	const s = "Failed to load sourced locales. "
	switch {

	case !c.isValid():
		return ekaerr.IllegalState.
			New(s + "Client is not valid.").
			Throw()

	case c.getState() == _LLS_READY:
		// There was no successful Source() call before Load() one?
		return ekaerr.IllegalState.
			New(s + "There was no successful Source() call before.").
			Throw()

	case !c.changeState(_LLS_STANDBY, _LLS_LOAD_PENDING):
		// If we can't do CAS, it because of data-race.
		// Another one Load() is called. Or Source(). No matter.
		return ekaerr.IllegalState.
			New(s + "Another Source() or Load() called.").
			AddFields("privet_allowed_states", strState(_LLS_STANDBY)).
			Throw()
	}

	// We got "lock" of c.state as _LLS_LOAD_PENDING.
	// We need to change it to _LLS_STANDBY or _LLS_READY when this func is over
	// depends on HOW this func is over.
	//
	// _LLS_READY, when:
	//  - All new locales has been loaded, nil is returned to the caller;
	//  - New locales has not been loaded, not nil error is returned to the caller,
	//    but there was previous successfully loaded locales.
	//
	// _LLS_STANDBY, when:
	//  - New locales has not been loaded, not nil error is returned to the caller,
	//    AND there was no previous loaded locales.

	defer func(c *Client){
		if c.storage != nil {
			c.changeStateForce(_LLS_READY)
		} else {
			c.changeStateForce(_LLS_STANDBY)
		}
	}(c)

	switch {
	case len(c.sourcesTmp) == 0:
		return ekaerr.IllegalState.
			New(s + "There is no valid sources counted yet.").
			Throw()
	}

	// We don't have Client's fields initialization.
	// So, initialize storageTmp here if it's not yet so.

	if c.storageTmp == nil {
		c.storageTmp = make(map[string]*Locale)
	}

	// We are ready to start loading.
	// Let's go.

	overwrite := atomic.LoadUint32(&c.config.OverwriteExistingKey) == 1

	var err *ekaerr.Error
	for i, n := 0, len(c.sourcesTmp); i < n && err == nil; i++ {
		err = c.load(i, overwrite)
	}

	// There is no necessary to hold locale's content anymore.
	// No matter, whether sources has been loaded successfully or not.

	for i, n := 0, len(c.sourcesTmp); i < n; i++ {
		c.sourcesTmp[i].content = nil
	}

	cleanupAfterFailedLoad := func(c *Client) {
		c.sourcesTmp = c.sourcesTmp[:0]
		c.storageTmp = nil
		//for localeName, locale := range c.storageTmp {
		//	locale.destroy()
		//	delete(c.storageTmp, localeName)
		//}
	}

	//goland:noinspection GoNilness
	if err.IsNotNil() {

		cleanupAfterFailedLoad(c)
		return err.
			AddMessage(s).
			Throw()
	}

	// Maybe files has been successfully parsed
	// but there is no loaded phrases?

	var phrasesCountTotal uint64
	for _, loadedLocale := range c.storageTmp {
		phrasesCountTotal += loadedLocale.phrasesCount
	}
	if phrasesCountTotal == 0 {
		cleanupAfterFailedLoad(c)
		return ekaerr.NotFound.
			New(s + "Sources has been parsed but there is no translation phrases.").
			Throw()
	}

	// OK. We are almost done.

	for _, loadedLocale := range c.storageTmp {
		loadedLocale.root.applyRecursively(func(node *localeNode) {
			node.contentTmp = nil
		})
	}

	c.storage = c.storageTmp
	c.storageTmp = nil

	c.sources = c.sourcesTmp
	c.sourcesTmp = c.sourcesTmp[:0]

	c.phrasesTotal = phrasesCountTotal
	c.localesTotal = uint32(len(c.storageTmp))

	c.setDefaultLocale(nil)

	return nil
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
