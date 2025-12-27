// Package customanchor defines a custom header anchor for goldmark/anchor
package customanchor

import "go.abhg.dev/goldmark/anchor"

// Texter is the custom texter for heading anchors.
type Texter struct{}

// AnchorText is the custom Texter function to hide anchor for level
// 1 headings.
func (*Texter) AnchorText(h *anchor.HeaderInfo) []byte {
	return []byte("#")
}
