package inventory

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

type runnersFile struct {
	Runners []runnerSection `toml:"runners"`
}

type runnerSection struct {
	Name     string `toml:"name"`
	URL      string `toml:"url"`
	Executor string `toml:"executor"`
	Token    string `toml:"token"`
}

func parseConfigFile(path string) ([]RunnerEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rf runnersFile
	if err := toml.Unmarshal(data, &rf); err != nil {
		return nil, err
	}
	out := make([]RunnerEntry, 0, len(rf.Runners))
	for _, r := range rf.Runners {
		out = append(out, RunnerEntry{
			Name:     r.Name,
			URL:      r.URL,
			Executor: r.Executor,
		})
	}
	return out, nil
}
