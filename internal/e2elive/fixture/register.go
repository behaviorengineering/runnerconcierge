//go:build e2e_live && (darwin || windows)

package fixture

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/internal/config"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/gitlabrunner"
	"github.com/behaviorengineering/runnerconcierge/pkg/install"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

// RegisterState is returned from RunRegister for DisposeRegister.
type RegisterState struct {
	*State
	GitLabRunnerID int
	PAT            string
	GitLabURL      string
}

// WantRegister reports whether the GitLab create/register/unregister path should run.
func WantRegister() bool {
	return envTruthy("E2E_LIVE_REGISTER") || strings.TrimSpace(os.Getenv("E2E_LIVE_REPO")) != ""
}

// SkipRegisterReason is empty when register e2e can run.
func SkipRegisterReason() string {
	if !WantRegister() {
		return "set E2E_LIVE_REPO=group/project (once) to include GitLab create/register in make e2e-live"
	}
	if strings.TrimSpace(os.Getenv("E2E_LIVE_REPO")) == "" {
		return "E2E_LIVE_REGISTER=1 requires E2E_LIVE_REPO=group/project"
	}
	return ""
}

// RunRegister creates a GitLab project runner, registers it, installs an isolated service, and waits online.
func RunRegister(parent context.Context, opts Options) (*RegisterState, error) {
	if opts.Exec == nil {
		return nil, wrap("RunRegister", errdefs.CodeCreateFailed, "exec is nil", nil)
	}
	ctx, cancel := ensureDeadline(parent, 25*time.Minute)
	defer cancel()

	id := strings.TrimSpace(opts.ID)
	if id == "" {
		id = randomID()
	}
	st := &State{ID: id, ServiceName: ServiceNameForID(id)}
	rs := &RegisterState{State: st}

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
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, wrap("RunRegister", errdefs.CodeCreateFailed, "mkdir workdir", err)
	}
	st.WorkDir = base
	st.ConfigPath = filepath.Join(base, "config.toml")

	cfg, err := config.Load("")
	if err != nil {
		cfg = &config.UserConfig{GitLabURL: "https://gitlab.com", DefaultExecutor: "shell"}
	}
	gitlabURL := strings.TrimSpace(os.Getenv("E2E_LIVE_GITLAB_URL"))
	if gitlabURL == "" {
		gitlabURL = cfg.GitLabURL
	}
	if gitlabURL == "" {
		gitlabURL = "https://gitlab.com"
	}
	rs.GitLabURL = gitlabURL

	pat := strings.TrimSpace(os.Getenv("GITLAB_TOKEN"))
	if pat == "" {
		if resolved, rerr := config.ResolvePAT(); rerr == nil {
			pat = strings.TrimSpace(resolved)
		}
	}
	rs.PAT = pat

	repo := strings.TrimSpace(os.Getenv("E2E_LIVE_REPO"))
	client := gitlabrunner.NewClient(gitlabURL, opts.Exec, nil)
	projectID, err := client.ResolveProjectID(ctx, repo)
	if err != nil {
		return rs, err
	}
	runnerID, token, err := client.CreateRunner(ctx, gitlabrunner.CreateRunnerRequest{
		RunnerType:  "project_type",
		ProjectID:   projectID,
		Description: st.ServiceName,
		TagList:     []string{"runnerconcierge-e2e", id},
		RunUntagged: false,
		Locked:      true,
	}, pat)
	if err != nil {
		return rs, err
	}
	rs.GitLabRunnerID = runnerID

	regArgs, err := gitlabrunner.BuildRegisterArgv(gitlabrunner.RegisterArgs{
		URL:              gitlabURL,
		Token:            token,
		Name:             st.ServiceName,
		Executor:         "shell",
		ConfigPath:       st.ConfigPath,
		WorkingDirectory: st.WorkDir,
	})
	if err != nil {
		return rs, wrap("RunRegister", errdefs.CodeRegisterFailed, "build register argv", err)
	}
	if err := client.Register(ctx, bin, regArgs); err != nil {
		return rs, err
	}

	login, err := service.LoginUser(opts.Exec)
	if err != nil {
		return rs, err
	}
	svc := service.New(opts.Exec)
	if err := svc.Install(ctx, service.InstallOpts{
		BinaryPath:       bin,
		ConfigPath:       st.ConfigPath,
		WorkingDirectory: st.WorkDir,
		ServiceName:      st.ServiceName,
		User:             login,
		WindowsUser:      login,
		WindowsPassword:  st.WindowsPassword,
		UseBrewServices:  false,
	}); err != nil {
		return rs, err
	}
	if err := svc.Start(ctx, service.StartOpts{ServiceName: st.ServiceName}); err != nil {
		return rs, wrap("RunRegister", errdefs.CodeServiceStart, "start isolated service", err)
	}
	if err := client.WaitOnline(ctx, runnerID, 2*time.Minute); err != nil {
		return rs, err
	}
	return rs, nil
}

// DisposeRegister uninstalls the isolated service, deletes the GitLab runner, and checks linger.
func DisposeRegister(parent context.Context, exec gitexec.Exec, rs *RegisterState) error {
	if rs == nil || rs.State == nil || exec == nil {
		return nil
	}
	ctx, cancel := ensureDeadline(parent, 10*time.Minute)
	defer cancel()
	st := rs.State
	_ = uninstallFixtureService(ctx, exec, st)
	_ = disposePlatform(ctx, exec, st)
	if rs.GitLabRunnerID > 0 {
		client := gitlabrunner.NewClient(rs.GitLabURL, exec, nil)
		_ = client.DeleteRunner(ctx, rs.GitLabRunnerID, rs.PAT)
	}
	if st.WorkDir != "" {
		_ = os.RemoveAll(st.WorkDir)
	}
	if err := assertLingerServices(ctx, exec, st.InitialServiceCount, st.ServiceName); err != nil {
		return err
	}
	return verifyConfigSnapshots(st.ConfigSnapshots)
}
