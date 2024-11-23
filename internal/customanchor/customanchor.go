package customanchor

import "go.abhg.dev/goldmark/anchor"

// Custom texter for heading anchors.
type Texter struct{}

// Custom Texter function to hide anchor for level 1 headings.
func (*Texter) AnchorText(h *anchor.HeaderInfo) []byte {
	return []byte("#")
}
