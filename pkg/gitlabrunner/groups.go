package gitlabrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

const listMemberGroupsOp = "gitlabrunner.ListMemberGroups"

// MemberGroup is a GitLab group the current user can access.
type MemberGroup struct {
	ID       int    `json:"id"`
	FullPath string `json:"full_path"`
	Name     string `json:"name"`
}

// ListMemberGroups returns groups the authenticated user is a member of.
func (c *Client) ListMemberGroups(ctx context.Context) ([]MemberGroup, error) {
	if c == nil {
		return nil, errdefs.New(listMemberGroupsOp, errdefs.CodeCreateFailed, "client is nil", nil)
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, errdefs.New(listMemberGroupsOp, errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	var out []MemberGroup
	for page := 1; page <= 10; page++ {
		q := "groups?membership=true&min_access_level=40&per_page=100&page=" + itoa(page)
		raw, err := c.Exec.Run(ctx, "glab", "api", q)
		if err != nil {
			return nil, errdefs.New(listMemberGroupsOp, errdefs.CodeCreateFailed, glabFailureMessage(err), err)
		}
		var batch []MemberGroup
		if err := json.Unmarshal(raw, &batch); err != nil {
			return nil, errdefs.New(listMemberGroupsOp, errdefs.CodeCreateFailed, "parse groups response", err)
		}
		if len(batch) == 0 {
			break
		}
		out = append(out, batch...)
		if len(batch) < 100 {
			break
		}
	}
	return out, nil
}

// ResolveGroupID looks up numeric group id from full path.
func (c *Client) ResolveGroupID(ctx context.Context, groupPath string) (int, error) {
	if _, ok := ctx.Deadline(); !ok {
		return 0, errdefs.New("ResolveGroupID", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	enc := url.PathEscape(groupPath)
	out, err := c.Exec.Run(ctx, "glab", "api", "groups/"+enc)
	if err != nil {
		return 0, errdefs.New("gitlabrunner.ResolveGroupID", errdefs.CodeCreateFailed, glabFailureMessage(err), err)
	}
	var resp struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, errdefs.New("gitlabrunner.ResolveGroupID", errdefs.CodeCreateFailed, "parse response", err)
	}
	if resp.ID == 0 {
		return 0, errdefs.New("gitlabrunner.ResolveGroupID", errdefs.CodeInvalidScope, "group id not found for "+groupPath, nil)
	}
	return resp.ID, nil
}

// ResolveGroupPathByID returns full_path for a numeric group id.
func (c *Client) ResolveGroupPathByID(ctx context.Context, groupID int) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		return "", errdefs.New("ResolveGroupPathByID", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if groupID <= 0 {
		return "", errdefs.New("gitlabrunner.ResolveGroupPathByID", errdefs.CodeInvalidScope, "group id is required", nil)
	}
	out, err := c.Exec.Run(ctx, "glab", "api", fmt.Sprintf("groups/%d", groupID))
	if err != nil {
		return "", errdefs.New("gitlabrunner.ResolveGroupPathByID", errdefs.CodeCreateFailed, glabFailureMessage(err), err)
	}
	var resp struct {
		FullPath string `json:"full_path"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", errdefs.New("gitlabrunner.ResolveGroupPathByID", errdefs.CodeCreateFailed, "parse response", err)
	}
	path := strings.TrimSpace(resp.FullPath)
	if path == "" {
		return "", errdefs.New("gitlabrunner.ResolveGroupPathByID", errdefs.CodeInvalidScope, "group path not found", nil)
	}
	return path, nil
}
