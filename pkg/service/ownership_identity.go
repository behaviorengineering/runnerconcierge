package service

import (
	"path/filepath"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/redact"
)

// Role classifies a discovered service unit.
const (
	RoleSupervisor = "supervisor"
	RoleHelper     = "helper"
	RoleFixture    = "fixture"
	RoleUnknown    = "unknown"
)

// MatchReason codes explain why a unit was included in inventory.
const (
	MatchReasonLaunchdLabel       = "launchd_label"
	MatchReasonBrewFormula        = "brew_formula"
	MatchReasonE2EFixture         = "e2e_fixture"
	MatchReasonWindowsServiceName = "windows_service_name"
)

// IsRunnerServiceName reports whether a service label matches runner inventory heuristics.
func IsRunnerServiceName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if strings.Contains(lower, "runnerconcierge-e2e-") {
		return true
	}
	return strings.Contains(lower, "gitlab-runner")
}

// ClassifyRole assigns supervisor, helper, fixture, or unknown.
func ClassifyRole(serviceName, kind, command string) string {
	lower := strings.ToLower(strings.TrimSpace(serviceName))
	if strings.Contains(lower, "runnerconcierge-e2e-") {
		return RoleFixture
	}
	if kind == "brew_services" && serviceName == "gitlab-runner" {
		return RoleSupervisor
	}
	if commandInvokesGitLabRunner(command) {
		return RoleSupervisor
	}
	if IsRunnerServiceName(serviceName) {
		return RoleHelper
	}
	return RoleUnknown
}

func commandInvokesGitLabRunner(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	for _, part := range strings.Fields(command) {
		part = strings.Trim(part, `"'`)
		if filepath.Base(part) == "gitlab-runner" {
			return true
		}
	}
	return false
}

// SanitizeCommand redacts secrets in a command line for display.
func SanitizeCommand(command string) string {
	return redact.String(strings.TrimSpace(command))
}

// HumanMatchReason turns a match code into operator text.
func HumanMatchReason(code string) string {
	switch code {
	case MatchReasonLaunchdLabel:
		return "launchd label contains gitlab-runner"
	case MatchReasonBrewFormula:
		return "Homebrew formula gitlab-runner"
	case MatchReasonE2EFixture:
		return "runnerconcierge e2e fixture label"
	case MatchReasonWindowsServiceName:
		return "Windows service name or path mentions gitlab-runner"
	default:
		if strings.TrimSpace(code) == "" {
			return "-"
		}
		return code
	}
}

// MatchReasonForLaunchd returns the match reason for a launchd label.
func MatchReasonForLaunchd(serviceName string) string {
	if strings.Contains(strings.ToLower(serviceName), "runnerconcierge-e2e-") {
		return MatchReasonE2EFixture
	}
	return MatchReasonLaunchdLabel
}
