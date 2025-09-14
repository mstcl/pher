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

type State struct {
	Config                   *config.Config
	Templates                *template.Template
	NodeMap                  map[nodepath.NodePath]node.Node
	UserAssetMap             map[assetpath.AssetPath]bool
	SkippedNodePathMap       map[nodepath.NodePath]bool
	NodegroupWithoutIndexMap map[nodepath.NodePath]bool
	NodePathLinkMap          map[nodepath.NodePath][]nodepathlink.NodePathLink
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
		NodePathLinkMap:    make(map[nodepath.NodePath][]nodepathlink.NodePathLink),
		SkippedNodePathMap: make(map[nodepath.NodePath]bool),
		NodeTags:           []tag.Tag{},
	}
}
