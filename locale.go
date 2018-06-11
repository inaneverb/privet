// Copyright © 2018. All rights reserved.
// Author: Alice Qio.
// Contacts: <qioalice@gmail.com>.
// License: https://opensource.org/licenses/MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom
// the Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
// IN THE SOFTWARE.

package i18n

import "fmt"
import "reflect"
import "io/ioutil"
import "path/filepath"
import "strings"
import "encoding/json"

// Type "Locale"
// This type represents the locale.
// Since this type is an alias to 'locnode', architecturally this type is
// root node of locale. See 'locnode' type for details.
type Locale locnode

// Type "Args"
// This type represents map of arguments that is used for interpolating
// translated phrase.
// See methods "Tr" of 'Locale' type and "tr" of 'locnode' type for details.
type Args map[string]interface{}


// ========================================================================= //
// =========================== PUBLIC FUNCTIONS ============================ //
// ========================================================================= //

// "LC" returns the object of locale with the specified 'name'.
// If the locale with the specified name doesn't exists,
// the default locale will be returned.
// But if the default locale isn't set, nil is returned.
// But using the method "Tr" for the nil object is safe.
// See method "Tr" of 'Locale' for details.
func LC(name string) *Locale {
	if name == "" { return nil }
	loc := getlocale(name)
	if loc == nil { return deflocale() }
	return loc
}

// "Tr" returns interpolated (using the specified 'args') translated string
// by the specified 'key' for locale with name 'locname'.
// See methods LC and Tr of 'Locale' for details.
func Tr(locname, key string, args ...interface{}) string {
	return LC(locname).Tr(key, args...)
}


// ========================================================================= //
// ============================ PUBLIC METHODS ============================= //
// ========================================================================= //

// "Tr" tries to get translate phrase by the specified 'key' and then tries to
// interpolate this phrase using 'args' (see type Args) if the 'args' is specified.
// If this method is called on nil object, the special string will be returned.
// No panics, no errors.
// In this way, returned string will have the following format: '<LC?>${{key}}',
// where 'key' is the specified 'key'.
// If key is empty, string '<empty key>' is returned.
// If the phrase by this key doesn't exists, string '${{key}}' is returned,
// where key is the specified 'key'.
func (l *Locale) Tr(key string, args ...interface{}) string {
	// Validate receiver and arguments
	if l == nil  { return "<?LC>${{"+key+"}}" }
	if key == "" { return "<empty key>" }
	// Find value by key
	text, ok := (*locnode)(l).tr(key)
	// If value is not exists, return placeholder like "${{key}}"
	if !ok { return "${{"+key+"}}" }
	// Format result if the extraction of args was successful
	return format(text, extractargs(args))
}

// "SetAsDefault" sets the current locale as default locale,
// if it is not nil and returns nil. Otherwise does nothing and returns error.
// After the some locale was selected as default locale,
// the "LC" function will return this locale if can't find requested locale.
// See "LC" function for details.
func (l *Locale) SetAsDefault() error {
	if l == nil {
		return fmt.Errorf("current object of locale is nil")
	}
	sema.Lock()
	defer sema.Unlock()
	defloc = l
	return nil
}


// ========================================================================= //
// ========================== PRIVATE FUNCTIONS ============================ //
// ========================================================================= //

// "getlocale" returns the locale with the specified 'name'.
// If this locale doesn't exists, nil is returned.
func getlocale(name string) *Locale {
	sema.RLock()
	defer sema.RUnlock()
	return locales[name]
}

// "deflocale" returns the default locale.
// If no locale marked as default, nil is returned.
func deflocale() *Locale {
	sema.RLock()
	defer sema.RUnlock()
	return defloc
}

// "makelocale" creates a new locale with the specified 'name'.
// Also, if locale with the requested name is already exists, it will be returned.
// Otherwise, new locale will be added into map of locales, and will be returned.
func makelocale(name string) *Locale {
	sema.Lock()
	defer sema.Unlock()
	if loc, ok := locales[name]; ok && loc != nil {
		return loc
	}
	loc := (*Locale)(makenode())
	locales[name] = loc
	return loc
}

