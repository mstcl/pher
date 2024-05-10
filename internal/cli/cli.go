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
	"github.com/mstcl/pher/internal/extract"
	"github.com/mstcl/pher/internal/feed"
	"github.com/mstcl/pher/internal/render"
	"github.com/mstcl/pher/internal/util"
)

var Templates embed.FS

func Parse() {
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
		log.Fatal(fmt.Errorf("getting absolute path: %w", err))
	}
	outDir, err = filepath.Abs(outDir)
	if err != nil {
		log.Fatal(fmt.Errorf("getting absolute path: %w", err))
	}
	cfgFile, err = filepath.Abs(cfgFile)
	if err != nil {
		log.Fatal(fmt.Errorf("getting absolute path: %w", err))
	}
	if err != nil {
		log.Fatal(fmt.Errorf("get absolute paths: %w", err))
	}

	// Check paths
	if fExist, err := util.IsFileExist(inDir); err != nil {
		log.Fatal(fmt.Errorf("stat paths: %w", err))
	} else if !fExist {
		log.Fatal(fmt.Errorf("no such file or directory: %s", cfgFile))
	}
	if fExist, err := util.IsFileExist(cfgFile); err != nil {
		log.Fatal(fmt.Errorf("stat paths: %w", err))
	} else if !fExist {
		log.Fatal(fmt.Errorf("no such file or directory: %s", cfgFile))
	}
	if err = util.EnsureDir(outDir); err != nil {
		log.Fatal(fmt.Errorf("make directory: %w", err))
	}

	// Handle configuration
	cfg, err := config.Read(cfgFile)
	if err != nil {
		log.Fatal(fmt.Errorf("read config: %w", err))
	}

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

	// Grab files and reorder so indexes are processed last
	files, err := zglob.Glob(inDir + "/**/*.md")
	if err != nil {
		log.Fatal(fmt.Errorf("glob files: %w", err))
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
		log.Fatal(fmt.Errorf("processing files: %w", err))
	}

	// Get listings
	files, l, skip, err := extract.FetchListingsCreateMissing(files, inDir, m, c, cfg.IsExt)
	if err != nil {
		log.Fatal(fmt.Errorf("extracting listings for files: %w", err))
	}

	// Copy asset dirs/files over to outDir
	if err = util.CopyExtraFiles(inDir, outDir, i); err != nil {
		log.Fatal(fmt.Errorf("mkdir: %w", err))
	}

	// Fetch templates
	tplDir := "web/template"
	tpl := template.Must(template.ParseFS(Templates, filepath.Join(tplDir, "*")))
	if err != nil {
		log.Fatal(fmt.Errorf("template directory: %w", err))
	}

	// Render
	if err = render.RenderAll(m, c, b, l, t, rl, inDir,
		outDir, tpl, cfg, files, isDry, skip); err != nil {
		log.Fatal(fmt.Errorf("render files: %w", err))
	}

	// Get atom feeds
	if err = feed.FetchFeed(*cfg, m, c, h, outDir, isDry); err != nil {
		log.Fatal(err)
	}
}
