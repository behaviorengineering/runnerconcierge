//go:build e2e_live && (darwin || windows)

package fixture

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/install"
	"github.com/behaviorengineering/runnerconcierge/pkg/inventory"
	"github.com/behaviorengineering/runnerconcierge/pkg/repair"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

// Options controls a fixture lifecycle run.
type Options struct {
	Exec         gitexec.Exec
	ID           string
	AllowInstall bool
	CLIPath      string
}

// State is returned from Run for Dispose and assertions.
type State struct {
	ID                  string
	WorkDir             string
	ConfigPath          string
	RunnerBin           string
	ServiceName         string
	UseBrew             bool
	SystemSeed          bool
	WindowsPassword     string
	InitialServiceCount int
	ConfigSnapshots     configSnapshot
	Before              *inventory.Report
	After               *inventory.Report
	Repair              *repair.Result
}

// Run executes install → seed bad service → inventory → repair → inventory.
func Run(parent context.Context, opts Options) (*State, error) {
	if opts.Exec == nil {
		return nil, fmt.Errorf("fixture: exec is nil")
	}
	ctx, cancel := ensureDeadline(parent, 25*time.Minute)
	defer cancel()

	id := strings.TrimSpace(opts.ID)
	if id == "" {
		id = randomID()
	}
	st := &State{ID: id}
	st.ServiceName = ServiceNameForID(id)

	if err := safetyAbortExisting(ctx, opts.Exec, st.ServiceName); err != nil {
		return nil, err
	}
	initialCount, err := countRunnerServices(ctx, opts.Exec)
	if err != nil {
		return nil, err
	}
	st.InitialServiceCount = initialCount
	snap, err := snapshotCandidateConfigs()
	if err != nil {
		return nil, err
	}
	st.ConfigSnapshots = snap

	inst := install.New(opts.Exec)
	bin, err := inst.Ensure(ctx, install.ToolGitLabRunner, true)
	if err != nil {
		return nil, err
	}
	st.RunnerBin = bin

	if runtime.GOOS == "windows" {
		pw, err := ensureWindowsPassword(ctx, opts.Exec)
		if err != nil {
			return nil, err
		}
		st.WindowsPassword = pw
	}

	base := filepath.Join(os.TempDir(), namePrefix+id)
	st.WorkDir = base
	st.ConfigPath, err = WriteSyntheticTOML(base, id)
	if err != nil {
		return nil, err
	}

	if err := seedBadService(ctx, opts.Exec, st); err != nil {
		if errors.Is(err, ErrSkipped) {
			return nil, err
		}
		return st, err
	}

	login, err := service.LoginUser(opts.Exec)
	if err != nil {
		return st, err
	}

	before, err := inventory.Run(ctx, inventory.Options{
		Exec:             opts.Exec,
		LoginUser:        login,
		ExtraConfigPaths: []string{st.ConfigPath},
	})
	if err != nil {
		return st, err
	}
	st.Before = before
	writeArtifact(before, "before-inventory.json")
	if strings.TrimSpace(opts.CLIPath) != "" && st.InitialServiceCount == 0 {
		if err := assertCLIStatus(ctx, opts.CLIPath, st.ConfigPath, 3); err != nil {
			return st, err
		}
	}
	if !hasExpectedBlockingForFixture(before, st.ConfigPath, st.ServiceName) {
		return st, wrap("Run", errdefs.CodeProcessConflict, "expected blocking smell before repair", nil)
	}

	repairOpts := repair.Options{
		Exec:             opts.Exec,
		ConfigPath:       st.ConfigPath,
		ServiceName:      st.ServiceName,
		AllowDestructive: true,
		MoveConfigToUser: true,
		UseBrewServices:  st.UseBrew,
		WindowsPassword:  st.WindowsPassword,
		LoginUser:        login,
	}
	res, err := repair.Run(ctx, repairOpts)
	if err != nil {
		return st, err
	}
	st.Repair = res

	after, err := inventory.Run(ctx, inventory.Options{
		Exec:             opts.Exec,
		LoginUser:        login,
		ExtraConfigPaths: []string{st.ConfigPath},
	})
	if err != nil {
		return st, err
	}
	st.After = after
	writeArtifact(after, "after-inventory.json")
	if hasBlockingForFixture(after, st.ConfigPath, st.ServiceName) {
		return st, wrap("Run", errdefs.CodeProcessConflict, "blocking findings remain after repair for fixture", nil)
	}

	if strings.TrimSpace(opts.CLIPath) != "" && st.InitialServiceCount == 0 {
		if err := assertCLIStatus(ctx, opts.CLIPath, st.ConfigPath, 0); err != nil {
			return st, err
		}
	}

	return st, nil
}

// Dispose uninstalls the fixture service and verifies linger rules.
func Dispose(parent context.Context, exec gitexec.Exec, st *State) error {
	if st == nil || exec == nil {
		return nil
	}
	ctx, cancel := ensureDeadline(parent, 10*time.Minute)
	defer cancel()

	_ = uninstallFixtureService(ctx, exec, st)
	_ = disposePlatform(ctx, exec, st)

	if st.WorkDir != "" {
		_ = removeRepairBackups(st.ConfigPath)
		_ = os.RemoveAll(st.WorkDir)
	}

	if err := assertLingerServices(ctx, exec, st.InitialServiceCount, st.ServiceName); err != nil {
		return err
	}
	return verifyConfigSnapshots(st.ConfigSnapshots)
}

func randomID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func ensureDeadline(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

func writeArtifact(rep *inventory.Report, name string) {
	dir := artifactDir()
	if dir == "" || rep == nil {
		return
	}
	_ = os.MkdirAll(dir, 0o700)
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return
	}
	defer f.Close()
	_ = inventory.RenderJSON(f, rep)
}

func removeRepairBackups(configPath string) error {
	dir := filepath.Dir(configPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	prefix := filepath.Base(configPath) + ".repair-bak-"
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	return nil
}
