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
import "strings"
import "sync"


// ========================================================================= //
// ============================ INITIALIZATION ============================= //
// ========================================================================= //

// The config of package.
var config = &Config{}
// Symbols that are used as a separator for parts of the key
// when using the Tr function.
var delimeters = `./\:`
// Default locale
// Returned by the LC function if the requested locale does not exist
// By default is nil and in this way (locale object is nil), placeholder string
// will be returned; but you can set any locale as default
// (see method SetAsDefault), and if the locale en_US was loaded,
// it will be used as the default locale.
var defloc *loc = nil
// Map from locale name to object of this locale.
var locales map[string]*loc
// Map from locale name to slice of files from which this locale was loaded.
var locdirs map[string][]string
// Sema for async operations
// ATTENTION!
// This package is not fully thread-safe.
// It is guaranteed that when locals are loaded (upgraded),
// it is impossible to obtain locale objects, but access to the locale objects
// is done by the pointer.
// It means that through the saved pointer during the load (update) period
// of the locale, it is possible to access the locale object,
// which can be updated at the moment.
// In the future, the concept will be revised,
// or the package will become fully thread-safe.
var sema sync.RWMutex

// Initialize function
// Allocates memory for maps.
func init() {
	locales = make(map[string]*loc)
	locdirs = make(map[string][]string)
}


// ========================================================================= //
// ========================== PRIVATE FUNCTIONS ============================ //
// ========================================================================= //

// "extractargs" extracts the objects of Args type or map[string]<Type> type
// from 'args' slice and unite them into one object of the Args type.
// Returns this object or nil if argument is empty slice.
func extractargs(args []interface{}) Args {
	// Avoid processing if args is nil/empty
	if len(args) == 0 {
		return nil
	}
	// Search 'Args' type in args, unite them if there are several
	realargs := Args{}
	for _, arg := range args {
		if arg == nil {
			continue
		}
		// If type of 'arg' is 'Args', copy all args to 'realargs'
		if margs, ok := arg.(Args); ok {
			for k, v := range margs {
				realargs[k] = v
			}
			continue
		}
		// If type is map[string]??
		rv := reflect.ValueOf(arg)
		tv := rv.Type()
		if tv.Kind() != reflect.Map {
			continue
		}
		// Get type of key of map, check whether is string
		if tv.Key().Kind() != reflect.String {
			continue
		}
		// Copy args using reflect
		for _, k := range rv.MapKeys() {
			realargs[k.String()] = rv.MapIndex(k).Interface()
		}
	} // end loop over args
	return realargs
}

// "format" formats the string 's' using 'args' for interpolation.
// Returns string 's' in which all keys (format '{{<key_name>}}') that exists
// in 'args' will be replaced by value at this key from 'args'.
// Unused values (in 'args') will be ignored.
// If string 's' have some key that 'args' doesn't, this key will be left
// in its original form.
func format(s string, args Args) string {
	// Avoid processing if args is nil/empty or s is ""
	if s == "" || args == nil || len(args) == 0 { return s }
	result := ""
	// Loop while original string isn't empty
	for s != "" {
		// Get the key start index
		// If this index is -1, append rest part of original string to result string
		// and force next iteration (at that the loop will be completed)
		idx_begin := strings.Index(s, "{{")
		if idx_begin == -1 {
			result += s
			s = ""
			continue
		}
		// Append part of string before the beginning of key to result string
		// Delete "{{" and appended part from rest of original string
		result += s[:idx_begin]
		s = s[idx_begin+2:]
		// Get the key end index (since the key start index)
		// If this index is -1, append back "{{" (but to the result string)
		// and append rest part to the result string
		// In generally, it means that key has form "{{...", that is incorrect
		// In this way, we think that it's part of string
		// Also, force the next iteration (at that the loop will be completed)
		idx_end := strings.Index(s, "}}")
		if idx_end == -1 {
			result += "{{" + s
			s = ""
			continue
		}
		// Extract the key, skip "}}" in the original string
		key := s[:idx_end]
		s = s[idx_end+2:]
		// Find key in args
		// If argument was found, format it and append to result string
		// Otherwise, save key in its original form
		if val, ok := args[key]; ok {
			result += formatarg(val)
		} else {
			result += "{{"+key+"}}"
		}
	} // end loop over original string
	return result
}

// "formatargs" represents the argument 'i' of interface{} type as string.
// Returns this string.
// At this moment, works through fmt.Sprintf function.
// May be this behaviour will be changed in the future.
func formatarg(i interface{}) string {
	return fmt.Sprintf("%+v", i)
}