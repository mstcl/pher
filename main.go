// pher is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// pher is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with pher.  If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mstcl/pher/v3/internal/cli"
)

//go:embed web/template/* web/static/*
var fs embed.FS

func main() {
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		TimeFormat: time.Kitchen,
	}))

	cli.EmbedFS = fs

	if err := cli.Handler(); err != nil {
		logger.Error(fmt.Sprintf("%v", err))

		os.Exit(1)
	}
}
