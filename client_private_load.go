// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"sync/atomic"

	"github.com/qioalice/ekago/v2/ekaerr"

	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v3"
)

var (
	/*
	TODO: comment
	*/
	loadContentUnknownResolvers = []struct{
		Unmarshaler    func(d []byte, v interface{}) error
		AssociatedType SourceItemType
	}{
		{
			Unmarshaler: yaml.Unmarshal,
			AssociatedType: SOURCE_ITEM_TYPE_CONTENT_YAML,
		},
		{
			Unmarshaler: toml.Unmarshal,
			AssociatedType: SOURCE_ITEM_TYPE_CONTENT_TOML,
		},
	}
)

/*
load literally does things Client.Load() method describes.
*/
func (c *Client) load() *ekaerr.Error {
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
		err = c.loadItem(i, overwrite)
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
loadItem tries to parse and then add all data from the SourceItem's locale content
placed in sourcesTmp by passed sourceItemIdx index.

To put it simply,
c.sourcesTmp[sourceItemIdx] will be loaded.
*/
func (c *Client) loadItem(sourceItemIdx int, overwrite bool) *ekaerr.Error {
	const s = "Failed to load sourced locale. "

	var (
		err        *ekaerr.Error
		rootMap    = make(map[string]interface{})
		sourceItem = c.sourcesTmp[sourceItemIdx]
	)

	switch sourceItem.Type {

	case SOURCE_ITEM_TYPE_FILE_YAML:
		legacyErr := yaml.Unmarshal(sourceItem.content, &rootMap)
		err = ekaerr.IllegalFormat.
			Wrap(legacyErr, s + "Failed to decode content using YAML decoder")

	case SOURCE_ITEM_TYPE_FILE_TOML:
		legacyErr := toml.Unmarshal(sourceItem.content, &rootMap)
		err = ekaerr.IllegalFormat.
			Wrap(legacyErr, s + "Failed to decode content using TOML decoder")

	case SOURCE_ITEM_TYPE_CONTENT_UNKNOWN:
		var legacyErr error
		for _, contentResolver := range loadContentUnknownResolvers {
			legacyErr = contentResolver.Unmarshaler(sourceItem.content, &rootMap)
			if legacyErr == nil {
				sourceItem.Type = contentResolver.AssociatedType
				break
			}
		}
		if legacyErr != nil {
			err = ekaerr.IllegalFormat.
				New(s + "All options for decoding the byte content have failed.")
		}

	default:
		// You should never see this error, because otherwise it's a bug.
		err = ekaerr.InternalError.
			New(s + "Unexpected type of SourceItem. This is a bug.")
	}

	//goland:noinspection GoNilness
	if err.IsNil() && len(rootMap) == 0 {
		err = ekaerr.IllegalFormat.
			New(s + "File has a valid format but an empty content.")
	}

	//goland:noinspection GoNilness
	if err.IsNil() {
		err = sourceItem.loadMetaData(rootMap)
	}

	//goland:noinspection GoNilness
	if err.IsNotNil() {
		return err.
			AddFields("privet_source", sourceItem.Path).
			Throw()
	}

	if err := c.scan(rootMap, sourceItemIdx, overwrite); err.IsNotNil() {
		return err.
			AddMessage(s).
			AddFields("privet_source", sourceItem.Path).
			Throw()
	}

	return nil
}

/*
TODO: comment
*/
func (c *Client) scan(

	root          map[string]interface{},
	sourceItemIdx int,
	overwrite     bool,

) *ekaerr.Error {

	sourceItem := c.sourcesTmp[sourceItemIdx]

	loc := c.storageTmp[sourceItem.LocaleName]
	if loc == nil {
		loc = c.makeLocale(sourceItem.LocaleName)
		c.storageTmp[sourceItem.LocaleName] = loc
	}

	if err := loc.root.scan(root, sourceItemIdx, overwrite); err.IsNotNil() {
		return err.
			Throw()
	}

	loc.root.applyRecursively(func(node *localeNode) {
		for key, value := range node.contentTmp {
			node.content[key] = value
			loc.phrasesCount++
			delete(node.contentTmp, key)
		}
	})

	return nil
}
