// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"strconv"
	"strings"

	"github.com/qioalice/ekago/v2/ekaerr"
	"github.com/qioalice/ekago/v2/ekaunsafe"

	"github.com/modern-go/reflect2"
)

type (
	/*
	localeNode represents the node of locale that contains:

	 - Its data as KV storage, K is a translation key, V is a language phrase.
	   Field content stores these phrases.

	 - Child (derived) localeNode s.
	   E.g: if translate key is "a.b", and the current node is root node,
	   then subNodes will contain localeNode by a key "a"
	   with the translate key "b" that represents requested phrase.

	While parsing locale's source, before one source parsing is completed,
	its phrases will saved into contentTmp, meaning them as a temporary.
	It's done to provide a mechanism to drop all one source's phrases
	if there was an error of parsing that source.

	After one source parsing is completed, the all phrases from contentTmp
	must be MOVED to the content.
	Client.scan() using localeNode.applyRecursively() doing that.

	usedSourcesIdx contains indexes of Client.sources or Client.sourcesTmp
	(depends of Client's state - either sources under loading or not),
	meaning that sources with these indexes were used
	to construct EXACTLY current node (content), neither nested nor parented.
	*/
	localeNode struct {
		parent         *Locale
		subNodes       map[string]*localeNode
		content        map[string]string
		contentTmp     map[string]string
		usedSourcesIdx []int
	}
)

/*
subNode returns a localeNode with the given name
from the current localeNode's subNodes map.

If 2nd argument is true,
the new empty localeNode will be created and initialized,
if there is no localeNode with the given name in subNodes map.
If it's false, nil is returned.
*/
func (n *localeNode) subNode(name string, createIfNotExist bool) *localeNode {
	subNode := n.subNodes[name]

	if subNode == nil && createIfNotExist {
		subNode = n.parent.makeSubNode()
		n.subNodes[name] = subNode
	}

	return subNode
}

/*
applyRecursively calls passed callback cb passing the current localeNode,
treating it as a root, and then doing the same work for each localeNode from
subNodes "recursively".

Note.
"Recursively" above means, that each embedded localeNode (no matter how deep it is)
will be processed.
Say we have the localeNode tree like:

        Root                                       (A)
          |
          |---- Level 1.1 node                     (B)
          |         |
          |         |---- Level 2.1 node           (C)
          |         |         |
          |         |         |---- Level 3.1 node (D)
          |         |
          |         |---- Level 2.2 node           (E)
          |         |
          |         |---- Level 2.3 node           (F)
          |
          |---- Level 1.2 node                     (G)

For each localeNode of: A, B, C, D, E, F, G, a cb will be called,
and each of that localeNode will be passed (order is not guaranteed).

There is two ways HOW it will be done (depends on 2nd argument).
Iterative or recursive way of algorithm.

Recursive is preferred when the deep level and whole amount of localeNode
are not too high.
Otherwise iterative algorithm must be used,
because recursive tail optimisation is impossible here.

Requirements:
 - Current localeNode (n) != nil, panic otherwise.
 - Passed callback (cb) != nil, panic otherwise.
*/
func (n *localeNode) applyRecursively(cb func(*localeNode), recursive ...bool) {

	var applicator func(_ *localeNode, _ func(node *localeNode))

	doRecursive := len(recursive) > 0 && recursive[0]

	// TODO: There's a infinity loop dunno why, when two files have the duplicated
	//  translated phrases (phrases with the same translation key).
	//  Moreover, it leads to infinity loop only in an iterative algo.
	//  So, till that issue is resolved, recursive algo is forced.
	doRecursive = true

	if doRecursive {
		applicator = func(nodeToProcess *localeNode, cb func(*localeNode)) {
			cb(nodeToProcess)
			for _, nodeToProcess = range nodeToProcess.subNodes {
				applicator(nodeToProcess, cb)
			}
		}
	} else {
		applicator = func(nodeToProcess *localeNode, cb func(node *localeNode)) {
			nodesToProcess := make([]*localeNode, 0, 1024)
			nodesToProcess = append(nodesToProcess, nodeToProcess)

			for len(nodesToProcess) > 0 {
				nodeToProcess = nodesToProcess[0]
				nodesToProcess = nodesToProcess[:len(nodesToProcess)-1]
				cb(nodeToProcess)

				for _, nodeToProcess = range nodeToProcess.subNodes {
					nodesToProcess = append(nodesToProcess, nodeToProcess)
				}
			}
		}
	}

	applicator(n, cb)
}

