package github

import "github.com/behaviorengineering/runnerconcierge/pkg/errdefs"

// Unsupported reports that a GitHub Actions verb is not implemented yet.
func Unsupported(verb string) error {
	return errdefs.New(verb+".github", errdefs.CodeUnsupportedForge,
		"GitHub Actions "+verb+" is not implemented; use "+verb+" gitlab", nil)
}
