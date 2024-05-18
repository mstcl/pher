package cli

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/entry"
	"github.com/mstcl/pher/internal/extract"
	"github.com/mstcl/pher/internal/feed"
	"github.com/mstcl/pher/internal/listing"
	"github.com/mstcl/pher/internal/render"
	"github.com/mstcl/pher/internal/tag"
	"github.com/mstcl/pher/internal/util"
)

var Templates embed.FS

type meta struct {
	c      *config.Config
	t      *template.Template
	inDir  string
	outDir string
	files  []string
	isDry  bool
}

func Parse() {
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
		log.Fatal(fmt.Errorf("error getting absolute path: %w", err))
	}
	outDir, err = filepath.Abs(outDir)
	if err != nil {
		log.Fatal(fmt.Errorf("error getting absolute path: %w", err))
	}
	cfgFile, err = filepath.Abs(cfgFile)
	if err != nil {
		log.Fatal(fmt.Errorf("error getting absolute path: %w", err))
	}
	if err != nil {
		log.Fatal(fmt.Errorf("error get absolute paths: %w", err))
	}

	// Check paths
	if fExist, err := util.IsFileExist(inDir); err != nil {
		log.Fatal(fmt.Errorf("error when stat file or directory %s: %w", inDir, err))
	} else if !fExist {
		log.Fatal(fmt.Errorf("no such file or directory: %s", cfgFile))
	}
	if fExist, err := util.IsFileExist(cfgFile); err != nil {
		log.Fatal(fmt.Errorf("error when stat file or directory %s: %w", cfgFile, err))
	} else if !fExist {
		log.Fatal(fmt.Errorf("no such file or directory: %s", cfgFile))
	}
	if err = util.EnsureDir(outDir); err != nil {
		log.Fatal(fmt.Errorf("make directory: %w", err))
	}
	mt.inDir = inDir
	mt.outDir = outDir

	// Handle configuration
	cfg, err := config.Read(cfgFile)
	if err != nil {
		log.Fatal(err)
	}
	mt.c = cfg

	// Clean output directory
	if !isDry {
		contents, err := filepath.Glob(outDir + "/*")
		if err != nil {
			log.Fatal(fmt.Errorf("glob files: %w", err))
		}
		if err = util.RemoveContents(contents); err != nil {
			log.Fatal(fmt.Errorf("rm files: %w", err))
		}
	}
	mt.isDry = isDry

	// Fetch templates
	tplDir := "web/template"
	tpl := template.Must(template.ParseFS(Templates, filepath.Join(tplDir, "*")))
	mt.t = tpl

	// Grab files and reorder so indexes are processed last
	files, err := zglob.Glob(inDir + "/**/*.md")
	if err != nil {
		log.Fatal(fmt.Errorf("glob files: %w", err))
	}
	mt.files = util.ReorderFiles(files)

	d, t, i, l, skip := mt.extract()
	mt.move(i)
	mt.render(d, t, l, skip)
	mt.feed(d)
}

// Grab data numbered (1) - (4)
func (mt *meta) extract() (
	map[string]entry.Entry,
	[]tag.Tag,
	map[string]bool,
	map[string][]listing.Listing,
	map[string]bool,
) {
	// Populate (1) entry data, (2) tags data, and (3) internal links.
	m := extract.Meta{C: mt.c, InDir: mt.inDir, OutDir: mt.outDir}
	d, t, i, err := m.ExtractEntry(mt.files)
	if err != nil {
		log.Fatal(err)
	}
	m.D = d

	// Get (4) listings, needs (1) entry data
	files, l, skip, err := m.ExtractAllListings(mt.files)
	if err != nil {
		log.Fatal(err)
	}
	mt.files = files
	return d, t, i, l, skip
}

// Copy asset dirs/files over to outDir.
// (3) internal links are used here.
func (mt *meta) move(i map[string]bool) {
	if err := util.CopyExtraFiles(mt.inDir, mt.outDir, i); err != nil {
		log.Fatal(err)
	}
}

// Render with (1) entry data, (2) tags data, and (4) listings
func (mt *meta) render(
	d map[string]entry.Entry,
	t []tag.Tag,
	l map[string][]listing.Listing,
	skip map[string]bool,
) {
	m := render.Meta{
		C: mt.c, InDir: mt.inDir, OutDir: mt.outDir, D: d, T: t,
		Templates: mt.t, L: l, Files: mt.files, Skip: skip, IsDry: mt.isDry,
	}
	if err := m.RenderAll(); err != nil {
		log.Fatal(err)
	}
}

// Construct and render atom feeds, need (1) entry data.
func (mt *meta) feed(d map[string]entry.Entry) {
	m := feed.Meta{C: mt.c, InDir: mt.inDir, OutDir: mt.outDir, IsDry: mt.isDry, D: d}
	atom, err := m.ConstructFeed()
	if err != nil {
		log.Fatal(err)
	}
	if err := m.SaveFeed(atom); err != nil {
		log.Fatal(err)
	}
}
