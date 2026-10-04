package cli

import (
	"context"
	"flag"
	"io"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/inventory"
)

func runStatus(ctx context.Context, args []string, w io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "JSON output")
	runnerCfg := fs.String("runner-config", "", "extra runner config.toml path (repeatable via multiple flags in cobra)")
	_ = fs.Parse(args)
	extra := []string{}
	if strings.TrimSpace(*runnerCfg) != "" {
		extra = append(extra, strings.TrimSpace(*runnerCfg))
	}
	for _, a := range fs.Args() {
		if strings.TrimSpace(a) != "" {
			extra = append(extra, strings.TrimSpace(a))
		}
	}
	ctx, cancel := withDeadline(ctx, 2*time.Minute)
	defer cancel()
	rep, err := inventory.Run(ctx, inventory.Options{
		Exec:             gitexec.New(),
		ExtraConfigPaths: extra,
	})
	if err != nil {
		writef(w, "status: %v\n", err)
		return ExitFail
	}
	if *jsonOut {
		if err := inventory.RenderJSON(w, rep); err != nil {
			writef(w, "status: %v\n", err)
			return ExitFail
		}
	} else {
		if err := inventory.Render(w, rep); err != nil {
			writef(w, "status: %v\n", err)
			return ExitFail
		}
	}
	if inventory.HasBlocking(rep) {
		return ExitDoctor
	}
	return ExitOK
}
