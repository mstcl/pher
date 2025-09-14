// Package cli [TODO]
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mstcl/pher/v2/internal/config"
	"github.com/mstcl/pher/v2/internal/state"
)

const (
	relTemplateDir     = "web/template"
	relStaticDir       = "web/static"
	relStaticOutputDir = "static"
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
	sanitize(&s, logger)

	// create output directory
	if err := createDir(s.OutputDir); err != nil {
		return err
	}
	logger.Debug("created output directory", slog.String("dir", s.OutputDir))

	// parse configuration
	s.Config, err = config.Read(s.ConfigFile)
	if err != nil {
		return err
	}
	logger.Debug("parsed configuration", slog.Any("config", s.Config))

	// clean output directory
	if !s.DryRun {
		exceptions := []string{relStaticOutputDir}

		if err := cleanOutputDir(s.OutputDir, exceptions); err != nil {
			return err
		}
		logger.Info("cleaned output directory", slog.Any("exceptions", exceptions))
	} else {
		logger.Debug("dry run — skipped cleaning output directory")
	}

	// initiate templates
	initTemplates(&s)
	logger.Debug("loaded and initialized templates")

	// get source files from input directory
	s.NodePaths, err = getNodePaths(s.InputDir, logger)
	if err != nil {
		return err
	}
	logger.Debug("found source files", slog.Any("paths", s.NodePaths))

	// TODO: refactor
	// update the state with various metadata
	if err := extractExtras(&s, logger); err != nil {
		return err
	}
	logger.Info("extracted metadata and file relations")

	// TODO: refactor
	// update the state with file listings, like backlinks and similar entries
	if err := populateNodePathLinks(&s, logger); err != nil {
		return err
	}
	logger.Info("created file index")

	// do the rest of our tasks concurrently
	if err := runConcurrentJobs(context.Background(), &s, logger); err != nil {
		return err
	}
	end := time.Since(start)
	logger.Info(
		"completed",
		slog.Duration("execution time", end),
		slog.Int("number of files", len(s.NodePaths)),
	)

	return nil
}
