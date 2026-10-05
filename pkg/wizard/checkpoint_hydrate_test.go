package wizard

import (
	"testing"

	"github.com/behaviorengineering/runnerconcierge/pkg/state"
)

func TestApplyCheckpointToOpts(t *testing.T) {
	cp := &state.Checkpoint{
		Tags:      []string{"lab", "docker"},
		Executor:  "docker",
		RepoPath:  "acme/widget",
		GroupPath: "acme",
	}
	opts := &Options{}
	applyCheckpointToOpts(opts, cp)
	if len(opts.TagList) != 2 || opts.TagList[0] != "lab" {
		t.Fatalf("tags %v", opts.TagList)
	}
	if opts.Executor != "docker" {
		t.Fatalf("executor %q", opts.Executor)
	}
	if opts.ProjectPath != "acme/widget" || opts.GroupPath != "acme" {
		t.Fatalf("paths project=%q group=%q", opts.ProjectPath, opts.GroupPath)
	}
	if opts.RunnerType != runnerTypeGroup {
		t.Fatalf("runner type %q", opts.RunnerType)
	}
	cp.Tags[0] = "mutated"
	if opts.TagList[0] == "mutated" {
		t.Fatal("TagList must not alias checkpoint slice")
	}
}

func TestApplyCheckpointToOptsRespectsCLI(t *testing.T) {
	cp := &state.Checkpoint{RepoPath: "from/cp", Tags: []string{"cp-tag"}}
	opts := &Options{ProjectPath: "from/cli", TagList: []string{"cli-tag"}}
	applyCheckpointToOpts(opts, cp)
	if opts.ProjectPath != "from/cli" {
		t.Fatalf("project %q", opts.ProjectPath)
	}
	if len(opts.TagList) != 1 || opts.TagList[0] != "cli-tag" {
		t.Fatalf("tags %v", opts.TagList)
	}
}

func TestApplyCheckpointInfersProjectType(t *testing.T) {
	cp := &state.Checkpoint{RepoPath: "g/p"}
	opts := &Options{}
	applyCheckpointToOpts(opts, cp)
	if opts.RunnerType != runnerTypeProject {
		t.Fatalf("type %q", opts.RunnerType)
	}
}
