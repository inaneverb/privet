// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

import (
	"bytes"
	"crypto/md5"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/qioalice/ekago/v2/ekaerr"
	"github.com/qioalice/ekago/v2/ekaunsafe"

	"github.com/modern-go/reflect2"
)

//goland:noinspection GoSnakeCaseUsage
const (
	/*
		Source() func and its private part, a sourceString() may scan a directory
		you specify recursively,
		meaning that if an original directory has a subdirectory(ies),
		it will be scanned also and so on.
		Up to this value.
	*/
	_SOURCE_MAX_RECURSIVELY_DIRECTORY_SCAN = 16
)

/*
source literally does things Client.Source() method describes.

All logic moved to the private method, because of changing signature.
Original Client.Source() ready to get variadic number of interface{} arguments,
but this source() method wants []interface{}.
Thus, calling source(args) as an once statement of Source() does not lead
to unnecessary copying. So for package level's Source() function.
*/
func (c *Client) source(args []interface{}) *ekaerr.Error {
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
sourceString tries to treat s as a path to file or directory.
The logic depends on whether it's a file or directory.

File.
If s is a filepath, and the file can be used as a locale source
(file is exist, access granted, file is not empty, file is valid) then it's OK.
A new _SourceItem for that file is created and placed into dest.
Argument deep is ignored for that case.

Directory.
If s is a path to the directory, the list of files and included directories
will be created, and sourceString() will be called recursively for each that item.
In that case deep is increased at the each recursive iteration,
until _SOURCE_MAX_RECURSIVELY_DIRECTORY_SCAN. When max is reached, error is returned.
For all included directories, sourceString() is also called recursively.
For all found locale files a new _SourceItem objects will be created and placed
into dest.
Caller must call sourceString() with deep == 0.

There is no check or any validation of file's content.
It will be validated at the Load() call (and its internal parts).
*/
func (c *Client) sourceString(dest *[]SourceItem, source string, deep int) *ekaerr.Error {
	const s = "Failed to analyse provided path as a locale source. "

	if source = strings.TrimSpace(source); source == "" {
		return ekaerr.IllegalArgument.
			New(s + "Path is empty.").
			Throw()
	}

	if !filepath.IsAbs(source) {
		if workDir, legacyErr := os.Getwd(); legacyErr == nil {
			source = filepath.Join(workDir, source)
		} else {
			return ekaerr.InternalError.
				Wrap(legacyErr, s + "Got relative path, failed to get work directory.").
				AddFields("privet_source_rel_path", source).
				Throw()
		}
	}

	if deep == 0 {
		source = filepath.Clean(source)
	}
	
	var (
		f         *os.File
		fi        os.FileInfo
		fis       []os.FileInfo
		legacyErr error
	)

	if f, legacyErr = os.Open(source); legacyErr != nil {
		return ekaerr.DataUnavailable.
			Wrap(legacyErr, s + "Failed to open provided path.").
			AddFields("privet_source_path", source).
			Throw()
	}

	// Explanation:
	// There is no deferring of f.Close(), because of this function is recursive.
	// I don't want to open many file descriptors recursively and then close them
	// at once.
	// The logic is that this function is recursive about moving into the deep,
	// but is iterative about opening/closing file descriptors.
	
	if fi, legacyErr = f.Stat(); legacyErr != nil {
		//goland:noinspection GoUnhandledErrorResult
		f.Close()
		return ekaerr.DataUnavailable.
			Wrap(legacyErr, s + "Opening path is successful but getting stat is failed.").
			AddFields("privet_source_path", source).
			Throw()
	}
	
	if !fi.IsDir() {

		// Ignore files that has an unsupported extension.

		ext := strings.ToLower(filepath.Ext(source))
		if ext != "" {
			ext = ext[1:]
		}

		switch ext {
		case "toml", "yml", "yaml":
			break
		case "":
			fallthrough
		default:
			return nil
		}

		h := md5.New()

		// We don't have Client's fields initialization.
		// So, initialize buf here if it's not yet so.
		if c.buf.Cap() == 0 {
			c.buf.Grow(64 * 1024)
		}
		c.buf.Reset()

		// We will:
		//  - read file storing its data to the RAM
		//  - calculate MD5 hash sum
		// at the one iteration, chunk by chunk.
		mw := io.MultiWriter(h, &c.buf)

		if _, legacyErr = io.Copy(mw, f); legacyErr != nil {
			return ekaerr.DataUnavailable.
				Wrap(legacyErr, s + "Failed to read file and calculate its MD5 hash sum.").
				AddFields("privet_source_path", source).
				Throw()
		}

		//goland:noinspection GoUnhandledErrorResult
		f.Close()

		md5sum := h.Sum(nil)
		b := append([]byte(nil), c.buf.Bytes()...)

		switch ext {
		case "yml", "yaml":
			c.sourceApprove(dest, SOURCE_ITEM_TYPE_FILE_YAML, source, b, md5sum)
		case "toml":
			c.sourceApprove(dest, SOURCE_ITEM_TYPE_FILE_TOML, source, b, md5sum)
		default:
			// You should never see this error, because otherwise it's a bug.
			return ekaerr.InternalError.
				New(s + "Unexpected extension of sourced document. This is a bug.").
				AddFields("privet_source_path", source).
				Throw()
		}

		return nil
	}

	// Ok, it's directory.

	if deep == _SOURCE_MAX_RECURSIVELY_DIRECTORY_SCAN {
		//goland:noinspection GoUnhandledErrorResult
		f.Close()
		return ekaerr.DataUnavailable.
			New(s + "Provided path contains too much nested directories.").
			AddFields("privet_source_path", source).
			Throw()
	}

	fis, legacyErr = f.Readdir(-1)

	//goland:noinspection GoUnhandledErrorResult
	f.Close()

	if legacyErr != nil {
		return ekaerr.DataUnavailable.
			Wrap(legacyErr, s + "Failed to scan a directory.").
			AddFields("privet_source_path", source).
			Throw()
	}

	for _, fi := range fis {

		// Before we gonna do a recursive call we need to construct full absolute path
		// to each included item in the current directory under processing.
		source := filepath.Join(source, fi.Name())

		if err := c.sourceString(dest, source, deep+1); err.IsNotNil() {
			return err.
				Throw()
		}
	}

	return nil
}

/*
sourceBytes creates a new _SourceItem for passed bytearray if it's not empty
and placed into dest.
There is no check or any validation of the byte content.
It will be validated at the Load() call (and its internal parts).
*/
func (c *Client) sourceBytes(dest *[]SourceItem, b []byte) *ekaerr.Error {
	const s = "Failed to analyse provided RAW data as a locale source. "

	_, file, lineNumber, ok := runtime.Caller(2)
	if ok && file != "" {
		file = ":" + strconv.Itoa(lineNumber)
	} else {
		file = "Source undefined. Failed to extract caller information."
	}

	if len(b) == 0 {
		return ekaerr.IllegalFormat.
			New(s + "Empty RAW data.").
			AddFields("privet_source_path", file).
			Throw()
	}

	h := md5.New()

	if _, legacyErr := io.Copy(h, bytes.NewBuffer(b)); legacyErr != nil {
		return ekaerr.DataUnavailable.
			Wrap(legacyErr, s + "Failed to copy RAW data and calculate its MD5 hash sum.").
			AddFields("privet_source_path", file).
			Throw()
	}

	md5sum := h.Sum(nil)

	c.sourceApprove(dest, SOURCE_ITEM_TYPE_CONTENT_UNKNOWN, file, b, md5sum)
	return nil
}

/*
sourceApprove is just _SourceItem constructor with passed typ, path, content arguments
and appender to the dest.
*/
func (_ *Client) sourceApprove(dest *[]SourceItem, typ SourceItemType, path string, content, md5sum []byte) {
	*dest = append(*dest, SourceItem{
		Type:    typ,
		Path:    path,
		content: content,
		md5:     string(md5sum),
	})
}
