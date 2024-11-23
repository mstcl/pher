package cli

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/internal/config"
	"github.com/mstcl/pher/internal/render"
	"github.com/mstcl/pher/internal/state"
	"golang.org/x/sync/errgroup"

	"github.com/mstcl/pher/internal/checks"
	"github.com/mstcl/pher/internal/feed"
)

const templateDir = "web/template"

var (
	Templates                 embed.FS
	configFile, outDir, inDir string
	dryRun                    bool
)

func Parse() error {
	var err error

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
		"Don't render (dry run)",
	)
	flag.Parse()

	// This is pher's s
	s := state.Init()
	s.DryRun = dryRun

	// Sanitize input directory
	inDir, err = filepath.Abs(inDir)
	if err != nil {
		return fmt.Errorf("absolute path: %w", err)
	}

	if err = checks.DirExist(outDir); err != nil {
		return fmt.Errorf("output directory: %w", err)
	}

	s.InDir = inDir

	// Sanitize output directory
	outDir, err = filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("absolute path: %w", err)
	}

	if err = checks.DirExist(outDir); err != nil {
		return fmt.Errorf("output directory: %w", err)
	}

	s.OutDir = outDir

	// Sanitize configuration file
	configFile, err = filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("absolute path: %w", err)
	}

	if fileExist, err := checks.FileExist(configFile); err != nil {
		return fmt.Errorf("stat: %v", err)
	} else if !fileExist {
		return fmt.Errorf("missing: %s", configFile)
	}

	// Read configuration
	s.Config, err = config.Read(configFile)
	if err != nil {
		return err
	}

	// Clean output directory
	if !s.DryRun {
		files, err := filepath.Glob(outDir + "/*")
		if err != nil {
			return fmt.Errorf("glob files: %w", err)
		}

		if err = removeFiles(files); err != nil {
			return fmt.Errorf("rm files: %w", err)
		}
	}

	// Initiate templates
	s.Templates = template.Must(template.ParseFS(
		Templates, filepath.Join(templateDir, "*")))

	// Grab files and reorder so indexes are processed last
	files, err := zglob.Glob(s.InDir + "/**/*.md")
	if err != nil {
		return fmt.Errorf("glob files: %w", err)
	}

	// Finalize list of files to process
	files = filterHiddenFiles(inDir, files)
	s.Files = reorderFiles(files)


	// Update the state with various metadata
	if err := extractExtras(&s); err != nil {
		return err
	}

	// Update the state with file listings, like backlinks and similar entries
	if err := makeFileListing(&s); err != nil {
		return err
	}

	// NOTE: The next three processes can run concurrently as they are
	// independent from each other

	// Construct and render atom feeds
	feedGroup, _ := errgroup.WithContext(context.Background())
	feedGroup.Go(func() error {
		atom, err := feed.Construct(&s)
		if err != nil {
			return err
		}

		return feed.Write(&s, atom)
	},
	)

	// Copy asset dirs/files over to output directory
	moveGroup, _ := errgroup.WithContext(context.Background())
	moveGroup.Go(func() error {
		if err := syncAssets(context.Background(), &s); err != nil {
			return err
		}

		return nil
	},
	)

	// Create beautiful HTML
	renderGroup, _ := errgroup.WithContext(context.Background())
	renderGroup.Go(func() error {
		return render.RenderAll(context.Background(), &s)
	})

	// Wait for all goroutines to finish
	if err := feedGroup.Wait(); err != nil {
		return err
	}

	if err := moveGroup.Wait(); err != nil {
		return err
	}

	if err := renderGroup.Wait(); err != nil {
		return err
	}

	return nil
}
