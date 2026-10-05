package gitlabrunner

import (
	"encoding/json"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

const createRunnerOp = "gitlabrunner.CreateRunner"

func newCreateErr(msg string, err error) *errdefs.Error {
	return errdefs.New(createRunnerOp, errdefs.CodeCreateFailed, msg, err)
}

func newInvalidScopeErr(msg string) *errdefs.Error {
	return errdefs.New(createRunnerOp, errdefs.CodeInvalidScope, msg, nil)
}

func gitlabMessageFromBody(body []byte) string {
	var o struct {
		Message interface{} `json:"message"`
		Error   string      `json:"error"`
	}
	if err := json.Unmarshal(body, &o); err != nil {
		return ""
	}
	switch m := o.Message.(type) {
	case string:
		if strings.TrimSpace(m) != "" {
			return strings.TrimSpace(m)
		}
	case []interface{}:
		var parts []string
		for _, item := range m {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				parts = append(parts, strings.TrimSpace(s))
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "; ")
		}
	}
	if strings.TrimSpace(o.Error) != "" {
		return strings.TrimSpace(o.Error)
	}
	return ""
}

func glabFailureMessage(err error) string {
	if err == nil {
		return "GitLab API request failed"
	}
	s := err.Error()
	if idx := strings.Index(s, "{"); idx >= 0 {
		if msg := gitlabMessageFromBody([]byte(s[idx:])); msg != "" {
			return msg
		}
	}
	if strings.Contains(s, "HTTP 400") {
		return "GitLab rejected the runner create request"
	}
	if strings.Contains(strings.ToLower(s), "project not found") {
		return "GitLab project not found"
	}
	if strings.Contains(strings.ToLower(s), "group not found") {
		return "GitLab group not found"
	}
	if strings.Contains(s, "HTTP 403") {
		return "GitLab forbidden"
	}
	return "GitLab API request failed"
}

const deleteRunnerOp = "gitlabrunner.DeleteRunner"

func newDeleteErr(msg string, err error) *errdefs.Error {
	return errdefs.New(deleteRunnerOp, errdefs.CodeCreateFailed, msg, err)
}

const listOwnedRunnersOp = "gitlabrunner.ListOwnedRunners"

const resetRunnerTokenOp = "gitlabrunner.ResetAuthenticationToken"

const memberAccessOp = "gitlabrunner.MemberAccessLevel"

const ensureCreateRunnerOp = "gitlabrunner.EnsureCanCreateRunner"

// AccessMaintainer is the GitLab access_level for Maintainer (create runners).
const AccessMaintainer = 40

func newResetErr(msg string, err error) *errdefs.Error {
	return errdefs.New(resetRunnerTokenOp, errdefs.CodeCreateFailed, msg, err)
}

// ValidateCreateRunnerRequest ensures runner_type scope fields are set before POST.
func ValidateCreateRunnerRequest(req CreateRunnerRequest) error {
	switch strings.TrimSpace(req.RunnerType) {
	case "project_type":
		if req.ProjectID <= 0 {
			return errdefsMissingProject()
		}
	case "group_type":
		if req.GroupID <= 0 {
			return errdefsMissingGroup()
		}
	}
	return nil
}

func errdefsMissingProject() error {
	return newInvalidScopeErr("project_id is required for project_type")
}

func errdefsMissingGroup() error {
	return newInvalidScopeErr("group_id is required for group_type")
}
