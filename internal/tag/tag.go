// Package tag defines the Tag struct
package tag

import "github.com/mstcl/pher/v3/internal/nodepathlink"

// Tag is used in extract.extractTags and render.RenderTags. Not to be
// conceptually confused with parse.Metadata.Tags !!!
//
// * Name: tag name
//
// * Count: number of references
//
// * Links: entries (represtend as listing.Listing) for a given tag
type Tag struct {
	Name  string
	Links []nodepathlink.NodePathLink
	Count int
}
