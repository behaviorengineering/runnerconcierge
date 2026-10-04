package cli

import (
	"context"
	"flag"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/prompt"
	"github.com/behaviorengineering/runnerconcierge/pkg/repair"
)

func runRepairService(ctx context.Context, args []string, w io.Writer) int {
	fs := flag.NewFlagSet("repair-service", flag.ContinueOnError)
	cfg := fs.String("runner-config", "", "runner config.toml path")
	service := fs.String("service", "", "service name when multiple")
	yes := fs.Bool("yes", false, "allow destructive repair without interactive confirm")
	winPW := fs.String("windows-password", "", "Windows service account password")
	useBrew := fs.Bool("brew", false, "use brew services on macOS")
	_ = fs.Parse(args)
	if strings.TrimSpace(*cfg) == "" {
		writef(w, "repair-service: --runner-config is required\n")
		return ExitUsage
	}
	pw := strings.TrimSpace(*winPW)
	if pw == "" {
		pw = strings.TrimSpace(os.Getenv("RUNNERCONCIERGE_WINDOWS_PASSWORD"))
	}
	var pr prompt.Prompter
	if !*yes {
		pr = prompt.ForTTY()
		if pw == "" && runtime.GOOS == "windows" {
			if p, err := pr.Password(ctx, "Windows service account password (login user)"); err == nil {
				pw = p
			}
		}
	} else if runtime.GOOS == "windows" && pw == "" {
		writef(w, "repair-service: Windows requires --windows-password or RUNNERCONCIERGE_WINDOWS_PASSWORD with --yes\n")
		return ExitFail
	}
	ctx, cancel := withDeadline(ctx, 30*time.Minute)
	defer cancel()
	res, err := repair.Run(ctx, repair.Options{
		Exec:             gitexec.New(),
		Prompter:         pr,
		ConfigPath:       strings.TrimSpace(*cfg),
		ServiceName:      strings.TrimSpace(*service),
		WindowsPassword:  pw,
		AllowDestructive: *yes,
		MoveConfigToUser: true,
		UseBrewServices:  *useBrew,
	})
	if err != nil {
		writef(w, "repair-service: %v\n", err)
		if res != nil {
			repair.PrintResult(w, res)
		}
		return ExitFail
	}
	repair.PrintResult(w, res)
	return ExitOK
}
