package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dslh/zh/internal/cache"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

func TestCompletionBash(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "bash"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("completion bash returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "bash") {
		t.Error("output should contain bash completion script")
	}
}

func TestCompletionZsh(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "zsh"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("completion zsh returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "zsh") || !strings.Contains(out, "compdef") {
		t.Error("output should contain zsh completion script")
	}
}

func TestCompletionFish(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "fish"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("completion fish returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "fish") || !strings.Contains(out, "complete") {
		t.Error("output should contain fish completion script")
	}
}

func TestCompletionHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("completion --help returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "bash") {
		t.Error("help should mention bash")
	}
	if !strings.Contains(out, "zsh") {
		t.Error("help should mention zsh")
	}
	if !strings.Contains(out, "fish") {
		t.Error("help should mention fish")
	}
}

func TestCompletePipelineNames(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	_ = cache.Set(resolve.PipelineCacheKey("ws-123"), []resolve.CachedPipeline{
		{ID: "p1", Name: "Backlog"},
		{ID: "p2", Name: "In Progress"},
		{ID: "p3", Name: "Done"},
	})

	names, directive := completePipelineNames(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}

	if len(names) < 3 {
		t.Fatalf("got %d names, want at least 3", len(names))
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"Backlog", "In Progress", "Done"} {
		if !found[want] {
			t.Errorf("missing pipeline name %q", want)
		}
	}
}

func TestCompleteSprintNames(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	_ = cache.Set(resolve.SprintCacheKey("ws-123"), []resolve.CachedSprint{
		{ID: "s1", Name: "Sprint 10", State: "OPEN"},
		{ID: "s2", Name: "Sprint 11", State: "OPEN"},
	})

	names, directive := completeSprintNames(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"current", "next", "previous", "Sprint 10", "Sprint 11"} {
		if !found[want] {
			t.Errorf("missing sprint name %q", want)
		}
	}
}

func TestCompleteEpicNames(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	_ = cache.Set(resolve.EpicCacheKey("ws-123"), []resolve.CachedEpic{
		{ID: "e1", Title: "Auth System", Type: "zenhub"},
		{ID: "e2", Title: "Billing Overhaul", Type: "zenhub"},
	})

	names, directive := completeEpicNames(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"Auth System", "Billing Overhaul"} {
		if !found[want] {
			t.Errorf("missing epic name %q", want)
		}
	}
}

func TestCompleteWorkspaceNames(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	type cachedWS struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		OrgName     string `json:"orgName"`
	}
	_ = cache.Set(cache.NewKey("workspaces"), []cachedWS{
		{ID: "ws1", Name: "team-alpha", DisplayName: "Team Alpha"},
		{ID: "ws2", Name: "team-beta", DisplayName: "Team Beta"},
	})

	names, directive := completeWorkspaceNames(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"Team Alpha", "Team Beta"} {
		if !found[want] {
			t.Errorf("missing workspace name %q", want)
		}
	}
}

func TestCompleteRepoNames(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	_ = cache.Set(resolve.RepoCacheKey("ws-123"), []resolve.CachedRepo{
		{ID: "r1", Name: "frontend", OwnerName: "acme"},
		{ID: "r2", Name: "backend", OwnerName: "acme"},
	})

	names, directive := completeRepoNames(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"frontend", "backend"} {
		if !found[want] {
			t.Errorf("missing repo name %q", want)
		}
	}
}

func TestCompleteLabelNames(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	_ = cache.Set(resolve.LabelCacheKey("ws-123"), []resolve.CachedLabel{
		{ID: "l1", Name: "bug", Color: "red"},
		{ID: "l2", Name: "enhancement", Color: "blue"},
	})

	names, directive := completeLabelNames(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"bug", "enhancement"} {
		if !found[want] {
			t.Errorf("missing label name %q", want)
		}
	}
}

func TestCompletePriorityNames(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	_ = cache.Set(resolve.PriorityCacheKey("ws-123"), []resolve.CachedPriority{
		{ID: "p1", Name: "Urgent", Color: "red"},
		{ID: "p2", Name: "High", Color: "orange"},
		{ID: "p3", Name: "Medium", Color: "yellow"},
	})

	names, directive := completePriorityNames(nil, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"Urgent", "High", "Medium"} {
		if !found[want] {
			t.Errorf("missing priority name %q", want)
		}
	}
}

func TestCompleteNoCacheReturnsEmpty(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-empty")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	// With no cache populated, completions should return nil gracefully
	for name, fn := range map[string]func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective){
		"pipeline": completePipelineNames,
		"sprint":   completeSprintNames,
		"epic":     completeEpicNames,
		"repo":     completeRepoNames,
		"label":    completeLabelNames,
		"priority": completePriorityNames,
	} {
		names, directive := fn(nil, nil, "")
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("%s: directive = %v, want NoFileComp", name, directive)
		}
		// Sprint returns relative refs even without cache
		if name == "sprint" {
			continue
		}
		if names != nil {
			t.Errorf("%s: got names %v, want nil", name, names)
		}
	}
}

func TestCompleteNoWorkspaceReturnsEmpty(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	names, directive := completePipelineNames(nil, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
	if names != nil {
		t.Errorf("got names %v, want nil when no workspace configured", names)
	}
}

func TestCompleteEpicStates(t *testing.T) {
	names, directive := completeEpicStates(nil, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
	want := map[string]bool{"open": true, "todo": true, "in_progress": true, "closed": true}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected state %q", n)
		}
		delete(want, n)
	}
	for s := range want {
		t.Errorf("missing state %q", s)
	}
}

func TestCompletionDoesNotTriggerSetup(t *testing.T) {
	// The completion command should be skippable (no setup wizard)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("ZH_API_KEY", "")
	t.Setenv("ZH_WORKSPACE", "")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"completion", "bash"})

	// Should not trigger the setup wizard even without config
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("completion bash should work without config: %v", err)
	}
}
