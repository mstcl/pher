package listing

import "html/template"

// A listing
//
// * Href: target link
//
// * Title: the title of source (declared in frontmatter)
//
// * Description: the description of source (declared in frontmatter)
//
// * IsDir: source is directory or not
//
// The rest are for Log View, similar to render.RenderData
type Listing struct {
	Href               string
	Title              string
	Description        string
	IsDir              bool
	Body               template.HTML
	Date               string
	DateUpdated        string
	MachineDate        string
	MachineDateUpdated string
	Tags               []string
}

// A tag
//
// * Count: number of references
type Tag struct {
	Name  string
	Count int
	Links []Listing
}
