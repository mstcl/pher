// Package cli is pher's entrypoint
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mstcl/pher/v3/internal/config"
	"github.com/mstcl/pher/v3/internal/state"
)

var (
	Logger      *slog.Logger
	LogLevelVar *slog.LevelVar
)

const (
	relTemplateDir     = "web/template"
	relStaticDir       = "web/static"
	relStaticOutputDir = "static"
)

func Handler() error {
	var err error

	start := time.Now() // start execution timer

	s := state.Init() // this is our app state

	parseFlags(&s) // parse all our CLI flags here (onto the state)

	if s.Debug {
		LogLevelVar.Set(slog.LevelDebug)
	}

	initRuntimeInfo() // get global runtime info

	Logger.Debug(
		"gathered runtime info",
		slog.String("revision", Revision),
		slog.String("version", Version),
		slog.String("go_version", GoVersion),
		slog.String("git_version", Version),
		slog.Time("build_date", BuildDate),
	)

	Logger.Debug("parsed flags",
		slog.String("inDir", s.InputDir),
		slog.String("outDir", s.OutputDir),
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
	sanitize(&s)

	// create output directory
	if err := createDir(s.OutputDir); err != nil {
		return err
	}
	Logger.Debug("created output directory", slog.String("dir", s.OutputDir))

	// parse configuration
	s.Config, err = config.Read(s.ConfigFile)
	if err != nil {
		return err
	}
	Logger.Debug("parsed configuration", slog.Any("config", s.Config))

	// clean output directory
	if !s.DryRun {
		exceptions := []string{relStaticOutputDir}

		if err := cleanOutputDir(s.OutputDir, exceptions); err != nil {
			return err
		}
		Logger.Info("cleaned output directory", slog.Any("exceptions", exceptions))
	} else {
		Logger.Debug("dry run — skipped cleaning output directory")
	}

	// initiate templates
	initTemplates(&s)
	Logger.Debug("loaded and initialized templates")

	// get source files from input directory
	s.NodePaths, err = getNodePaths(s.InputDir)
	if err != nil {
		return err
	}
	Logger.Debug("found source files", slog.Any("paths", s.NodePaths))

	// TODO: refactor
	// update the state with various metadata
	if err := extractExtras(&s); err != nil {
		return err
	}
	Logger.Info("extracted metadata and file relations")

	// TODO: refactor
	// update the state with file listings, like backlinks and similar entries
	if err := populateNodePathLinks(&s); err != nil {
		return err
	}
	Logger.Info("created file index")

	// do the rest of our tasks concurrently
	if err := runConcurrentJobs(context.Background(), &s); err != nil {
		return err
	}

	end := time.Since(start)
	Logger.Info(
		"completed",
		slog.Duration("execution time", end),
		slog.Int("number of files", len(s.NodePaths)),
	)

	return nil
}
