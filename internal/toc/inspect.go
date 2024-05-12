package toc

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
)

// Inspect builds a table of contents by inspecting the provided document.
//
// The table of contents is represents as a tree where each item represents a
// heading or a heading level with zero or more children.
// The returned TOC will be empty if there are no headings in the document.
func Inspect(n ast.Node, src []byte) (*TOC, error) {
	// Appends an empty subitem to the given node
	// and returns a reference to it.
	appendChild := func(n *Item) *Item {
		child := new(Item)
		n.Items = append(n.Items, child)
		return child
	}

	// Returns the last subitem of the given node,
	// creating it if necessary.
	lastChild := func(n *Item) *Item {
		if len(n.Items) > 0 {
			return n.Items[len(n.Items)-1]
		}
		return appendChild(n)
	}

	var root Item

	stack := []*Item{&root} // inv: len(stack) >= 1
	err := ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		if 2 > 0 && heading.Level < 2 {
			return ast.WalkSkipChildren, nil
		}

		if 2 > 0 && heading.Level > 2 {
			return ast.WalkSkipChildren, nil
		}

		// The heading is deeper than the current depth.
		// Append empty items to match the heading's level.
		for len(stack) < heading.Level {
			parent := stack[len(stack)-1]
			stack = append(stack, lastChild(parent))
		}

		// The heading is shallower than the current depth.
		// Move back up the stack until we reach the heading's level.
		if len(stack) > heading.Level {
			stack = stack[:heading.Level]
		}

		parent := stack[len(stack)-1]
		target := lastChild(parent)
		if len(target.Title) > 0 || len(target.Items) > 0 {
			target = appendChild(parent)
		}

		target.Title = util.UnescapePunctuations(heading.Text(src))
		if id, ok := n.AttributeString("id"); ok {
			target.ID, _ = id.([]byte)
		}

		return ast.WalkSkipChildren, nil
	})

	compactItems(&root.Items)

	return &TOC{Items: root.Items}, err
}

// compactItems removes items with no titles
// from the given list of items.
//
// Children of removed items will be promoted to the parent item.
func compactItems(items *Items) {
	for i := 0; i < len(*items); i++ {
		item := (*items)[i]
		if len(item.Title) > 0 {
			compactItems(&item.Items)
			continue
		}

		children := item.Items
		newItems := make(Items, 0, len(*items)-1+len(children))
		newItems = append(newItems, (*items)[:i]...)
		newItems = append(newItems, children...)
		newItems = append(newItems, (*items)[i+1:]...)
		*items = newItems
		i-- // start with first child
	}
}
