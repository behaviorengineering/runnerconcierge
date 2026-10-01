package detect

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

// Issue is a doctor finding.
type Issue struct {
	Code    errdefs.Code
	Message string
	Block   bool
}

// Report is the doctor output.
type Report struct {
	GOOS         string
	GOARCH       string
	Username     string
	Elevated     bool
	GitOK        bool
	RunnerPath   string
	RunnerVer    string
	GlabPath     string
	GlabVer      string
	BrewOK       bool
	WingetOK     bool
	DockerOK     bool
	RunnerProcs  int
	ConfigPaths  []string
	Issues       []Issue
}

// Doctor runs preflight checks.
type Doctor struct {
	Exec gitexec.Exec
}

// NewDoctor returns a doctor with required exec.
func NewDoctor(exec gitexec.Exec) *Doctor {
	if exec == nil {
		panic("detect: exec is nil")
	}
	return &Doctor{Exec: exec}
}

// Run collects a report. requireDocker gates docker checks.
func (d *Doctor) Run(ctx context.Context, requireDocker bool) (*Report, error) {
	if d == nil {
		return nil, errdefs.New("detect.Run", errdefs.CodeMissingDeadline, "doctor is nil", nil)
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, errdefs.New("detect.Run", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}

	r := &Report{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}
	if u, err := os.UserHomeDir(); err == nil {
		_ = u
	}
	if cu, err := userName(); err == nil {
		r.Username = cu
	}
	r.Elevated = isElevated()
	r.ConfigPaths = candidateConfigPaths()

	switch runtime.GOOS {
	case "darwin", "windows":
	default:
		r.Issues = append(r.Issues, Issue{
			Code:    errdefs.CodeUnsupportedOS,
			Message: "runnerconcierge supports macOS and Windows only; use native packages on Linux",
			Block:   true,
		})
	}

	if _, err := d.Exec.LookPath("git"); err != nil {
		r.Issues = append(r.Issues, Issue{
			Code:    errdefs.CodeGitMissing,
			Message: "git is not on PATH",
			Block:   true,
		})
	} else {
		r.GitOK = true
	}

	if path, err := d.Exec.LookPath("gitlab-runner"); err == nil {
		r.RunnerPath = path
		if ver, err := d.toolVersion(ctx, path); err == nil {
			r.RunnerVer = ver
		}
	}
	if path, err := d.Exec.LookPath("glab"); err == nil {
		r.GlabPath = path
		if ver, err := d.toolVersion(ctx, "glab"); err == nil {
			r.GlabVer = ver
		}
	}
	if _, err := d.Exec.LookPath("brew"); err == nil {
		r.BrewOK = true
	}
	if _, err := d.Exec.LookPath("winget"); err == nil {
		r.WingetOK = true
	}

	r.RunnerProcs = countRunnerProcs(ctx, d.Exec)

	if requireDocker {
		if err := dockerOK(ctx, d.Exec); err != nil {
			r.Issues = append(r.Issues, Issue{
				Code:    errdefs.CodeRunnerOffline,
				Message: "docker is not reachable: " + err.Error(),
				Block:   false,
			})
		} else {
			r.DockerOK = true
		}
	}

	if r.RunnerProcs > 1 {
		r.Issues = append(r.Issues, Issue{
			Code:    errdefs.CodeProcessConflict,
			Message: "multiple gitlab-runner manager processes detected; stop extras before setup",
			Block:   true,
		})
	}

	if runtime.GOOS == "windows" && !r.Elevated {
		r.Issues = append(r.Issues, Issue{
			Code:    errdefs.CodeElevationRequired,
			Message: "not elevated; Windows service install will require an Administrator session",
			Block:   false,
		})
	}

	return r, nil
}

func (d *Doctor) toolVersion(ctx context.Context, name string) (string, error) {
	out, err := d.Exec.Run(ctx, name, "--version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func dockerOK(ctx context.Context, exec gitexec.Exec) error {
	_, err := exec.Run(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	return err
}

func countRunnerProcs(ctx context.Context, exec gitexec.Exec) int {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Run(ctx, "pgrep", "-lf", "gitlab-runner run")
		if err != nil {
			return 0
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) == 1 && lines[0] == "" {
			return 0
		}
		return len(lines)
	case "windows":
		out, err := exec.Run(ctx, "powershell", "-NoProfile", "-Command", "(Get-Process -Name gitlab-runner -ErrorAction SilentlyContinue).Count")
		if err != nil {
			return 0
		}
		n := strings.TrimSpace(string(out))
		if n == "" {
			return 0
		}
		var c int
		_, _ = parseInt(n, &c)
		return c
	default:
		return 0
	}
}

func parseInt(s string, out *int) (int, error) {
	var n int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + int(ch-'0')
	}
	*out = n
	return n, nil
}

func userName() (string, error) {
	if u := os.Getenv("USERNAME"); u != "" {
		return u, nil
	}
	out, err := exec.Command("whoami").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func candidateConfigPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			"C:\\GitLab-Runner\\config.toml",
			os.ExpandEnv("${LOCALAPPDATA}\\GitLab-Runner\\config.toml"),
		}
	default:
		home, _ := os.UserHomeDir()
		return []string{
			home + "/.gitlab-runner/config.toml",
			"/etc/gitlab-runner/config.toml",
		}
	}
}

func isElevated() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		"([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "True"
}

// Fingerprint hashes host identity for checkpoint resume.
func Fingerprint() string {
	host, _ := os.Hostname()
	user, _ := userName()
	raw := runtime.GOOS + "|" + runtime.GOARCH + "|" + host + "|" + user
	return hashString(raw)
}

func hashString(s string) string {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return formatHex(h)
}

func formatHex(h uint64) string {
	const hexdigits = "0123456789abcdef"
	var b [16]byte
	for i := 7; i >= 0; i-- {
		b[i*2] = hexdigits[h&0xf]
		b[i*2+1] = hexdigits[(h>>4)&0xf]
		h >>= 8
	}
	return string(b[:])
}

// WithDoctorDeadline wraps ctx with a default doctor timeout.
func WithDoctorDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 10*time.Second)
}
