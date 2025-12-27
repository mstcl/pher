// Package state defines the asbstracted state struct
package state

import (
	"html/template"

	"github.com/mstcl/pher/v2/internal/assetpath"
	"github.com/mstcl/pher/v2/internal/config"
	"github.com/mstcl/pher/v2/internal/node"
	"github.com/mstcl/pher/v2/internal/nodepath"
	"github.com/mstcl/pher/v2/internal/nodepathlink"
	"github.com/mstcl/pher/v2/internal/tag"
)

// State struct centralises all computed values so we can pass it into child
// functions with ease.
//
// * SkippedNodePathMap: map of NodePaths that shouldn't be rendered because its
// nodegroup is of Log listing type.
//
// * NodegroupWithoutIndexMap: map of Nodegroups that don't have an index file
type State struct {
	Config                   *config.Config
	Templates                *template.Template
	NodeMap                  map[nodepath.NodePath]node.Node
	UserAssetMap             map[assetpath.AssetPath]bool
	SkippedNodePathMap       map[nodepath.NodePath]bool
	NodegroupWithoutIndexMap map[nodepath.NodePath]bool
	NodePathLinksMap         map[nodepath.NodePath][]nodepathlink.NodePathLink
	InputDir                 string
	OutputDir                string
	ConfigFile               string
	NodePaths                []nodepath.NodePath
	NodeTags                 []tag.Tag
	ShowVersion              bool
	Debug                    bool
	DryRun                   bool
}

func Init() State {
	return State{
		NodeMap:            make(map[nodepath.NodePath]node.Node),
		UserAssetMap:       make(map[assetpath.AssetPath]bool),
		NodePathLinksMap:   make(map[nodepath.NodePath][]nodepathlink.NodePathLink),
		SkippedNodePathMap: make(map[nodepath.NodePath]bool),
		NodeTags:           []tag.Tag{},
	}
}
