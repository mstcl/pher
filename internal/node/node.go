// Package node defines the Node struct
package node

import (
	"github.com/mstcl/pher/v2/internal/metadata"
	"github.com/mstcl/pher/v2/internal/nodepathlink"
)

// Node is an abstracted idea of a source markdown file. It is a file
// represented in our state.
type Node struct {
	Href         string
	Backlinks    []nodepathlink.NodePathLink
	Relatedlinks []nodepathlink.NodePathLink
	Body         []byte
	ChromaCSS    []byte
	Metadata     metadata.Metadata
}
