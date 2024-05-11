package entry

import (
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/parse"
)

// An entry's data, containing: the metadata, the html body, the backlinks
// (entries that mention this entry) the related links (other entries that
// share tags), and the href.
type Entry struct {
	Metadata           parse.Metadata
	Body               []byte
	Backlinks          []listing.Listing
	Relatedlinks       []listing.Listing
	Href               string
}
