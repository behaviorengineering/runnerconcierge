//go:build e2e_live && (darwin || windows)

package fixture

import "os"

func envIs(name, want string) bool {
	return os.Getenv(name) == want
}

func envTruthy(name string) bool {
	v := os.Getenv(name)
	return v == "1" || v == "true" || v == "TRUE"
}

func artifactDir() string {
	return os.Getenv("E2E_LIVE_ARTIFACT_DIR")
}

func allowExistingServices() bool {
	return envTruthy("E2E_LIVE_ALLOW_EXISTING")
}

func allowWindowsPasswordReset() bool {
	return envTruthy("E2E_LIVE_SET_WINDOWS_PASSWORD") || envIs("GITHUB_ACTIONS", "true")
}
