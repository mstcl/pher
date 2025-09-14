package state

import (
	"html/template"

	"github.com/mstcl/pher/v2/internal/assetpath"
	"github.com/mstcl/pher/v2/internal/config"
	"github.com/mstcl/pher/v2/internal/listentry"
	"github.com/mstcl/pher/v2/internal/node"
	"github.com/mstcl/pher/v2/internal/nodepath"
	"github.com/mstcl/pher/v2/internal/tag"
)

type State struct {
	Config                 *config.Config
	Templates              *template.Template
	Nodes                  map[nodepath.NodePath]node.Node
	UserAssets             map[assetpath.AssetPath]bool
	Skip                   map[nodepath.NodePath]bool
	NodegroupsMissingIndex map[nodepath.NodePath]bool
	ListEntries            map[nodepath.NodePath][]listentry.ListEntry
	InputDir               string
	OutputDir              string
	ConfigFile             string
	NodePaths              []nodepath.NodePath
	NodeTags               []tag.Tag
	ShowVersion            bool
	Debug                  bool
	DryRun                 bool
}

func Init() State {
	return State{
		Nodes:       make(map[nodepath.NodePath]node.Node),
		UserAssets:  make(map[assetpath.AssetPath]bool),
		ListEntries: make(map[nodepath.NodePath][]listentry.ListEntry),
		Skip:        make(map[nodepath.NodePath]bool),
		NodeTags:    []tag.Tag{},
	}
}
