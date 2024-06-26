package frontmatter

import (
	"gopkg.in/yaml.v3"
)

// DefaultFormats is the list of frontmatter formats
// that are recognized by default.
var DefaultFormats = []Format{YAML}

// YAML provides support for frontmatter in the YAML format.
// Front matter in this format is expected to be delimited
// by three or more '-' characters.
//
//	---
//	title: Hello, world!
//	tags:
//	  - foo
//	  - bar
//	---
var YAML = Format{
	Name:      "YAML",
	Delim:     '-',
	Unmarshal: yaml.Unmarshal,
}

// Format defines a front matter format recognized by this package.
type Format struct {
	// Delim specifies the delimiter that marks front matter
	// in this format.
	//
	// There must be at least three of these in a row
	// for the front matter to be recognized.

	// Unmarshal unmarshals the front matter data into the provided value.
	Unmarshal func([]byte, any) error

	// Name is a human-readable name for the format.
	//
	// It may be used in error messages.
	Name string

	Delim byte
}
