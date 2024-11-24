package entry

import (
	"github.com/mstcl/pher/v2/internal/listing"
	"github.com/mstcl/pher/v2/internal/metadata"
)

// An entry's data, containing: the metadata, the html body, the backlinks
// (entries that mention this entry) the related links (other entries that
// share tags), and the href.
type Entry struct {
	Href         string
	Backlinks    []listing.Listing
	Relatedlinks []listing.Listing
	Body         []byte
	ChromaCSS    []byte
	Metadata     metadata.Metadata
}
