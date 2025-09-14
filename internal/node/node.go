package node

import (
	"github.com/mstcl/pher/v2/internal/listentry"
	"github.com/mstcl/pher/v2/internal/metadata"
)

// Node is an abstracted idea of a source markdown file. It is a file
// represented in our state.
type Node struct {
	Href         string
	Backlinks    []listentry.ListEntry
	Relatedlinks []listentry.ListEntry
	Body         []byte
	ChromaCSS    []byte
	Metadata     metadata.Metadata
}
