package gitlabrunner

import (
	"context"
	"net/url"
	"strings"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

const matchRunnerOp = "gitlabrunner.MatchRunner"

// MatchRunner finds a GitLab runner id by local register name and optional GitLab URL.
func (c *Client) MatchRunner(ctx context.Context, pat, name, runnerURL string) (int, error) {
	if c == nil {
		return 0, errdefs.New(matchRunnerOp, errdefs.CodeCreateFailed, "client is nil", nil)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errdefs.New(matchRunnerOp, errdefs.CodeInvalidScope, "runner name is required", nil)
	}
	if !urlHostsMatch(runnerURL, c.BaseURL) && strings.TrimSpace(runnerURL) != "" {
		return 0, errdefs.New(matchRunnerOp, errdefs.CodeInvalidScope, "GitLab URL does not match runner config URL", nil)
	}
	list, err := c.ListOwnedRunners(ctx, pat)
	if err != nil {
		return 0, err
	}
	var matches []RunnerInfo
	for _, r := range list {
		if runnerNameMatches(r.Name, name) {
			matches = append(matches, r)
		}
	}
	switch len(matches) {
	case 0:
		return 0, errdefs.New(matchRunnerOp, errdefs.CodeInvalidScope, "GitLab runner not found for name", nil)
	case 1:
		return matches[0].ID, nil
	default:
		return 0, errdefs.New(matchRunnerOp, errdefs.CodeInvalidScope, "multiple GitLab runners match; pass --id", nil)
	}
}

func runnerNameMatches(description, localName string) bool {
	desc := strings.TrimSpace(description)
	local := strings.TrimSpace(localName)
	if desc == "" || local == "" {
		return false
	}
	if desc == local {
		return true
	}
	if strings.EqualFold(desc, local) {
		return true
	}
	parts := strings.Fields(desc)
	if len(parts) > 0 && strings.EqualFold(parts[len(parts)-1], local) {
		return true
	}
	return false
}

func urlHostsMatch(runnerURL, clientBase string) bool {
	rh := hostFromURL(runnerURL)
	ch := hostFromURL(clientBase)
	if rh == "" || ch == "" {
		return true
	}
	return strings.EqualFold(rh, ch)
}

func hostFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}