/*
scan walks over passed map[string]interface{},
treating it like a source of locale's content for the current localeNode,
doing next things:

 - If a value is a basic Golang type (such as string, bool, int, uint, float, nil),
   that value is saved with corresponding key to the contentTmp
   using store() method.

 - If a value is the same type map (map[string]interface{}),
   the embedded localeNode by the corresponding key
   will be either extracted from the subNodes or created an empty new one,
   and scan() will be called recursively for that sub localeNode and that map.

 - If a value has any other type,
   it's an error, even if it's array (arrays are prohibited).

sourceItemIdx will be saved to the usedSourcesIdx,
after the whole map is successfully parsed and if there is no the same index yet.
*/
func (n *localeNode) scan(

	from          map[string]interface{},
	sourceItemIdx int,
	overwrite     bool,

) *ekaerr.Error {

	const s = "Failed to scan a key-value component."

	var err *ekaerr.Error
	for key, value := range from {
		switch rtype := reflect2.RTypeOf(value); {

		case key == "":
			err = ekaerr.IllegalFormat.
				New(s + "Key is empty.")

		case rtype == 0:
			err = n.store(key, "<undefined>", overwrite)

		case rtype == ekaunsafe.RTypeString():
			err = n.store(key, value.(string), overwrite)

		case rtype == ekaunsafe.RTypeBool():
			b := *(*bool)(ekaunsafe.TakeRealAddr(value))
			value := "false"
			if b {
				value = "true"
			}
			err = n.store(key, value, overwrite)

		case ekaunsafe.RTypeIsIntAny(rtype):
			i64 := *(*int64)(ekaunsafe.TakeRealAddr(value))
			err = n.store(key, strconv.FormatInt(i64, 10), overwrite)

		case ekaunsafe.RTypeIsUintAny(rtype):
			u64 := *(*uint64)(ekaunsafe.TakeRealAddr(value))
			err = n.store(key, strconv.FormatUint(u64, 10), overwrite)

		case ekaunsafe.RTypeIsFloatAny(rtype):
			f64 := *(*float64)(ekaunsafe.TakeRealAddr(value))
			bitSize := 32
			if rtype == ekaunsafe.RTypeFloat64() {
				bitSize = 64
			}
			err = n.store(key, strconv.FormatFloat(f64, 'f', 2, bitSize), overwrite)

		case rtype == ekaunsafe.RTypeMapStringInterface():
			embeddedMap := value.(map[string]interface{})
			err = n.subNode(key, true).scan(embeddedMap, sourceItemIdx, overwrite)

		default:
			err = ekaerr.IllegalFormat.
				New(s + "Unexpected type of value.").
				AddFields("privet_source_value_type", reflect2.TypeOf(value).String())
		}

		//goland:noinspection GoNilness
		if err.IsNotNil() {
			return err.
				AddMessage(s).
				AddFields("privet_source_key", key).
				Throw()
		}
	}

	// All is good.
	// We may proceed.

	needToAdd := true
	for _, alreadyMarkedSourceIdx := range n.usedSourcesIdx {
		if alreadyMarkedSourceIdx == sourceItemIdx {
			needToAdd = false
			break
		}
	}

	if needToAdd {
		n.usedSourcesIdx = append(n.usedSourcesIdx, sourceItemIdx)
	}

	return nil
}

/*
store saves passed key, value to the contentTmp map,
if there is no the same key yet in content map, or if overwriting is allowed.

Returns an error if overwriting is prohibited and it's a duplication.
*/
func (n *localeNode) store(key, value string, overwrite bool) *ekaerr.Error {

	// contentTmp contains only the current file processing keys;
	// it will be so strange (and impossible), if there will be the same keys.

	if _, isExist := n.content[key]; isExist && !overwrite {
		alreadyUsedSources := make([]string, len(n.usedSourcesIdx))
		for i, usedSourceIdx := range n.usedSourcesIdx {
			alreadyUsedSources[i] = n.parent.owner.sourcesTmp[usedSourceIdx].Path
		}
		return ekaerr.AlreadyExist.
			New("Failed to add new translation phrase. Already exist.").
			AddFields(
				"privet_source_applied",   strings.Join(alreadyUsedSources, ", "),
				"privet_source_key",       key,
				"privet_source_new_value", value,
				"privet_source_old_value", n.content[key]).
			Throw()
	}

	n.contentTmp[key] = value
	return nil
}
