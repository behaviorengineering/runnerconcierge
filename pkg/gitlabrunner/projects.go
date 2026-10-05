package gitlabrunner

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

const listMemberProjectsOp = "gitlabrunner.ListMemberProjects"

// MemberProject is a GitLab project the current user can access.
type MemberProject struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
	Name              string `json:"name"`
}

// ListMemberProjects returns projects the authenticated user is a member of.
func (c *Client) ListMemberProjects(ctx context.Context) ([]MemberProject, error) {
	if c == nil {
		return nil, errdefs.New(listMemberProjectsOp, errdefs.CodeCreateFailed, "client is nil", nil)
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, errdefs.New(listMemberProjectsOp, errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	var out []MemberProject
	for page := 1; page <= 10; page++ {
		q := "projects?membership=true&min_access_level=40&simple=true&per_page=100&order_by=last_activity_at&page=" + itoa(page)
		raw, err := c.Exec.Run(ctx, "glab", "api", q)
		if err != nil {
			return nil, errdefs.New(listMemberProjectsOp, errdefs.CodeCreateFailed, glabFailureMessage(err), err)
		}
		var batch []MemberProject
		if err := json.Unmarshal(raw, &batch); err != nil {
			return nil, errdefs.New(listMemberProjectsOp, errdefs.CodeCreateFailed, "parse projects response", err)
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

// ProjectPathFromGitRemote returns group/project from origin when it matches the GitLab host.
func ProjectPathFromGitRemote(ctx context.Context, exec interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}, gitlabBaseURL string) (string, error) {
	if exec == nil {
		return "", nil
	}
	raw, err := exec.Run(ctx, "git", "remote", "get-url", "origin")
	if err != nil {
		return "", nil
	}
	path, ok := ParseGitLabProjectPath(strings.TrimSpace(string(raw)), GitLabHostFromBase(gitlabBaseURL))
	if !ok {
		return "", nil
	}
	return path, nil
}

// GitLabHostFromBase extracts hostname from https://gitlab.com/ style base URL.
func GitLabHostFromBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "gitlab.com"
	}
	if !strings.Contains(base, "://") {
		base = "https://" + base
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
	}
	return u.Host
}

// ParseGitLabProjectPath maps a git remote URL to group/project for a given GitLab host.
func ParseGitLabProjectPath(remoteURL, gitlabHost string) (string, bool) {
	remoteURL = strings.TrimSpace(remoteURL)
	gitlabHost = strings.ToLower(strings.TrimSpace(gitlabHost))
	if remoteURL == "" || gitlabHost == "" {
		return "", false
	}
	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	lower := strings.ToLower(remoteURL)

	if strings.HasPrefix(lower, "git@") {
		rest := remoteURL[4:]
		colon := strings.Index(rest, ":")
		if colon < 0 {
			return "", false
		}
		host := strings.ToLower(rest[:colon])
		if host != gitlabHost {
			return "", false
		}
		path := strings.Trim(rest[colon+1:], "/")
		if path == "" {
			return "", false
		}
		return path, true
	}

	u, err := url.Parse(remoteURL)
	if err != nil || u.Host == "" {
		return "", false
	}
	if strings.ToLower(u.Host) != gitlabHost {
		return "", false
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "", false
	}
	return path, true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
