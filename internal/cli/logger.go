package cli

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

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
