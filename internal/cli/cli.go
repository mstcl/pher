// Package cli [TODO]
package cli

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/v2/internal/config"
	"github.com/mstcl/pher/v2/internal/render"
	"github.com/mstcl/pher/v2/internal/state"
	"golang.org/x/sync/errgroup"

	"github.com/mstcl/pher/v2/internal/checks"
	"github.com/mstcl/pher/v2/internal/feed"
)

const (
	relTemplateDir     = "web/template"
	relStaticDir       = "web/static"
	relStaticOutputDir = "static"
	version            = "v2.3.2"
)

var (
	EmbedFS                   embed.FS
	configFile, outDir, inDir string
	dryRun                    bool
	showVersion               bool
	debug                     bool
)

func Parse() error {
	start := time.Now()

	var err error

	flag.BoolVar(
		&showVersion,
		"v",
		false,
		"Show version and exit",
	)

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
	flag.BoolVar(
		&debug,
		"debug",
		false,
		"Verbose (debug) mode",
	)
	flag.Parse()

	var lvl slog.Level

	if debug {
		lvl = slog.LevelDebug
	} else {
		lvl = slog.LevelInfo
	}

	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      lvl,
		TimeFormat: time.Kitchen,
	}))

	logger.Debug("parsed flags",
		slog.String("inDir", inDir),
		slog.String("outDir", outDir),
		slog.String("configFile", configFile),
		slog.Bool("version", showVersion),
		slog.Bool("dryRun", dryRun),
		slog.Bool("debug", debug),
	)

	if showVersion {
		fmt.Printf("pher %v\n", version)

		return nil
	}

	// Global state
	s := state.Init()
	s.DryRun = dryRun

	// Sanitize input directory
	inDir, err = filepath.Abs(inDir)
	if err != nil {
		return fmt.Errorf("filepath.Abs: %w", err)
	}

	s.InDir = inDir

	logger.Debug("sanitized input directory", slog.String("path", inDir))

	// Sanitize output directory
	outDir, err = filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("filepath.Abs: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll %s: %w", outDir, err)
	}

	s.OutDir = outDir

	logger.Debug("sanitized output directory", slog.String("path", outDir))

	// Sanitize configuration file
	configFile, err = filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("absolute path: %w", err)
	}

	if fileExist, err := checks.FileExist(configFile); err != nil {
		return fmt.Errorf("stat: %w", err)
	} else if !fileExist {
		return fmt.Errorf("missing: %s", configFile)
	}

	logger.Debug("sanitized config file", slog.String("path", configFile))

	// Read configuration
	s.Config, err = config.Read(configFile)
	if err != nil {
		return err
	}

	logger.Debug("read configuration", slog.Any("config", s.Config))

	// Clean output directory
	if !s.DryRun {
		logger.Info("cleaning output directory")

		// TODO: update this so it ignores staticOutputDir
		files, err := filepath.Glob(outDir + "/*")
		if err != nil {
			return fmt.Errorf("glob files: %w", err)
		}

		if err = removeFiles(files); err != nil {
			return fmt.Errorf("rm files: %w", err)
		}
	} else {
		logger.Debug("dry run: skipped cleaning output directory")
	}

	// Initiate templates

	// TODO: split this out to separate package
	funcMap := template.FuncMap{
		"joinPath": path.Join,
	}

	tmpl := template.New("main")
	tmpl = tmpl.Funcs(funcMap)
	s.Templates = template.Must(tmpl.ParseFS(EmbedFS, filepath.Join(relTemplateDir, "*")))

	logger.Debug("loaded and initialized templates")

	// Grab files and reorder so indexes are processed last
	files, err := zglob.Glob(s.InDir + "/**/*.md")
	if err != nil {
		return fmt.Errorf("glob files: %w", err)
	}

	files = filterHiddenFiles(inDir, files)
	s.Files = reorderFiles(files)

	logger.Debug("finalized list of files to process", slog.Any("files", s.Files))

	// Update the state with various metadata
	if err := extractExtras(&s, logger); err != nil {
		return err
	}

	logger.Info("extracted metadata and file relations")

	// Update the state with file listings, like backlinks and similar entries
	if err := makeFileListing(&s, logger); err != nil {
		return err
	}

	logger.Info("created file index")

	// NOTE: The next three processes can run concurrently as they are
	// independent from each other

	// Construct and render atom feeds
	feedGroup, _ := errgroup.WithContext(context.Background())
	feedGroup.Go(func() error {
		atom, err := feed.Construct(&s, logger)
		if err != nil {
			return err
		}

		return feed.Write(&s, atom)
	},
	)

	logger.Info("created atom feed")

	// Copy asset dirs/files over to output directory
	assetsMoveGroup, _ := errgroup.WithContext(context.Background())
	assetsMoveGroup.Go(func() error {
		if err := syncAssets(context.Background(), &s, logger); err != nil {
			return err
		}

		return nil
	},
	)

	logger.Info("synced user assets")

	// Copy static content to the output directory
	staticMoveGroup, _ := errgroup.WithContext(context.Background())
	staticMoveGroup.Go(func() error {
		if err := copyStatic(&s, logger); err != nil {
			return err
		}

		return nil
	},
	)

	logger.Info("copied static files")

	// Create beautiful HTML
	renderGroup, _ := errgroup.WithContext(context.Background())
	renderGroup.Go(func() error {
		return render.Render(context.Background(), &s, logger)
	})

	logger.Info("templated all source files")

	// Wait for all goroutines to finish
	if err := feedGroup.Wait(); err != nil {
		return err
	}

	if err := assetsMoveGroup.Wait(); err != nil {
		return err
	}

	if err := staticMoveGroup.Wait(); err != nil {
		return err
	}

	if err := renderGroup.Wait(); err != nil {
		return err
	}

	end := time.Since(start)

	logger.Info(
		"completed",
		slog.Duration("execution time", end),
		slog.Int("number of files", len(files)),
	)

	return nil
}
