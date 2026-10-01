package main

import (
	"context"
	"os"
	"runtime/debug"

	"github.com/behaviorengineering/olly/pkg/dump"
	"github.com/behaviorengineering/olly/pkg/olly"
	"github.com/behaviorengineering/runnerconcierge/internal/cli"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
)

var buildVersion string

func main() {
	ctx := context.Background()
	dir, err := config.ConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	shutdown, err := olly.Init(olly.Config{
		Enabled:     true,
		ServiceName: "runnerconcierge",
		Dump: dump.Config{
			Dir:         dir + "/failures",
			MaxAgeHours: 48,
			MaxFiles:    20,
		},
	})
	if err == nil {
		defer shutdown(context.Background())
	}
	cli.SetVersion(version())
	os.Exit(cli.Main(ctx))
}

func version() string {
	if buildVersion != "" && buildVersion != "dev" {
		return buildVersion
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return "dev"
}
