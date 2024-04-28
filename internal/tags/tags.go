package tags

import (
	"github.com/mstcl/pher/internal/listing"
	"sort"
)

// A tag
//
// * Count: number of references
type Tag struct {
	Name  string
	Count int
	Links []listing.Listing
}

// Transform map t to list of sorted tags
func GatherTags(tc map[string]int, tl map[string][]listing.Listing) []Tag {
	tags := []Tag{}
	keys := make([]string, 0, len(tc))
	for k := range tc {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		tags = append(tags, Tag{Name: k, Count: tc[k], Links: tl[k]})
	}
	return tags
}
