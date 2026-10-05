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
	RunnerType  string
	ProjectID   int
	GroupID     int
	Description string
	TagList     []string
	RunUntagged bool
	Locked      bool
	Paused      bool
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
		return 0, "", errdefs.New(createRunnerOp, errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if err := ValidateCreateRunnerRequest(req); err != nil {
		return 0, "", err
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
		return 0, "", newCreateErr(glabFailureMessage(err), err)
	}
	var resp struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, "", newCreateErr("parse response", err)
	}
	if resp.Token == "" {
		return 0, "", newCreateErr("empty token in response", nil)
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
		return 0, "", newCreateErr("build request", err)
	}
	httpReq.Header.Set("PRIVATE-TOKEN", pat)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return 0, "", newCreateErr("HTTP request failed", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusForbidden {
		return 0, "", errdefs.New(createRunnerOp, errdefs.CodeAuthScopeInsufficient, "forbidden; need create_runner scope", nil)
	}
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		msg := gitlabMessageFromBody(body)
		if msg == "" {
			msg = fmt.Sprintf("GitLab returned HTTP %d", res.StatusCode)
		}
		return 0, "", newCreateErr(msg, nil)
	}
	var parsed struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, "", newCreateErr("parse response", err)
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
		return nil, errdefs.New("gitlabrunner.ListRunners", errdefs.CodeCreateFailed, glabFailureMessage(err), err)
	}
	var list []RunnerInfo
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, errdefs.New("gitlabrunner.ListRunners", errdefs.CodeCreateFailed, "parse response", err)
	}
	return list, nil
}

