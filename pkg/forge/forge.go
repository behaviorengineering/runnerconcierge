package forge

import "github.com/behaviorengineering/runnerconcierge/pkg/errdefs"

// Shared action verbs. Each forge implements these with its own workflow.
const (
	VerbSetup   = "setup"
	VerbDoctor  = "doctor"
	VerbVerify  = "verify"
	VerbStatus  = "status"
	VerbRepair  = "repair-service"
	VerbRunners = "runners"
)

// Forge names used as CLI subcommands under each verb.
const (
	GitLab = "gitlab"
	GitHub = "github"
)

// NeedForge is returned when a verb is invoked without a forge path.
func NeedForge(verb string) error {
	return errdefs.New(verb, errdefs.CodeInvalidScope,
		"choose a forge: runnerconcierge "+verb+" gitlab | runnerconcierge "+verb+" github", nil)
}
