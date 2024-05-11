package listing

import "html/template"

// A listing (widely used: e.g. entry archive, links, etc.)
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
	Body               template.HTML
	Href               string
	Title              string
	Description        string
	Date               string
	DateUpdated        string
	MachineDate        string
	MachineDateUpdated string
	Tags               []string
	IsDir              bool
}
