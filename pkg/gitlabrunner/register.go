package gitlabrunner

import (
	"fmt"
	"runtime"
	"strings"
)

// RegisterArgs builds gitlab-runner register argv for GitLab 16+ auth tokens.
// Server-side tags and run_untagged MUST be set via POST /user/runners only.
type RegisterArgs struct {
	URL              string
	Token            string
	Name             string
	Executor         string
	Shell            string
	ConfigPath       string
	WorkingDirectory string
	DockerImage      string
}

// BuildRegisterArgv returns non-interactive register arguments.
func BuildRegisterArgv(in RegisterArgs) ([]string, error) {
	url := strings.TrimSpace(in.URL)
	if url == "" {
		return nil, fmt.Errorf("gitlabrunner: url is required")
	}
	if strings.TrimSpace(in.Token) == "" {
		return nil, fmt.Errorf("gitlabrunner: token is required")
	}
	executor := strings.TrimSpace(in.Executor)
	if executor == "" {
		executor = "shell"
	}

	args := []string{
		"register",
		"--non-interactive",
		"--url", url,
		"--token", in.Token,
		"--name", in.Name,
		"--executor", executor,
	}
	if in.ConfigPath != "" {
		args = append(args, "--config", in.ConfigPath)
	}
	if in.WorkingDirectory != "" {
		args = append(args, "--working-directory", in.WorkingDirectory)
	}
	switch executor {
	case "shell":
		shell := in.Shell
		if shell == "" {
			shell = defaultShell()
		}
		args = append(args, "--shell", shell)
	case "docker":
		if strings.TrimSpace(in.DockerImage) != "" {
			args = append(args, "--docker-image", in.DockerImage)
		}
	}
	return args, nil
}

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "pwsh"
	}
	return "bash"
}
