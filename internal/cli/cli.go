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
	config   *config.Config
	template *template.Template
	entries  map[string]entry.Entry
	assets   map[string]bool
	skip     map[string]bool
	listings map[string][]listing.Listing
	inDir    string
	outDir   string
	files    []string
	tags     []tag.Tag
	dryRun   bool
}

func Parse() error {
	m := meta{}

	var configFile, outDir, inDir string
	var err error
	var dryRun bool

	flag.StringVar(
		&configFile,
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
		&dryRun,
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
	configFile, err = filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("error getting absolute path: %w", err)
	}

	// Check paths
	if fileExist, err := ioutil.IsFileExist(inDir); err != nil {
		return fmt.Errorf("error when stat file or directory %s: %w", inDir, err)
	} else if !fileExist {
		return fmt.Errorf("no such file or directory: %s", configFile)
	}
	if fileExist, err := ioutil.IsFileExist(configFile); err != nil {
		return fmt.Errorf("error when stat file or directory %s: %w", configFile, err)
	} else if !fileExist {
		return fmt.Errorf("no such file or directory: %s", configFile)
	}
	if err = ioutil.EnsureDir(outDir); err != nil {
		return fmt.Errorf("make directory: %w", err)
	}
	m.inDir = inDir
	m.outDir = outDir

	// Handle configuration
	cfg, err := config.Read(configFile)
	if err != nil {
		return err
	}
	m.config = cfg

	// Clean output directory
	if !dryRun {
		contents, err := filepath.Glob(outDir + "/*")
		if err != nil {
			return fmt.Errorf("glob files: %w", err)
		}
		if err = ioutil.RemoveContents(contents); err != nil {
			return fmt.Errorf("rm files: %w", err)
		}
	}
	m.dryRun = dryRun

	// Fetch templates
	tplDir := "web/template"
	tpl := template.Must(template.ParseFS(Templates, filepath.Join(tplDir, "*")))
	m.template = tpl

	// Grab files and reorder so indexes are processed last
	files, err := zglob.Glob(inDir + "/**/*.md")
	if err != nil {
		return fmt.Errorf("glob files: %w", err)
	}

	files = ioutil.RemoveHiddenFiles(inDir, files)

	// Rearrange files and add to meta
	m.files = ioutil.ReorderFiles(files)

	if err := m.extractEntries(); err != nil {
		return err
	}

	if err := m.entryList(); err != nil {
		return err
	}

	if err := m.move(); err != nil {
		return err
	}

	if err := m.render(); err != nil {
		return err
	}

	if err := m.feed(); err != nil {
		return err
	}

	return nil
}

// Copy asset dirs/files over to outDir.
// (3) internal links are used here.
func (m *meta) move() error {
	if err := ioutil.CopyExtraFiles(m.inDir, m.outDir, m.assets); err != nil {
		return err
	}
	return nil
}

// Render with (1) entry data, (2) tags data, and (4) listings
func (m *meta) render() error {
	d := render.RenderDeps{
		Config: m.config, InDir: m.inDir, OutDir: m.outDir, Entries: m.entries, Tags: m.tags,
		Templates: m.template, Listings: m.listings, Files: m.files, Skip: m.skip, DryRun: m.dryRun,
	}
	if err := d.RenderAll(); err != nil {
		return err
	}
	return nil
}

// Construct and render atom feeds, need (1) entry data.
func (m *meta) feed() error {
	d := feed.FeedDeps{Config: m.config, InDir: m.inDir, OutDir: m.outDir, DryRun: m.dryRun, Entries: m.entries}
	atom, err := d.ConstructFeed()
	if err != nil {
		return err
	}
	if err := d.SaveFeed(atom); err != nil {
		return err
	}
	return nil
}

func (m *meta) extractEntries() error {
	d := entry.ExtractDeps{Config: m.config, InDir: m.inDir, OutDir: m.outDir}
	if err := d.ExtractEntries(m.files); err != nil {
		return err
	}

	// update meta
	m.entries = d.Entries
	m.assets = d.Assets
	m.tags = d.Tags
	return nil
}

func (m *meta) entryList() error {
	d := entry.ListDeps{Config: m.config, InDir: m.inDir, Entries: m.entries}
	files, err := d.List(m.files)
	if err != nil {
		return err
	}

	// update meta
	m.files = files
	m.listings = d.Listings
	m.skip = d.Skip
	m.entries = d.Entries
	return nil
}
