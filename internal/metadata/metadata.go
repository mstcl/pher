// Package metadata defines available fields for the frontmatter
package metadata

// Metadata contains allowed frontmatter in unmarshalled YAML.
//
// # Default values
//
// * Pinned: false
//
// * Unlisted: false
//
// * ShowHeader: true
//
// * Layout: "list"
//
// * Draft: false
//
// * TOC: false
type Metadata struct {
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Date        string   `yaml:"date"`
	DateUpdated string   `yaml:"dateUpdated"`
	Layout      string   `yaml:"layout"`
	Tags        []string `yaml:"tags"`
	Pinned      bool     `yaml:"pinned"`
	Unlisted    bool     `yaml:"unlisted"`
	Draft       bool     `yaml:"draft"`
	TOC         bool     `yaml:"toc"`
	ShowHeader  bool     `yaml:"showHeader"`
}

// Default returns the defaults for unspecified frontmatter field values
func Default() *Metadata {
	return &Metadata{
		Pinned:     false,
		Unlisted:   false,
		ShowHeader: true,
		Layout:     "list",
		Draft:      false,
		TOC:        false,
	}
}
