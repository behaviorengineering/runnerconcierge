package forge

import "github.com/behaviorengineering/runnerconcierge/pkg/errdefs"

// Shared action verbs. Each forge implements these with its own workflow.
const (
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

// NeedSubcommand is returned when a forge verb is invoked without list|setup.
func NeedSubcommand(verb, forgeName string) error {
	return errdefs.New(verb+"."+forgeName, errdefs.CodeInvalidScope,
		"choose a subcommand: runnerconcierge "+verb+" "+forgeName+" list | runnerconcierge "+verb+" "+forgeName+" setup",
		nil)
}
