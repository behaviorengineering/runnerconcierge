package gitlabrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

// CreateRunnerRequest is POST /user/runners payload.
type CreateRunnerRequest struct {
	RunnerType   string
	ProjectID    int
	GroupID      int
	Description  string
	TagList      []string
	RunUntagged  bool
	Locked       bool
	Paused       bool
}

// Client talks to GitLab runner APIs.
type Client struct {
	BaseURL string
	Exec    gitexec.Exec
	HTTP    *http.Client
	Token   string // PAT for HTTP fallback
}

// NewClient requires exec and base URL.
func NewClient(baseURL string, exec gitexec.Exec, httpClient *http.Client) *Client {
	if exec == nil {
		panic("gitlabrunner: exec is nil")
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &Client{BaseURL: baseURL, Exec: exec, HTTP: httpClient}
}

// CreateRunner registers a runner in GitLab and returns id + glrt token.
func (c *Client) CreateRunner(ctx context.Context, req CreateRunnerRequest, pat string) (int, string, error) {
	if c == nil {
		return 0, "", fmt.Errorf("gitlabrunner: client is nil")
	}
	if _, ok := ctx.Deadline(); !ok {
		return 0, "", errdefs.New("CreateRunner", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if strings.TrimSpace(pat) != "" {
		return c.createHTTP(ctx, req, pat)
	}
	return c.createGlab(ctx, req)
}

func (c *Client) createGlab(ctx context.Context, req CreateRunnerRequest) (int, string, error) {
	args := []string{"api", "--method", "POST", "user/runners",
		"-F", "runner_type=" + req.RunnerType,
		"-F", fmt.Sprintf("description=%s", req.Description),
		"-F", fmt.Sprintf("run_untagged=%t", req.RunUntagged),
		"-F", fmt.Sprintf("locked=%t", req.Locked),
		"-F", fmt.Sprintf("paused=%t", req.Paused),
	}
	if req.ProjectID > 0 {
		args = append(args, "-F", fmt.Sprintf("project_id=%d", req.ProjectID))
	}
	if req.GroupID > 0 {
		args = append(args, "-F", fmt.Sprintf("group_id=%d", req.GroupID))
	}
	if len(req.TagList) > 0 {
		args = append(args, "-F", "tag_list="+strings.Join(req.TagList, ","))
	}
	out, err := c.Exec.Run(ctx, "glab", args...)
	if err != nil {
		return 0, "", errdefs.New("CreateRunner", errdefs.CodeCreateFailed, "glab api failed", err)
	}
	var resp struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, "", errdefs.New("CreateRunner", errdefs.CodeCreateFailed, "parse response", err)
	}
	if resp.Token == "" {
		return 0, "", errdefs.New("CreateRunner", errdefs.CodeCreateFailed, "empty token in response", nil)
	}
	return resp.ID, resp.Token, nil
}

func (c *Client) createHTTP(ctx context.Context, req CreateRunnerRequest, pat string) (int, string, error) {
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
	form := url.Values{}
	form.Set("runner_type", req.RunnerType)
	form.Set("description", req.Description)
	form.Set("run_untagged", fmt.Sprintf("%t", req.RunUntagged))
	form.Set("locked", fmt.Sprintf("%t", req.Locked))
	form.Set("paused", fmt.Sprintf("%t", req.Paused))
	if req.ProjectID > 0 {
		form.Set("project_id", fmt.Sprintf("%d", req.ProjectID))
	}
	if req.GroupID > 0 {
		form.Set("group_id", fmt.Sprintf("%d", req.GroupID))
	}
	if len(req.TagList) > 0 {
		form.Set("tag_list", strings.Join(req.TagList, ","))
	}
	endpoint := c.BaseURL + "/api/v4/user/runners"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, "", err
	}
	httpReq.Header.Set("PRIVATE-TOKEN", pat)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return 0, "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusForbidden {
		return 0, "", errdefs.New("CreateRunner", errdefs.CodeAuthScopeInsufficient, "forbidden; need create_runner scope", nil)
	}
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		return 0, "", errdefs.New("CreateRunner", errdefs.CodeCreateFailed, fmt.Sprintf("http %d: %s", res.StatusCode, string(body)), nil)
	}
	var parsed struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, "", err
	}
	return parsed.ID, parsed.Token, nil
}

// WhoAmI returns the authenticated GitLab username via glab.
func (c *Client) WhoAmI(ctx context.Context) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		return "", errdefs.New("WhoAmI", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	out, err := c.Exec.Run(ctx, "glab", "api", "user")
	if err != nil {
		return "", errdefs.New("WhoAmI", errdefs.CodeAuthUnauthorized, "glab user", err)
	}
	var u struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(out, &u); err != nil {
		return "", err
	}
	if u.Username == "" {
		return "", errdefs.New("WhoAmI", errdefs.CodeAuthRequired, "not logged in; run glab auth login", nil)
	}
	return u.Username, nil
}

// RunnerInfo is a project runner summary.
type RunnerInfo struct {
	ID     int    `json:"id"`
	Online bool   `json:"online"`
	Name   string `json:"description"`
}

// ListRunners returns runners for a project id.
func (c *Client) ListRunners(ctx context.Context, projectID int) ([]RunnerInfo, error) {
	if projectID <= 0 {
		return nil, fmt.Errorf("gitlabrunner: project id required")
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, errdefs.New("ListRunners", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	out, err := c.Exec.Run(ctx, "glab", "api", fmt.Sprintf("projects/%d/runners", projectID))
	if err != nil {
		return nil, err
	}
	var list []RunnerInfo
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// ResolveProjectID looks up numeric project id from path.
func (c *Client) ResolveProjectID(ctx context.Context, projectPath string) (int, error) {
	if _, ok := ctx.Deadline(); !ok {
		return 0, errdefs.New("ResolveProjectID", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	enc := url.PathEscape(projectPath)
	out, err := c.Exec.Run(ctx, "glab", "api", "projects/"+enc)
	if err != nil {
		return 0, err
	}
	var resp struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, err
	}
	if resp.ID == 0 {
		return 0, fmt.Errorf("gitlabrunner: project id not found for %s", projectPath)
	}
	return resp.ID, nil
}

// WaitOnline polls runner status until online or timeout.
func (c *Client) WaitOnline(ctx context.Context, runnerID int, timeout time.Duration) error {
	if runnerID <= 0 {
		return fmt.Errorf("gitlabrunner: invalid runner id")
	}
	deadline := time.Now().Add(timeout)
	for {
		if _, ok := ctx.Deadline(); !ok {
			return errdefs.New("WaitOnline", errdefs.CodeMissingDeadline, "context missing deadline", nil)
		}
		out, err := c.Exec.Run(ctx, "glab", "api", fmt.Sprintf("runners/%d", runnerID))
		if err == nil {
			var st struct {
				Online bool `json:"online"`
			}
			if json.Unmarshal(out, &st) == nil && st.Online {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return errdefs.New("WaitOnline", errdefs.CodeRunnerOffline, "runner did not become online", nil)
		}
		time.Sleep(2 * time.Second)
	}
}

// Register runs gitlab-runner register with built argv.
func (c *Client) Register(ctx context.Context, runnerBin string, args []string) error {
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("Register", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	full := append([]string{}, args...)
	_, err := c.Exec.Run(ctx, runnerBin, full...)
	if err != nil {
		return errdefs.New("Register", errdefs.CodeRegisterFailed, "register failed", err)
	}
	return nil
}
