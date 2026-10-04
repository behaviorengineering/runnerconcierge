package gitlabrunner

import (
	"fmt"
	"strings"
)

// UnregisterArgs builds gitlab-runner unregister argv.
type UnregisterArgs struct {
	Name       string
	URL        string
	ConfigPath string
}

// BuildUnregisterArgv returns argv for gitlab-runner unregister.
// Do not add --non-interactive; the unregister subcommand does not accept it (19.x).
func BuildUnregisterArgv(in UnregisterArgs) ([]string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("gitlabrunner: runner name is required for unregister")
	}
	args := []string{
		"unregister",
		"--name", name,
	}
	if strings.TrimSpace(in.ConfigPath) != "" {
		args = append(args, "--config", strings.TrimSpace(in.ConfigPath))
	}
	if strings.TrimSpace(in.URL) != "" {
		args = append(args, "--url", strings.TrimSpace(in.URL))
	}
	return args, nil
}
