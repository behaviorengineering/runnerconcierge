package preset

// Preset supplies host-specific defaults for the setup wizard.
type Preset struct {
	GitLabURL     string
	RepoPath      string
	RunnerType    string
	GroupID       int
	Executor      string
	TagList       []string
	RunUntagged   bool
	Description   string
	DockerImage   string
	SkipQuestions map[string]bool
	HostEpilogue  string
}

// Empty reports whether no preset fields are set.
func (p Preset) Empty() bool {
	return p.GitLabURL == "" && p.RepoPath == "" && p.Executor == "" && len(p.TagList) == 0
}
