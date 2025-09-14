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
	"runtime"
	"runtime/debug"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-zglob"
	"github.com/mstcl/pher/v2/internal/config"
	"github.com/mstcl/pher/v2/internal/feed"
	"github.com/mstcl/pher/v2/internal/render"
	"github.com/mstcl/pher/v2/internal/state"
	"golang.org/x/sync/errgroup"
)

const (
	relTemplateDir     = "web/template"
	relStaticDir       = "web/static"
	relStaticOutputDir = "static"
)

var EmbedFS embed.FS

var (
	Version    = "unknown"
	GoVersion  = runtime.Version()
	Revision   = "unknown"
	BuildDate  time.Time
	DirtyBuild = true
)

func Handler() error {
	var err error

	start := time.Now() // start execution timer

	s := state.Init() // this is our lobal state

	parseFlags(&s) // parse all our CLI flags here (onto the state)

	logger := createLogger(s.Debug) // create our "global" logger

	initRuntimeInfo() // get global runtime info

	logger.Debug(
		"gathered runtime info",
		slog.String("revision", Revision),
		slog.String("version", Version),
		slog.String("go_version", GoVersion),
		slog.String("git_version", Version),
		slog.Time("build_date", BuildDate),
	)

	logger.Debug("parsed flags",
		slog.String("inDir", s.InDir),
		slog.String("outDir", s.OutDir),
		slog.String("configFile", s.ConfigFile),
		slog.Bool("version", s.ShowVersion),
		slog.Bool("dryRun", s.DryRun),
		slog.Bool("debug", s.Debug),
	)

	// show version and exit if that's the case
	if s.ShowVersion {
		fmt.Printf("pher %v\n", Version)

		return nil
	}

	// sanitize paths
	sanitize(&s, logger)

	// create output directory
	if err := os.MkdirAll(s.OutDir, 0o755); err != nil {
		return fmt.Errorf("os.MkdirAll %s: %w", s.OutDir, err)
	}

	logger.Debug("created output directory", slog.String("dir", s.OutDir))

	// parse configuration
	s.Config, err = config.Read(s.ConfigFile)
	if err != nil {
		return err
	}

	logger.Debug("read configuration", slog.Any("config", s.Config))

	// clean output directory
	if !s.DryRun {
		exceptions := []string{relStaticOutputDir}

		if err := cleanOutputDir(s.OutDir, exceptions); err != nil {
			return fmt.Errorf("clean output directory: %w", err)
		}

		logger.Info("cleaned output directory", slog.Any("exceptions", exceptions))
	} else {
		logger.Debug("dry run — skipped cleaning output directory")
	}

	// initiate templates
	funcMap := template.FuncMap{
		"joinPath": path.Join,
	} // TODO: split this out to separate package

	tmpl := template.New("main")
	tmpl = tmpl.Funcs(funcMap)
	s.Templates = template.Must(tmpl.ParseFS(EmbedFS, filepath.Join(relTemplateDir, "*")))

	logger.Debug("loaded and initialized templates")

	// find the files we need to process by recursively glob for all markdown files
	srcFiles, err := zglob.Glob(filepath.Join(s.InDir, "**", "*.md"))
	if err != nil {
		return fmt.Errorf("glob files: %w", err)
	}

	// sanitize by removing all hidden files
	srcFiles = dropHiddenFiles(srcFiles)

	// reorder the list so indexes are processed last
	s.Files = reorderFiles(srcFiles)

	logger.Debug("finalized list of files to process", slog.Any("files", s.Files))

	// update the state with various metadata
	if err := extractExtras(&s, logger); err != nil {
		return err
	}

	logger.Info("extracted metadata and file relations")

	// update the state with file listings, like backlinks and similar entries
	if err := makeFileListing(&s, logger); err != nil {
		return err
	}

	logger.Info("created file index")

	// NOTE: The next three processes can run concurrently as they are
	// independent from each other

	// construct and render atom feeds
	constructFeedGroup, _ := errgroup.WithContext(context.Background())
	constructFeedGroup.Go(func() error {
		atom, err := feed.Construct(&s, logger)
		if err != nil {
			return err
		}

		return feed.Write(&s, atom)
	},
	)

	logger.Info("created atom feed")

	// copy asset dirs/files over to output directory
	copyUserAssetsGroup, _ := errgroup.WithContext(context.Background())
	copyUserAssetsGroup.Go(func() error {
		if err := copyUserAssets(context.Background(), &s, logger); err != nil {
			return err
		}

		return nil
	},
	)

	logger.Info("synced user assets")

	// copy static content to the output directory
	copyStaticGroup, _ := errgroup.WithContext(context.Background())
	copyStaticGroup.Go(func() error {
		if err := copyStatic(&s, logger); err != nil {
			return err
		}

		return nil
	},
	)

	logger.Info("copied static files")

	// render all markdown files
	renderGroup, _ := errgroup.WithContext(context.Background())
	renderGroup.Go(func() error {
		return render.Render(context.Background(), &s, logger)
	})

	logger.Info("templated all source files")

	// wait for all goroutines to finish
	if err := constructFeedGroup.Wait(); err != nil {
		return err
	}

	if err := copyUserAssetsGroup.Wait(); err != nil {
		return err
	}

	if err := copyStaticGroup.Wait(); err != nil {
		return err
	}

	if err := renderGroup.Wait(); err != nil {
		return err
	}

	end := time.Since(start)

	logger.Info(
		"completed",
		slog.Duration("execution time", end),
		slog.Int("number of files", len(srcFiles)),
	)

	return nil
}

// initRuntimeInfo grabs package info
// stolen from https://www.piotrbelina.com/blog/go-build-info-debug-readbuildinfo-ldflags/
func initRuntimeInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	if info.Main.Version != "" {
		Version = info.Main.Version
	}

	for _, kv := range info.Settings {
		if kv.Value == "" {
			continue
		}
		switch kv.Key {
		case "vcs.revision":
			Revision = kv.Value
		case "vcs.time":
			BuildDate, _ = time.Parse(time.RFC3339, kv.Value)
		case "vcs.modified":
			DirtyBuild = kv.Value == "true"
		}
	}
}

func parseFlags(s *state.State) {
	flag.BoolVar(&s.ShowVersion, "v", false, "Show version and exit")
	flag.BoolVar(&s.DryRun, "d", false, "Don't render (dry run)")
	flag.BoolVar(&s.Debug, "debug", false, "Verbose (debug) mode")

	flag.StringVar(&s.ConfigFile, "c", "config.yaml", "Path to config file")
	flag.StringVar(&s.InDir, "i", ".", "Input directory")
	flag.StringVar(&s.OutDir, "o", "_site", "Output directory")

	flag.Parse()
}

func createLogger(debug bool) *slog.Logger {
	var lvl slog.Level

	if debug {
		lvl = slog.LevelDebug
	} else {
		lvl = slog.LevelInfo
	}

	return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      lvl,
		TimeFormat: time.Kitchen,
	}))
}
