package metadata

// Allowed frontmatter in unmarshalled YAML.
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
