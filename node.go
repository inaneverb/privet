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
import "strconv"

// Type "locnode"
// This type represents the node of locale and have two entities:
// subnodes (named nodes) and content (key-value storage - dictionary
// in which 'key' is the last part of name by which
// the 'value' (translated phrase) is available).
type locnode struct {
	subnodes map[string]*locnode
	content map[string]string
}


// ========================================================================= //
// ========================== PRIVATE FUNCTIONS ============================ //
// ========================================================================= //

// "makenode" creates an empty node of locale.
// Returns this node.
func makenode() *locnode {
	return &locnode{
		subnodes: make(map[string]*locnode),
		content: make(map[string]string),
	}
}


// ========================================================================= //
// =========================== PRIVATE METHODS ============================= //
// ========================================================================= //

// "tr" tries to find the translate for specified key.
// If the key has more than one part, it splits key into first part
// (before first dot) and the rest (after first dot).
// In this way, first "tr" tries to find named subnode using first part
// as name of subnode.
// If this subnode is exists, "tr" will be recursively called for this subnode
// using the rest part of the original key as key by that search will made.
// Otherwise, if 'key' doesn't contain the parts, "tr" tries to find translated
// phrase for this key in 'content' section.
// Returns translated phrase and 'true' if translate phrase was found.
// Otherwise, returns empty string and 'false'.
func (n *locnode) tr(key string) (string, bool) {
	// Try to split key by any of delimeters
	if idx := strings.IndexAny(key, delimeters); idx != -1 {
		prefix, key := key[:idx], key[idx+1:]
		if len(key) == 0 || len(prefix) == 0 {
			return "", false
		}
		// If the splitting is successful and the result parts aren't empty,
		// try to get the locale subnode using prefix (1st part of splitting) as key
		if subnode := n.subnode(prefix); subnode != nil {
			return subnode.tr(key)
		} else {
			return "", false
		}
	}
	// If no delimeter is found in key, use this key as key for content of node
	val, ok := n.content[key]
	return val, ok
}

// "appendNode" saves the 'node' by the specified 'key' into 'subnode' section.
// If node by this key is already exists, "appendNode" tries to move
// all own nodes and content into existed node.
// Content is moved using the "appendValue" method (see its description),
// but subnodes is moved using this function (recursively).
// Returns nil, if this operation was successful,
// otherwise generated error will be returned.
func (n *locnode) appendNode(key string, node *locnode, overwrite bool) error {
	// General error message
	var glerr string
	// If node with the specified 'key' is alredy exists, try to
	// move all values from appended 'node' into existed
	if node_existed, exist := n.subnodes[key]; exist && node_existed != nil {
		// Check if you try to append existed node
		if node == node_existed {
			return nil
		}
		// Move 'content'
		for k, v := range node.content {
			if err := node_existed.appendValue(k, v, overwrite); err != nil {
				glerr += key+": "+err.Error()+", "
			}
		}
		// Move 'subnodes' (recursively)
		for k, v := range node.subnodes {
			if err := node_existed.appendNode(k, v, overwrite); err != nil {
				glerr += err.Error()
			}
		}
		// Check errors
		if glerr != "" { return fmt.Errorf("%s", glerr[:len(glerr)-2]) }
		return nil
	}
	// If node with the specified 'key' doesn't exists, just add it
	n.subnodes[key] = node
	return nil
}

// "appendValue" saves the 'value' by the specified 'key' into 'content' section.
// If value by this key is already exists and overwritting is forbidden,
// does nothing and returns error.
// Otherwise, value will be saved and nil will be returned.
func (n *locnode) appendValue(key string, value string, overwrite bool) error {
	// If the value by this key is exists and overwrite is forbidden, return err
	if _, exists := n.content[key]; exists && !overwrite {
		return fmt.Errorf("value is exists")
	}
	// Otherwise save value by specified key into 'content' section
	n.content[key] = value
	return nil
}

// "subnode" extracts the subnode with the specified 'name' from current node.
// If a subnode with this name exists in 'subnode' section, it will be returned.
// Otherwise, nil will be returned.
func (n *locnode) subnode(name string) *locnode {
	node, exists := n.subnodes[name]
	if exists && node != nil { return node }
	return nil
}

// "cresubnodes" creates the nested subnodes with the specified 'names' and
// appends this subnode tree to 'subnode' section of current node.
// For example, if 'keys' is ['key1', 'key2'], after calling this function,
// current node will be have the subnode by key 'key1' and this subnode will be
// have the sub-subnode by key 'key2'.
// The second subnode (sub-subnode) will be returned in this way.
// If slice of keys is empty, current node will be returned.
func (n *locnode) cresubnodes(keys []string) *locnode {
	if len(keys) == 0 { return n }
	subnode := n.subnode(keys[0])
	if subnode == nil {
		subnode = makenode()
		n.appendNode(keys[0], subnode, false)
	}
	return subnode.cresubnodes(keys[1:])
}

