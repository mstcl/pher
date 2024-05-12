package tag

import "github.com/mstcl/pher/internal/listing"

// A tag struct for extract.extractTags and render.RenderTags. Not to be
// conceptually confused with parse.Metadata.Tags !!!
//
// * Name: tag name
//
// * Count: number of references
//
// * Links: entries (represtend as listing.Listing) for a given tag
type Tag struct {
	Name  string
	Links []listing.Listing
	Count int
}
