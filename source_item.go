// Copyright © 2020. All rights reserved.
// Author: Ilya Stroy.
// Contacts: qioalice@gmail.com, https://github.com/qioalice
// License: https://opensource.org/licenses/MIT

package privet

type (
	/*
	SourceItem is a type that represents one thing that will be used as a source
	locale data will be load from.

	It may represent a file or a RAW data (content).
	Since we're supporting only YAML and TOML formats for now,
	SourceItem could be a link to either YAML or TOML file
	or just holds a content of that.

	If SourceItem represents a file, Path contains a absolute filepath to that,
	content is nil.
	If SourceItem represents a some locale content, content contains that,
	but Path contain an absolute path of the Go source file that calls Source()
	with the line number.

	SourceItem doesn't mean that source it holds is valid.
	*/
	SourceItem struct {
		Type       SourceItemType
		Path       string
		LocaleName string
		content    []byte
		md5        string
	}

	/*
	SourceItemType allows you to know which data SourceItem holds:
	A file? A RAW data? Which format? YAML? TOML?
	SourceItemType allows SourceItem to get an answers about itself.
	*/
	SourceItemType uint8
)

//goland:noinspection GoSnakeCaseUsage
const (
	/*
	There are a constants of SourceItemType.
	In the source code you may determine using these constants what SourceItem is.
	*/
	SOURCE_ITEM_TYPE_FILE_YAML       SourceItemType = 100
	SOURCE_ITEM_TYPE_FILE_TOML       SourceItemType = 101
	SOURCE_ITEM_TYPE_CONTENT_UNKNOWN SourceItemType = 150
	SOURCE_ITEM_TYPE_CONTENT_YAML    SourceItemType = 151
	SOURCE_ITEM_TYPE_CONTENT_TOML    SourceItemType = 151
)
