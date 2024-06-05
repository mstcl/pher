package cli

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/entry"

	"github.com/mstcl/pher/internal/feed"
	"github.com/mstcl/pher/internal/ioutil"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/render"
	"github.com/mstcl/pher/internal/tag"
)

var Templates embed.FS

type meta struct {
	c      *config.Config
	tpl    *template.Template
	d      map[string]entry.Entry
	a      map[string]bool
	skip   map[string]bool
	l      map[string][]listing.Listing
	inDir  string
	outDir string
	files  []string
	t      []tag.Tag
	isDry  bool
}

func Parse() error {
	mt := meta{}

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

	// Get absolute paths
	inDir, err = filepath.Abs(inDir)
	if err != nil {
		return fmt.Errorf("error getting absolute path: %w", err)
	}
	outDir, err = filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("error getting absolute path: %w", err)
	}
	cfgFile, err = filepath.Abs(cfgFile)
	if err != nil {
		return fmt.Errorf("error getting absolute path: %w", err)
	}

	// Check paths
	if fExist, err := ioutil.IsFileExist(inDir); err != nil {
		return fmt.Errorf("error when stat file or directory %s: %w", inDir, err)
	} else if !fExist {
		return fmt.Errorf("no such file or directory: %s", cfgFile)
	}
	if fExist, err := ioutil.IsFileExist(cfgFile); err != nil {
		return fmt.Errorf("error when stat file or directory %s: %w", cfgFile, err)
	} else if !fExist {
		return fmt.Errorf("no such file or directory: %s", cfgFile)
	}
	if err = ioutil.EnsureDir(outDir); err != nil {
		return fmt.Errorf("make directory: %w", err)
	}
	mt.inDir = inDir
	mt.outDir = outDir

	// Handle configuration
	cfg, err := config.Read(cfgFile)
	if err != nil {
		return err
	}
	mt.c = cfg

	// Clean output directory
	if !isDry {
		contents, err := filepath.Glob(outDir + "/*")
		if err != nil {
			return fmt.Errorf("glob files: %w", err)
		}
		if err = ioutil.RemoveContents(contents); err != nil {
			return fmt.Errorf("rm files: %w", err)
		}
	}
	mt.isDry = isDry

	// Fetch templates
	tplDir := "web/template"
	tpl := template.Must(template.ParseFS(Templates, filepath.Join(tplDir, "*")))
	mt.tpl = tpl

	// Grab files and reorder so indexes are processed last
	files, err := zglob.Glob(inDir + "/**/*.md")
	if err != nil {
		return fmt.Errorf("glob files: %w", err)
	}

	files = ioutil.RemoveHiddenFiles(inDir, files)

	// Rearrange files and add to meta
	mt.files = ioutil.ReorderFiles(files)

	if err := mt.extractEntries(); err != nil {
		return err
	}

	if err := mt.entryList(); err != nil {
		return err
	}

	if err := mt.move(); err != nil {
		return err
	}

	if err := mt.render(); err != nil {
		return err
	}

	if err := mt.feed(); err != nil {
		return err
	}

	return nil
}

// Copy asset dirs/files over to outDir.
// (3) internal links are used here.
func (mt *meta) move() error {
	if err := ioutil.CopyExtraFiles(mt.inDir, mt.outDir, mt.a); err != nil {
		return err
	}
	return nil
}

// Render with (1) entry data, (2) tags data, and (4) listings
func (mt *meta) render() error {
	m := render.Meta{
		C: mt.c, InDir: mt.inDir, OutDir: mt.outDir, D: mt.d, T: mt.t,
		Templates: mt.tpl, L: mt.l, Files: mt.files, Skip: mt.skip, IsDry: mt.isDry,
	}
	if err := m.RenderAll(); err != nil {
		return err
	}
	return nil
}

// Construct and render atom feeds, need (1) entry data.
func (mt *meta) feed() error {
	m := feed.Meta{C: mt.c, InDir: mt.inDir, OutDir: mt.outDir, IsDry: mt.isDry, D: mt.d}
	atom, err := m.ConstructFeed()
	if err != nil {
		return err
	}
	if err := m.SaveFeed(atom); err != nil {
		return err
	}
	return nil
}

func (mt *meta) extractEntries() error {
	m := entry.ExtractDeps{C: mt.c, InDir: mt.inDir, OutDir: mt.outDir}
	if err := m.ExtractEntries(mt.files); err != nil {
		return err
	}

	// update meta
	mt.d = m.D
	mt.a = m.A
	mt.t = m.T
	return nil
}

func (mt *meta) entryList() error {
	m := entry.ListDeps{C: mt.c, InDir: mt.inDir, D: mt.d}
	files, err := m.List(mt.files)
	if err != nil {
		return err
	}

	// update meta
	mt.files = files
	mt.l = m.L
	mt.skip = m.Skip
	mt.d = m.D
	return nil
}
