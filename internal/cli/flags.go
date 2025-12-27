package cli

import (
	"flag"

	"github.com/mstcl/pher/v2/internal/state"
)

func parseFlags(s *state.State) {
	flag.BoolVar(&s.ShowVersion, "v", false, "Show version and exit")
	flag.BoolVar(&s.DryRun, "d", false, "Don't render (dry run)")
	flag.BoolVar(&s.Debug, "debug", false, "Verbose (debug) mode")

	flag.StringVar(&s.ConfigFile, "c", "config.yaml", "Path to config file")
	flag.StringVar(&s.InputDir, "i", ".", "Input directory")
	flag.StringVar(&s.OutputDir, "o", "_site", "Output directory")

	flag.Parse()
}
