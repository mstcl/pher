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

	// Read configuration
	s.Config, err = config.Read(s.ConfigFile)
	if err != nil {
		return err
	}

	logger.Debug("read configuration", slog.Any("config", s.Config))

	// Clean output directory
	if !s.DryRun {
		logger.Info("cleaning output directory")

		// TODO: update this so it ignores staticOutputDir
		files, err := filepath.Glob(filepath.Join(s.OutDir, "/*"))
		if err != nil {
			return fmt.Errorf("filepath.Glob: %w", err)
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

	// Glob for markdown files and reorder so indexes are processed last
	files, err := zglob.Glob(s.InDir + "/**/*.md")
	if err != nil {
		return fmt.Errorf("glob files: %w", err)
	}

	files = filterHiddenFiles(s.InDir, files)
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

func sanitize(s *state.State, logger *slog.Logger) error {
	var err error

	// Sanitize configuration file
	s.ConfigFile, err = filepath.Abs(s.ConfigFile)
	if err != nil {
		return fmt.Errorf("absolute path: %w", err)
	}

	// Check whether configuration file exists
	_, err = os.Stat(s.ConfigFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("missing: %s", s.ConfigFile)
	} else if err != nil {
		return fmt.Errorf("os.Stat %s: %w", s.ConfigFile, err)
	}

	logger.Debug("sanitized config file", slog.String("path", s.ConfigFile))

	// Sanitize input directory
	s.InDir, err = filepath.Abs(s.InDir)
	if err != nil {
		return fmt.Errorf("filepath.Abs: %w", err)
	}

	logger.Debug("sanitized input directory", slog.String("path", s.InDir))

	// Sanitize output directory
	s.OutDir, err = filepath.Abs(s.OutDir)
	if err != nil {
		return fmt.Errorf("filepath.Abs: %w", err)
	}

	logger.Debug("sanitized output directory", slog.String("path", s.OutDir))

	return nil
}
