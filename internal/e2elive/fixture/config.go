//go:build e2e_live && (darwin || windows)

package fixture

import (
	"fmt"
	"os"
	"path/filepath"
)

const namePrefix = "runnerconcierge-e2e-"

// ServiceNameForID returns an isolated gitlab-runner service unit name.
func ServiceNameForID(id string) string {
	return namePrefix + id
}

// WriteSyntheticTOML creates a minimal [[runners]] config for inventory parse (token is not real).
func WriteSyntheticTOML(dir, id string) (configPath string, err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := namePrefix + id
	body := fmt.Sprintf(`concurrent = 1
check_interval = 0

[[runners]]
  name = "%s"
  url = "https://gitlab.example.invalid/"
  token = "glrt-e2e-not-a-real-token"
  executor = "shell"
`, name)
	configPath = filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		return "", err
	}
	return configPath, nil
}
