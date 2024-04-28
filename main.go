// pher is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// pher is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with pher.  If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/extract"
	"github.com/mstcl/pher/internal/feed"
	"github.com/mstcl/pher/internal/parse"
	"github.com/mstcl/pher/internal/render"
	"github.com/mstcl/pher/internal/util"
)

//go:embed web/template/*
var tDirFs embed.FS

func main() {
	var cfgFile, outDir, inDir string
	var err error
	var isDry bool

	flag.StringVar(
		&cfgFile,
		"c",
		"config.yaml",
		"Path to config file",
	)
	flag.StringVar(
		&inDir,
		"i",
		".",
		"Input directory",
	)
	flag.StringVar(
		&outDir,
		"o",
		"_site",
		"Output directory",
	)
	flag.BoolVar(
		&isDry,
		"d",
		false,
		"Dry run---don't render (default false)",
	)
	flag.Parse()

	// Check fs
	cfgFile, err = filepath.Abs(cfgFile)
	if err != nil {
		_ = fmt.Errorf("config file: %w", err)
	}
	inDir, err = filepath.Abs(inDir)
	if err != nil {
		_ = fmt.Errorf("input directory: %w", err)
	}
	outDir, err = filepath.Abs(outDir)
	if err != nil {
		_ = fmt.Errorf("output directory: %w", err)
	}
	err = os.Mkdir(outDir, 0775)
	if err != nil {
		_ = fmt.Errorf("make directory: %w", err)
	}

	// Handle configuration
	cfg, err := config.Read(cfgFile)
	if err != nil {
		_ = fmt.Errorf("read config: %w", err)
	}

	// Clean output directory
	if !isDry {
		contents, err := filepath.Glob(outDir + "/*")
		if err != nil {
			_ = fmt.Errorf("glob files: %w", err)
		}
		err = util.RemoveContents(contents)
		if err != nil {
			_ = fmt.Errorf("rm files: %w", err)
		}
	}

	// Grab files and reorder so indexes are processed last
	files, err := zglob.Glob(inDir + "/**/*.md")
	if err != nil {
		_ = fmt.Errorf("glob files: %w", err)
	}
	files = util.ReorderFiles(files)

	// Populate metadata, content, indexes, tags, related links, hrefs and
	// internal links
	m, c, b, t, rl, h, i, err := extract.Extract(
		files,
		inDir,
		cfg.CodeHighlight,
		cfg.IsExt,
	)
	if err != nil {
		_ = fmt.Errorf("processing files: %w", err)
	}

	// Populate listing for indexes
	// Additionally make listings if there are none
	l, missing, err := extract.ExtractIndexListing(inDir, m, cfg.IsExt)
	if err != nil {
		_ = fmt.Errorf("indexing listing: %w", err)
	}
	for k := range missing {
		files = append(files, k)
		md := parse.DefaultMetadata()
		md.Title = util.GetFilePath(util.GetRelativeFilePath(k, inDir))
		m[k] = md
	}

	// Copy asset dirs/files over to outDir
	err = util.CopyExtraFiles(inDir, outDir, i)
	if err != nil {
		_ = fmt.Errorf("mkdir: %w", err)
	}

	// Fetch templates
	tplDir := "web/template"
	tpl := template.Must(template.ParseFS(tDirFs, filepath.Join(tplDir, "*")))
	if err != nil {
		_ = fmt.Errorf("template directory: %w", err)
	}

	// Render
	err = render.RenderAll(m, c, b, l, t, rl, inDir, outDir, tpl, cfg, files,
		isDry)
	if err != nil {
		_ = fmt.Errorf("render files: %w", err)
	}

	// Make feed
	atom, err := feed.MakeFeed(*cfg, m, c, h)
	if err != nil {
		_ = fmt.Errorf("make atom feed: %w", err)
	}
	err = feed.SaveFeed(outDir, atom, isDry)
	if err != nil {
		_ = fmt.Errorf("write atom feed: %w", err)
	}
}
