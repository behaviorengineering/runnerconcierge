package cleanup

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

func resolveDockerBin(ctx context.Context, exec gitexec.Exec) (string, error) {
	if exec == nil {
		return "", fmt.Errorf("cleanup: exec is nil")
	}
	if p, err := exec.LookPath("docker"); err == nil && p != "" {
		return p, nil
	}
	if runtime.GOOS == "darwin" {
		for _, p := range []string{"/opt/homebrew/bin/docker", "/usr/local/bin/docker"} {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("docker not found")
}