// "json_loadSlice" is part of the JSON loading interface.
// This function tries to load slice that presented as reflect.Value type into
// current node.
// Generally, this method scans the slice and checks the type of each element.
// If the type is not map, this value will be ignored (error message about it
// will be added to general error message), but map will be appended to
// the current node using "json_loadMap" method.
// Returns nil, if slice was successfully added,
// otherwise the generated error is returned.
func (n *locnode) json_loadSlice(v reflect.Value, overwrite bool) error {
	// General error message (contain all error messages)
	var errcount = 0
	var glerr string
	// Loop over reflect.Value slice
	for i := 0; i < v.Len(); i++ {
		// Get value by index as reflect.Value
		v := v.Index(i)
		// Processing only if v is map, ignore the other all types
		switch v.Type().Kind() {
		// If map, try to unpacking map into current node
		case reflect.Map:
			if err := n.json_loadMap(v, overwrite); err != nil {
				glerr += fmt.Sprintf("[]: %s, ", err.Error())
				errcount++
			}
		default:
			glerr += fmt.Sprintf("[]: Invalid type of slice (%s)", v.Type().String())
			errcount++
		}
	} // end loop over reflect.Value slice
	// If glerr is not empty, generate error message
	if glerr != "" {
		if glerr[len(glerr)-1] == ' ' { glerr = glerr[:len(glerr)-1] }
		if glerr[len(glerr)-1] == ',' { glerr = glerr[:len(glerr)-1] }
		return fmt.Errorf("%d errors occured: {%s}", errcount, glerr)
	}
	return nil
}

// "json_loadMap" is part of the JSON loading interface.
// This function tries to load map that presented as reflect.Value type
// into current node.
// Generally, there are two options:
// 1. Value (from map) isn't slice or map.
// In this way, this value will be represented as string
// (if the value is scalar Golang type)
// and will be saved by the map key into 'content' section of current node.
// 2. Value (from map) is slice or map.
// Slice will be processed by the special function - "json_loadSlice".
// If value is map, the subnode will be created
// (by the key of value of original map) and the value-map wiil be extracted
// into this created subnode.
// Later this subnode will be added into section 'subnodes' of current node.
func (n *locnode) json_loadMap(v reflect.Value, overwrite bool) error {
	// General error message (contain all error messages)
	var errcount = 0
	var glerr string
	// Loop over reflect.Value map keys
	for _, key := range v.MapKeys() {
		// Represent reflect.Value key as string
		skey := key.String()
		// Get value by key as reflect.Value
		v := v.MapIndex(key)
		v = reflect.ValueOf(v.Interface())
		// Determine the type of value
		switch v.Type().Kind() {
		// If slice, create subnode and try to unpacking slice into subnode
		// If after all destination node isn't empty, add it to current node
		case reflect.Slice:
			subnode := n.subnode(skey)
			if subnode == nil {
				subnode = makenode()
			}
			err := subnode.json_loadSlice(v, overwrite)
			if err != nil {
				glerr += fmt.Sprintf("[%s]: %s, ", skey, err.Error())
				errcount++
				continue
			}
			if len(subnode.subnodes) != 0 || len(subnode.content) != 0 {
				if err = n.appendNode(skey, subnode, overwrite); err != nil {
					glerr += fmt.Sprintf("[%s]: %s, ", skey, err.Error())
					errcount++
				}
			}
		// If map, create subnode and try to unpacking map into subnode
		// If after all destination node isn't empty, add it to current node
		case reflect.Map:
			subnode := n.subnode(skey)
			if subnode == nil {
				subnode = makenode()
			}
			err := subnode.json_loadMap(v, overwrite)
			if err != nil {
				glerr += fmt.Sprintf("[%s]: %s, ", skey, err.Error())
				errcount++
				continue
			}
			if len(subnode.subnodes) != 0 || len(subnode.content) != 0 {
				if err = n.appendNode(skey, subnode, overwrite); err != nil {
					glerr += fmt.Sprintf("[%s]: %s, ", skey, err.Error())
					errcount++
				}
			}
		// If int, convert int to string and add this value by specified key
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			err := n.appendValue(skey, strconv.FormatInt(v.Int(), 10), overwrite)
			if err != nil {
				glerr += fmt.Sprintf("[%s]: %s, ", skey, err.Error())
				errcount++
			}
		// If uint, convert int to string and add this value by specified key
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			err := n.appendValue(skey, strconv.FormatUint(v.Uint(), 10), overwrite)
			if err != nil {
				glerr += fmt.Sprintf("[%s]: %s, ", skey, err.Error())
				errcount++
			}
		// If float, convert int to string and add this value by specified key
		case reflect.Float32, reflect.Float64:
			err := n.appendValue(skey, strconv.FormatFloat(v.Float(), 'g', -1, 64), overwrite)
			if err != nil {
				glerr += fmt.Sprintf("[%s]: %s, ", skey, err.Error())
				errcount++
			}
		// If string, convert int to string and add this value by specified key
		case reflect.String:
			err := n.appendValue(skey, v.String(), overwrite)
			if err != nil {
				glerr += fmt.Sprintf("[%s]: %s, ", skey, err.Error())
				errcount++
			}
		// In other way, unknown type, error
		default:
			glerr += fmt.Sprintf("[%s]: Unknown type (%s)", skey, v.Type().String())
		}
	} // end loop over reflect.Value map keys
	// If glerr is not empty, generate error message
	if glerr != "" {
		if glerr[len(glerr)-1] == ' ' { glerr = glerr[:len(glerr)-1] }
		if glerr[len(glerr)-1] == ',' { glerr = glerr[:len(glerr)-1] }
		return fmt.Errorf("%d errors occured: {%s}", errcount, glerr)
	}
	return nil
}