// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/qioalice/ekago/v2/ekaerr"
	"github.com/qioalice/ekago/v2/ekaunsafe"

	"github.com/modern-go/reflect2"
)

var (
	rtypeArrMapStringInterface = reflect2.RTypeOf([]map[string]interface{}(nil))
)

/*
loadMetaData tries to parse root considering that this
is a root of sourced locale document that must contain some metadata about itself
like locale name, etc.
*/
func (si *SourceItem) loadMetaData(root map[string]interface{}) *ekaerr.Error {
	const s = "Failed to find or parse metadata of content. "

	var (
		metaDataOriginalKey string
		metaData            interface{}
		metaDataMap         map[string]interface{}
	)

	for key, value := range root {

		// May happen more than one times because of strings.ToLower().
		// So, if root has "__MeTaDaTa__" and "__Metadata__" nodes,
		// this iteration will work for both of them.
		// So, we have to handle case when there is more than 1 sections.

		switch proceed := strings.ToLower(key) == "__metadata__"; {

		case proceed && metaData == nil:
			metaDataOriginalKey = key
			metaData = value
			delete(root, key)

		case proceed && metaData != nil:
			return ekaerr.IllegalFormat.
				New(s + "Metadata found but is ambiguous. Found two or more sections.").
				AddFields(
					"privet_metadata_key_1", metaDataOriginalKey,
					"privet_metadata_key_2", key).
				Throw()
		}
	}

	switch {
	case metaData == nil && si.LocaleName == "":
		return ekaerr.IllegalFormat.
			New(s + "Metadata not found, or has an incorrect tag.").
			Throw()

	case metaData == nil:
		return nil
	}

	// Value must be an object or an array with one object.
	switch t := reflect2.TypeOf(metaData); t.RType() {

	case ekaunsafe.RTypeMapStringInterface():
		metaDataMap = metaData.(map[string]interface{})

	case rtypeArrMapStringInterface:
		arr := metaData.([]map[string]interface{})
		if len(arr) != 1 {
			return ekaerr.IllegalFormat.
				New(s + "Metadata found but is ambiguous. Found two or more objects.").
				AddFields("privet_metadata_key", metaDataOriginalKey).
				Throw()
		}
		metaDataMap = arr[0]

	default:
		return ekaerr.IllegalFormat.
			New(s + "Metadata tag found but has an incorrect type. Should be an object.").
			AddFields(
				"privet_metadata_key",  metaDataOriginalKey,
				"privet_metadata_type", t.String()).
			Throw()
	}

	if si.LocaleName == "" && len(metaDataMap) == 0 {
		return ekaerr.IllegalFormat.
			New(s + "Metadata found but does not have any field.").
			AddFields("privet_metadata_key", metaDataOriginalKey).
			Throw()
	}

	// Extract locale name
	for key, value := range metaDataMap {
		switch strings.ToLower(key) {

		case "locale_name", "localename", "locale", "name":
			if t := reflect2.TypeOf(value); t.RType() == ekaunsafe.RTypeString() {
				if si.LocaleName == "" {
					t.UnsafeSet(unsafe.Pointer(&si.LocaleName), ekaunsafe.TakeRealAddr(value))
				} else {
					return ekaerr.IllegalFormat.
						New(s + "Metadata found, but locale name is ambiguous. " +
							"Found two or more locale names. " +
							"Maybe filepath already contain locale name?").
						AddFields("privet_metadata_key", metaDataOriginalKey).
						Throw()
				}
			} else {
				return ekaerr.IllegalFormat.
					New(s + "Metadata found, but locale name has an incorrect type.").
					AddFields(
						"privet_metadata_key",              metaDataOriginalKey,
						"privet_metadata_locale_name_type", t.String()).
					Throw()
			}
		}
	}

	// Validate locale name
	switch {

	case si.LocaleName == "":
		return ekaerr.IllegalFormat.
			New(s + "Metadata found, but locale name is not provided or empty.").
			AddFields("privet_metadata_key", metaDataOriginalKey).
			Throw()

	case !isValidLocaleName(si.LocaleName):
		return ekaerr.IllegalFormat.
			New(s + "Metadata found but locale name has an incorrect format. Should be: xx_YY.").
			AddFields("privet_metadata_key", metaDataOriginalKey).
			Throw()
	}

	return nil
}

/*
findLocaleInFilepath tries to find a locale name in the current SourceItem's filepath.
Any part of filepath MAY contain (it's not necessary to be exactly equal)
a locale name. If it so, it will be parsed and associated with the current SourceItem.

Returns nil if filepath don't have a locale name,
but an error if contain more than one.
*/
func (si *SourceItem) findLocaleInFilepath() *ekaerr.Error {
	const s = "Failed to check whether source filepath contains locale name. "

	const SEPARATORS = "_-. "

	var (
		// pathParts contains not exactly directories and filename,
		// but a parts of that directories and filename,
		// that are split by any of allowed separator.
		pathParts []string

		// tmp contains a parts of one directory name or file name.
		// These parts are will be added to the pathParts later.
		tmp = make([]string, 0, 32)
	)

	for _, filePathPart := range strings.Split(
		si.Path[len(filepath.VolumeName(si.Path)):], // si.Path w/o volume
		string(filepath.Separator),                  // splits by os.PathSeparator
	) {
		if filePathPart == "" {
			continue
		}

		tmp = tmp[:0]

		for
		index := strings.IndexAny(filePathPart, SEPARATORS);
		index != -1;
		index = strings.IndexAny(filePathPart, SEPARATORS){
			tmp = append(tmp, filePathPart[:index])
			tmp = append(tmp, string(filePathPart[index]))
			filePathPart = filePathPart[index+1:]
		}

		// Do not forgot to add last item (or once one if there was no separator).
		tmp = append(tmp, filePathPart)

		// Analyse pathPartParts.
		// Concatenate sequences of "xx", "_", "YY" if there any.
		// Remove separators (from ALLOWED_SEPARATORS) otherwise.

		for i, n := 0, len(tmp); i < n; i++ {
			if i < n-2  && len(tmp[i]) + len(tmp[i+1]) + len(tmp[i+2]) == 5 {
				if s := tmp[i] + tmp[i+1] + tmp[i+2]; isValidLocaleName(s) {
					pathParts = append(pathParts, s)
					i += 2
				}
			} else if len(tmp[i]) > 1 || strings.IndexByte(SEPARATORS, tmp[i][0]) == -1 {
				pathParts = append(pathParts, tmp[i])
			}
		}
	}

	for _, pathPart := range pathParts {
		switch proceed := isValidLocaleName(pathPart); {

		case proceed && si.LocaleName == "":
			si.LocaleName = pathPart

		case proceed && si.LocaleName != "":
			return ekaerr.IllegalFormat.
				New(s + "Locale name is ambiguous. Found two or more locale names in filepath.").
				Throw()
		}
	}

	return nil
}
