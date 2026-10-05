package inventory

import (
	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/service"
)

// Options controls inventory collection.
type Options struct {
	Exec             gitexec.Exec
	ExtraConfigPaths []string
	LoginUser        string
}

// Report is the full machine inventory.
type Report struct {
	GOOS         string
	GOARCH       string
	LoginUser    string
	Elevated     bool
	RunnerBinary string
	RunnerVer    string
	Configs      []ConfigReport
	Services     []service.Ownership
	Findings     []Finding
	NextActions  []string
}

// ConfigReport describes one config.toml file.
type ConfigReport struct {
	Path       string
	Readable   bool
	SystemPath bool
	Runners    []RunnerEntry
}

// RunnerEntry is one [[runners]] section (no token).
type RunnerEntry struct {
	Name     string
	URL      string
	Executor string
	GitLabID int
}

// Finding is a typed smell.
type Finding struct {
	Code       errdefs.Code
	Block      bool
	Message    string
	ConfigPath string
	Service    string
	Repairable bool
}
