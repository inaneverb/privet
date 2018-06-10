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
import "io/ioutil"
import "path/filepath"

// Type "Config"
// This type represents the config that (will be) used for loading
// locale directories and files.
// todo: Implement dependency the Load() method by the config
type Config struct {
	OverwriteExistingKey bool
}


// ========================================================================= //
// =========================== PUBLIC FUNCTIONS ============================ //
// ========================================================================= //

// "Load" takes the path to the root of locales.
// This root (root directory) will be used for initializing all locales.
// "Load" extracts all subdirectories from 'rootdir' and,
// if their names are correct locale names
// (See "isValidLocaleName" function for details),
// determine each directory as source of locale data.
// Generate object of locale (by locale dir name) and load the data of
// directory into locale with which directory is associated.
// todo: Type of arg -> interface{} (string, os.File, os.FileInfo)
// todo: Args -> ...interface{} (few args, few directories, few roots)
func Load(rootdir string) error {
	// Check is rootdir is empty
	if rootdir == "" {
		return fmt.Errorf("arg 'rootdir' is empty")
	}
	// Convert rootdir to abs path (if relative)
	if absrootdir, err := filepath.Abs(rootdir); err == nil {
		rootdir = absrootdir
	}
	// Get all entities from rootdir, check error, check length
	entities, err := ioutil.ReadDir(rootdir)
	if err != nil {
		return err
	}
	if len(entities) == 0 {
		return fmt.Errorf("dir is empty")
	}
	// Count of loaded directories
	count := 0
	// Loop over entities in rootdir
	for _, entity := range entities {
		// Skip files, links, etc
		if !entity.IsDir() {
			continue
		}
		// Save dirname, check whether name is corrent and valid
		dir := entity.Name()
		if !isValidLocaleName(dir) {
			continue
		}
		// Try to get locale by name
		// If locale doesn't exists, create it
		loc := getlocale(dir)
		if loc == nil {
			loc = makelocale(dir)
		}
		// Generate fullpath to locale directory, try to load directory
		// into locale
		fullpath := filepath.Join(rootdir, dir)
		if err := loc.load(fullpath); err != nil {
			return err
		}
		// If success, increase count of successful loaded directories
		// binds path to locale directory to locale name
		count++
		locdirs[dir] = append(locdirs[dir], fullpath)
	} // end loop over entities in rootdir
	// Check whether count of loaded directories more than zero
	if count == 0 {
		return fmt.Errorf("there is no valid locale directory in the specified " +
			"root directory (%s)", rootdir)
	}
	// Set default locale if 'en_US' locale was loaded
	if enloc := LC("en_US"); enloc != nil {
		enloc.SetAsDefault()
	}
	return nil
}


// ========================================================================= //
// ========================== PRIVATE FUNCTIONS ============================ //
// ========================================================================= //

// "isValidLocaleName" returns true if the 'name' is valid locale name,
// otherwise returns false.
// Locale name is valid only if it consists of 5 characters, 1st and 2nd
// characters are lowercase letters, 4th and 5th are uppercase letters and
// the 3rd symbol is '_' or '-'.
// In this way, first part of name is language name and the second part is
// country name.
// For example, valid locales is: 'en_GB', 'en_US', 'es_ES', etc.
func isValidLocaleName(name string) bool {
	return len(name) == 5 &&
		isLowerCaseLetter(name[0]) && isLowerCaseLetter(name[1]) &&
		isUpperCaseLetter(name[3]) && isUpperCaseLetter(name[4]) &&
		(name[2] == '_' || name[2] == '-')
}

// "isUpperCaseLetter" returns true if the 'letter' is an uppercase letter,
// otherwise returns false.
func isUpperCaseLetter(letter byte) bool {
	return letter >= 'A' && letter <= 'Z'
}

// "isLowerCaseLetter" returns true if the 'letter' is a lowercase letter,
// otherwise returns false.
func isLowerCaseLetter(letter byte) bool {
	return letter >= 'a' && letter <= 'z'
}