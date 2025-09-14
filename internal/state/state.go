package state

import (
	"html/template"

	"github.com/mstcl/pher/v2/internal/config"
	"github.com/mstcl/pher/v2/internal/listentry"
	"github.com/mstcl/pher/v2/internal/node"
	"github.com/mstcl/pher/v2/internal/tag"
)

type State struct {
	Config                 *config.Config
	Templates              *template.Template
	Nodes                  map[string]node.Node
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
		Nodes:       make(map[string]node.Node),
		UserAssets:  make(map[string]bool),
		ListEntries: make(map[string][]listentry.ListEntry),
		Skip:        make(map[string]bool),
		NodeTags:    []tag.Tag{},
	}
}