// ListOwnedRunners returns runners visible to the authenticated user.
func (c *Client) ListOwnedRunners(ctx context.Context, pat string) ([]RunnerInfo, error) {
	if c == nil {
		return nil, fmt.Errorf("gitlabrunner: client is nil")
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, errdefs.New(listOwnedRunnersOp, errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if strings.TrimSpace(pat) != "" {
		return c.listOwnedHTTP(ctx, pat)
	}
	return c.listOwnedGlab(ctx)
}

func (c *Client) listOwnedGlab(ctx context.Context) ([]RunnerInfo, error) {
	var all []RunnerInfo
	for page := 1; page <= 50; page++ {
		out, err := c.Exec.Run(ctx, "glab", "api", fmt.Sprintf("runners?page=%d&per_page=100", page))
		if err != nil {
			return nil, errdefs.New(listOwnedRunnersOp, errdefs.CodeCreateFailed, glabFailureMessage(err), err)
		}
		var batch []RunnerInfo
		if err := json.Unmarshal(out, &batch); err != nil {
			return nil, errdefs.New(listOwnedRunnersOp, errdefs.CodeCreateFailed, "parse response", err)
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
	}
	return all, nil
}

func (c *Client) listOwnedHTTP(ctx context.Context, pat string) ([]RunnerInfo, error) {
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
	var all []RunnerInfo
	for page := 1; page <= 50; page++ {
		endpoint := fmt.Sprintf("%s/api/v4/runners?page=%d&per_page=100", c.BaseURL, page)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, errdefs.New(listOwnedRunnersOp, errdefs.CodeCreateFailed, "build request", err)
		}
		httpReq.Header.Set("PRIVATE-TOKEN", pat)
		res, err := c.HTTP.Do(httpReq)
		if err != nil {
			return nil, errdefs.New(listOwnedRunnersOp, errdefs.CodeCreateFailed, "HTTP request failed", err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode == http.StatusForbidden {
			return nil, errdefs.New(listOwnedRunnersOp, errdefs.CodeAuthScopeInsufficient, "forbidden listing runners", nil)
		}
		if res.StatusCode != http.StatusOK {
			msg := gitlabMessageFromBody(body)
			if msg == "" {
				msg = fmt.Sprintf("GitLab returned HTTP %d", res.StatusCode)
			}
			return nil, errdefs.New(listOwnedRunnersOp, errdefs.CodeCreateFailed, msg, nil)
		}
		var batch []RunnerInfo
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, errdefs.New(listOwnedRunnersOp, errdefs.CodeCreateFailed, "parse response", err)
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
	}
	return all, nil
}

// DeleteRunner removes a GitLab runner by id (HTTP PAT when set, else glab).
func (c *Client) DeleteRunner(ctx context.Context, runnerID int, pat string) error {
	if c == nil {
		return errdefs.New(deleteRunnerOp, errdefs.CodeCreateFailed, "client is nil", nil)
	}
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New(deleteRunnerOp, errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	if runnerID <= 0 {
		return errdefs.New(deleteRunnerOp, errdefs.CodeInvalidScope, "runner id is required", nil)
	}
	if strings.TrimSpace(pat) != "" {
		return c.deleteHTTP(ctx, runnerID, pat)
	}
	return c.deleteGlab(ctx, runnerID)
}

func (c *Client) deleteGlab(ctx context.Context, runnerID int) error {
	_, err := c.Exec.Run(ctx, "glab", "api", "--method", "DELETE", fmt.Sprintf("runners/%d", runnerID))
	if err != nil {
		msg := glabFailureMessage(err)
		if strings.Contains(strings.ToLower(msg), "forbidden") || strings.Contains(err.Error(), "HTTP 403") {
			return errdefs.New(deleteRunnerOp, errdefs.CodeAuthScopeInsufficient, "forbidden; need permission to delete runner", err)
		}
		return newDeleteErr(msg, err)
	}
	return nil
}

func (c *Client) deleteHTTP(ctx context.Context, runnerID int, pat string) error {
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
	endpoint := fmt.Sprintf("%s/api/v4/runners/%d", c.BaseURL, runnerID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return newDeleteErr("build request", err)
	}
	httpReq.Header.Set("PRIVATE-TOKEN", pat)
	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return newDeleteErr("HTTP request failed", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusForbidden {
		return errdefs.New(deleteRunnerOp, errdefs.CodeAuthScopeInsufficient, "forbidden; need permission to delete runner", nil)
	}
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		msg := gitlabMessageFromBody(body)
		if msg == "" {
			msg = fmt.Sprintf("GitLab returned HTTP %d", res.StatusCode)
		}
		return newDeleteErr(msg, nil)
	}
	return nil
}

// Unregister runs gitlab-runner unregister with built argv.
func (c *Client) Unregister(ctx context.Context, runnerBin string, args []string) error {
	if _, ok := ctx.Deadline(); !ok {
		return errdefs.New("Unregister", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	_, err := c.Exec.Run(ctx, runnerBin, args...)
	if err != nil {
		return errdefs.New("Unregister", errdefs.CodeRegisterFailed, "unregister failed", err)
	}
	return nil
}

// ResolveProjectID looks up numeric project id from path.
func (c *Client) ResolveProjectID(ctx context.Context, projectPath string) (int, error) {
	if _, ok := ctx.Deadline(); !ok {
		return 0, errdefs.New("ResolveProjectID", errdefs.CodeMissingDeadline, "context missing deadline", nil)
	}
	enc := url.PathEscape(projectPath)
	out, err := c.Exec.Run(ctx, "glab", "api", "projects/"+enc)
	if err != nil {
		return 0, errdefs.New("gitlabrunner.ResolveProjectID", errdefs.CodeCreateFailed, glabFailureMessage(err), err)
	}
	var resp struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, errdefs.New("gitlabrunner.ResolveProjectID", errdefs.CodeCreateFailed, "parse response", err)
	}
	if resp.ID == 0 {
		return 0, errdefs.New("gitlabrunner.ResolveProjectID", errdefs.CodeInvalidScope, "project id not found for "+projectPath, nil)
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
		msg := "register failed"
		if hint := childErrorSummary(err); hint != "" {
			msg += ": " + hint
		}
		return errdefs.New("Register", errdefs.CodeRegisterFailed, msg, err)
	}
	return nil
}
