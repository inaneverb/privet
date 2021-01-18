// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
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
load tries to parse and then add all data from the SourceItem's locale content
placed in sourcesTmp by passed sourceItemIdx index.

To put it simply,
c.sourcesTmp[sourceItemIdx] will be loaded.
*/
func (c *Client) load(sourceItemIdx int, overwrite bool) *ekaerr.Error {
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
