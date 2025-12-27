// Package nodepathlink defines the nodepathlink struct
package nodepathlink

import "html/template"

// NodePathLink is a link to another nodegroup or node
// Each nodegroup/node has its a list of nodepath links
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
type NodePathLink struct {
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
