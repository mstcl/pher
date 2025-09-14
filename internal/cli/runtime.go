package cli

import (
	"runtime"
	"runtime/debug"
	"time"
)

var (
	Version    = "unknown"
	GoVersion  = runtime.Version()
	Revision   = "unknown"
	BuildDate  time.Time
	DirtyBuild = true
)

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
