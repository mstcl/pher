package state

import (
	"html/template"

	"github.com/mstcl/pher/v2/internal/config"
	"github.com/mstcl/pher/v2/internal/entry"
	"github.com/mstcl/pher/v2/internal/listing"
	"github.com/mstcl/pher/v2/internal/tag"
)

type State struct {
	Config      *config.Config
	Templates   *template.Template
	Entries     map[string]entry.Entry
	Assets      map[string]bool
	Skip        map[string]bool
	Missing     map[string]bool
	Listings    map[string][]listing.Listing
	InDir       string
	OutDir      string
	ConfigFile  string
	Files       []string
	Tags        []tag.Tag
	ShowVersion bool
	Debug       bool
	DryRun      bool
}

func Init() State {
	return State{
		Entries:  make(map[string]entry.Entry),
		Assets:   make(map[string]bool),
		Listings: make(map[string][]listing.Listing),
		Skip:     make(map[string]bool),
		Tags:     []tag.Tag{},
	}
}
