package state

import (
	"html/template"

	"github.com/mstcl/pher/v2/internal/config"
	"github.com/mstcl/pher/v2/internal/entry"
	"github.com/mstcl/pher/v2/internal/listentry"
	"github.com/mstcl/pher/v2/internal/tag"
)

type State struct {
	Config                 *config.Config
	Templates              *template.Template
	Entries                map[string]entry.Entry
	UserAssets             map[string]bool
	Skip                   map[string]bool
	NodegroupsMissingIndex map[string]bool
	ListEntries            map[string][]listentry.ListEntry
	InputDir               string
	OutputDir              string
	ConfigFile             string
	NodePaths              []string
	NodeTags               []tag.Tag
	ShowVersion            bool
	Debug                  bool
	DryRun                 bool
}

func Init() State {
	return State{
		Entries:     make(map[string]entry.Entry),
		UserAssets:  make(map[string]bool),
		ListEntries: make(map[string][]listentry.ListEntry),
		Skip:        make(map[string]bool),
		NodeTags:    []tag.Tag{},
	}
}
