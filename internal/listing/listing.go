package listing

// A listing
//
// * Href: target link
//
// * Title: the title of source (declared in frontmatter)
//
// * Description: the description of source (declared in frontmatter)
//
// * IsDir: source is directory or not
type Listing struct {
	Href        string
	Title       string
	Description string
	IsDir       bool
}
