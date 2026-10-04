package gitlabrunner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMatchRunner_lastTokenHostname(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/runners" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]RunnerInfo{
			{ID: 7, Name: "polypus my-mac"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, &fakeExec{}, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := client.MatchRunner(ctx, "glpat-test", "my-mac", srv.URL)
	if err != nil || id != 7 {
		t.Fatalf("match: id=%d err=%v", id, err)
	}
}

func TestMatchRunner_ambiguous(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]RunnerInfo{
			{ID: 1, Name: "host"},
			{ID: 2, Name: "prefix host"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, &fakeExec{}, srv.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.MatchRunner(ctx, "glpat-test", "host", srv.URL)
	if err == nil {
		t.Fatal("expected ambiguous match error")
	}
}

func TestRunnerNameMatches(t *testing.T) {
	if !runnerNameMatches("polypus host", "host") {
		t.Fatal("last token")
	}
	if runnerNameMatches("other", "host") {
		t.Fatal("should not match")
	}
}
