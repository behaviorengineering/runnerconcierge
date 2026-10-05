package gitlabrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

// EnsureCanCreateRunner verifies the user has Maintainer+ on the target group or project.
func (c *Client) EnsureCanCreateRunner(ctx context.Context, runnerType, path string, user GitLabUser) error {
	if c == nil {
		return errdefs.New(ensureCreateRunnerOp, errdefs.CodeCreateFailed, "client is nil", nil)
	}
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New(ensureCreateRunnerOp, errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if user.ID <= 0 {
		return errdefs.New(ensureCreateRunnerOp, errdefs.CodeAuthRequired, "not logged in; run glab auth login", nil)
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return errdefs.New(ensureCreateRunnerOp, errdefs.CodeInvalidScope, "group or project path is required", nil)
	}
	switch runnerType {
	case "group_type":
		id, err := c.ResolveGroupID(ctx, path)
		if err != nil {
			return err
		}
		level, err := c.MemberAccessLevel(ctx, "groups", id, user.ID)
		if err != nil {
			return err
		}
		if level < AccessMaintainer {
			return errdefs.New(ensureCreateRunnerOp, errdefs.CodeAuthScopeInsufficient,
				fmt.Sprintf("need Maintainer on group %s", path), nil)
		}
		return nil
	case "project_type":
		id, err := c.ResolveProjectID(ctx, path)
		if err != nil {
			return err
		}
		level, err := c.MemberAccessLevel(ctx, "projects", id, user.ID)
		if err != nil {
			return err
		}
		if level < AccessMaintainer {
			return errdefs.New(ensureCreateRunnerOp, errdefs.CodeAuthScopeInsufficient,
				fmt.Sprintf("need Maintainer on project %s", path), nil)
		}
		return nil
	default:
		return errdefs.New(ensureCreateRunnerOp, errdefs.CodeInvalidScope, "runner type is required", nil)
	}
}

// MemberAccessLevel returns the user's access_level on a group or project (includes inherited).
func (c *Client) MemberAccessLevel(ctx context.Context, resource string, resourceID, userID int) (int, error) {
	if c == nil {
		return 0, errdefs.New(memberAccessOp, errdefs.CodeCreateFailed, "client is nil", nil)
	}
	if _, ok := ctx.Deadline(); !ok {
		return 0, errdefs.New(memberAccessOp, errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if resourceID <= 0 || userID <= 0 {
		return 0, errdefs.New(memberAccessOp, errdefs.CodeInvalidScope, "resource and user id required", nil)
	}
	switch resource {
	case "groups", "projects":
	default:
		return 0, errdefs.New(memberAccessOp, errdefs.CodeInvalidScope, "resource must be groups or projects", nil)
	}
	path := fmt.Sprintf("%s/%d/members/all/%d", resource, resourceID, userID)
	out, err := c.Exec.Run(ctx, "glab", "api", path)
	if err != nil {
		msg := glabFailureMessage(err)
		low := strings.ToLower(err.Error())
		if strings.Contains(strings.ToLower(msg), "forbidden") || strings.Contains(low, "http 403") || strings.Contains(low, "404") {
			return 0, errdefs.New(memberAccessOp, errdefs.CodeAuthScopeInsufficient, "not a member or insufficient access", err)
		}
		return 0, errdefs.New(memberAccessOp, errdefs.CodeAuthScopeInsufficient, "not a member or insufficient access", err)
	}
	var resp struct {
		AccessLevel int `json:"access_level"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, errdefs.New(memberAccessOp, errdefs.CodeCreateFailed, "parse membership response", err)
	}
	return resp.AccessLevel, nil
}
