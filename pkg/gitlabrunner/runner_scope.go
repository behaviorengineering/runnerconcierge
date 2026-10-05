package gitlabrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

const (
	getRunnerOp          = "gitlabrunner.GetRunner"
	resolveRunnerScopeOp = "gitlabrunner.ResolveRunnerScope"
)

// RunnerDetail is a subset of GET /runners/:id used for resume scope hydration.
type RunnerDetail struct {
	ID          int
	RunnerType  string
	GroupID     int
	ProjectID   int
	GroupPath   string
	ProjectPath string
	TagList     []string
}

type runnerGroupRef struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"full_path"`
	WebURL   string `json:"web_url"`
}

type runnerProjectRef struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
}

func groupPathFromRef(g runnerGroupRef) string {
	if p := strings.TrimSpace(g.FullPath); p != "" {
		return p
	}
	if p := strings.TrimSpace(g.Name); p != "" {
		return p
	}
	return pathFromGitLabWebURL(g.WebURL, "groups")
}

func projectPathFromRef(p runnerProjectRef) string {
	if path := strings.TrimSpace(p.PathWithNamespace); path != "" {
		return path
	}
	return pathFromGitLabWebURL(p.WebURL, "projects")
}

func pathFromGitLabWebURL(webURL, segment string) string {
	webURL = strings.TrimSpace(webURL)
	if webURL == "" {
		return ""
	}
	u, err := url.Parse(webURL)
	if err != nil {
		return ""
	}
	prefix := "/" + segment + "/"
	if !strings.HasPrefix(u.Path, prefix) {
		return ""
	}
	return strings.Trim(strings.TrimPrefix(u.Path, prefix), "/")
}

// GetRunner loads runner metadata via glab api runners/:id.
func (c *Client) GetRunner(ctx context.Context, runnerID int) (RunnerDetail, error) {
	if c == nil {
		return RunnerDetail{}, errdefs.New(getRunnerOp, errdefs.CodeCreateFailed, "client is nil", nil)
	}
	if _, ok := ctx.Deadline(); !ok {
		return RunnerDetail{}, errdefs.New(getRunnerOp, errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if runnerID <= 0 {
		return RunnerDetail{}, errdefs.New(getRunnerOp, errdefs.CodeInvalidScope, "runner id is required", nil)
	}
	out, err := c.Exec.Run(ctx, "glab", "api", fmt.Sprintf("runners/%d", runnerID))
	if err != nil {
		return RunnerDetail{}, errdefs.New(getRunnerOp, errdefs.CodeCreateFailed, glabFailureMessage(err), err)
	}
	var raw struct {
		ID         int                `json:"id"`
		RunnerType string             `json:"runner_type"`
		GroupID    int                `json:"group_id"`
		ProjectID  int                `json:"project_id"`
		TagList    []string           `json:"tag_list"`
		Groups     []runnerGroupRef   `json:"groups"`
		Projects   []runnerProjectRef `json:"projects"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return RunnerDetail{}, errdefs.New(getRunnerOp, errdefs.CodeCreateFailed, "parse response", err)
	}
	if raw.ID == 0 {
		return RunnerDetail{}, errdefs.New(getRunnerOp, errdefs.CodeInvalidScope, "runner not found", nil)
	}
	detail := RunnerDetail{
		ID:         raw.ID,
		RunnerType: strings.TrimSpace(raw.RunnerType),
		GroupID:    raw.GroupID,
		ProjectID:  raw.ProjectID,
		TagList:    raw.TagList,
	}
	if detail.GroupID == 0 && len(raw.Groups) > 0 {
		detail.GroupID = raw.Groups[0].ID
		detail.GroupPath = groupPathFromRef(raw.Groups[0])
	}
	if detail.ProjectID == 0 && len(raw.Projects) > 0 {
		detail.ProjectID = raw.Projects[0].ID
		detail.ProjectPath = projectPathFromRef(raw.Projects[0])
	}
	return detail, nil
}

// ResolveRunnerScope returns runner_type and namespace path (group full_path or project path_with_namespace).
func (c *Client) ResolveRunnerScope(ctx context.Context, runnerID int) (runnerType, path string, err error) {
	detail, err := c.GetRunner(ctx, runnerID)
	if err != nil {
		return "", "", err
	}
	switch detail.RunnerType {
	case "group_type":
		if path := strings.TrimSpace(detail.GroupPath); path != "" {
			return "group_type", path, nil
		}
		if detail.GroupID <= 0 {
			return "", "", errdefs.New(resolveRunnerScopeOp, errdefs.CodeInvalidScope, "runner has no group scope", nil)
		}
		path, err := c.ResolveGroupPathByID(ctx, detail.GroupID)
		if err != nil {
			return "", "", errdefs.New(resolveRunnerScopeOp, errdefs.CodeOf(err), "could not resolve group path", err)
		}
		return "group_type", path, nil
	case "project_type":
		if path := strings.TrimSpace(detail.ProjectPath); path != "" {
			return "project_type", path, nil
		}
		if detail.ProjectID <= 0 {
			return "", "", errdefs.New(resolveRunnerScopeOp, errdefs.CodeInvalidScope, "runner has no project scope", nil)
		}
		path, err := c.ResolveProjectPathByID(ctx, detail.ProjectID)
		if err != nil {
			return "", "", errdefs.New(resolveRunnerScopeOp, errdefs.CodeOf(err), "could not resolve project path", err)
		}
		return "project_type", path, nil
	default:
		return "", "", errdefs.New(resolveRunnerScopeOp, errdefs.CodeInvalidScope,
			"unsupported runner_type "+detail.RunnerType, nil)
	}
}