// "rmlocifempty" deletes the locale with the specified 'name' only if it is empty.
// Returns true if the locale with the specified name has been deleted.
// Otherwise false is returned.
// ATTENTION!
// Checking whether the locale is empty is only performed by size of 'content'
// and 'subnodes' of root locale node. This check is not recursive!
// todo: Make this function recursive
func rmlocifempty(name string) bool {
	sema.Lock()
	defer sema.Unlock()
	loc := locales[name]
	if loc == nil {
		return false
	}
	if len(loc.subnodes) == 0 && len(loc.content) == 0 {
		delete(locales, name)
		return true
	}
	return false
}

// ========================================================================= //
// =========================== PRIVATE METHODS ============================= //
// ========================================================================= //

// "load" loads the files from 'dirpath' directory, parses them and represents
// as locale data. Later, it transforms this data into current locale data.
// "load" perform handling each file in the specified directory
// depending of the file type.
// Returns nil, if directory contain valid locale files and these files were
// successfully loaded and processed.
// Otherwise, error object is returned.
func (l *Locale) load(dirpath string) error {
	// Load all entities from directory, check error, check length
	entities, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return fmt.Errorf("can't get list of files (%s) in the directory (%s)",
			err, dirpath)
	}
	if len(entities) == 0 {
		return fmt.Errorf("no files in the directory (%s)", dirpath)
	}
	// Count of loaded files
	count := 0
	// Loop over entities in 'dirpath'
	for _, entity := range entities {
		// Skip directories
		if entity.IsDir() {
			continue
		}
		// Save name of file
		name := entity.Name()
		// todo: Make the division into formats - JSON, INI, YAML, TOML, etc
		// Type of loader - the function that will load and parse file
		var f func(string) error
		// Determine the loader
		switch strings.ToUpper(filepath.Ext(name)) {
		case ".JSON": f = l.loadJSON
		}
		// If loader was determined
		if f != nil {
			err := f(filepath.Join(dirpath, name))
			if err != nil {
				return err
			}
			count++
		}
	} // end loop over entities in 'dirpath'
	// Check whether count of loaded files more than zero
	if count == 0 {
		return fmt.Errorf("there is no valid locale files in locale directory (%s)",
			dirpath)
	}
	return nil
}

// "loadJson" loads the file by the specified 'fpath' filepath,
// parses it as JSON file and appends its data to current locale.
// All non last keys in json-objects will become subnodes,
// and last (in the tree) json-objects will became as locale content -
// a dictionary - map from keys to translated phrases.
// Returns nil, if file was successfully loaded and processed.
// Otherwise error object is returned.
func (l *Locale) loadJSON(fpath string) error {
	// Read file
	bytes, err := ioutil.ReadFile(fpath)
	if err != nil {
		return fmt.Errorf("can't open JSON file (%s) because (%s)", fpath, err)
	}
	// Unmarshal JSON data into interface{} (will be slice or map)
	dest := interface{}(nil)
	if err := json.Unmarshal(bytes, &dest); err != nil {
		return fmt.Errorf("can't unmarshal JSON data into Golang datatypes (%s) " +
			"from JSON file (%s)", err, fpath)
	}
	// Generate subnodes by filename
	// For example, if filename is 'test.file.json', will be generated subnode
	// 'test' that will have subnode 'file'
	// All data will be loaded into subnode 'file' ('test.file')
	parts := strings.Split(filepath.Base(fpath), ".")
	parts = parts[:len(parts)-1]
	// Generate subnode
	subnode := (*locnode)(l).cresubnodes(parts)
	// Determine root type of JSON tree
	v := reflect.ValueOf(dest)
	switch v.Type().Kind() {
	// If map, load map into generated subnode
	case reflect.Map:
		return subnode.json_loadMap(v, config.OverwriteExistingKey)
	// If slice, load slice into generated subnode
	case reflect.Slice:
		return subnode.json_loadSlice(v, config.OverwriteExistingKey)
	// Otherwise, return error
	default:
		return fmt.Errorf("can't unmarshal JSON data into Golang datatypes " +
			"(toplevel JSON value must be array or object) from JSON file (%s)",
			fpath)
	}
}