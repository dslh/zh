package cmd

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- selectItem ---

func TestSelectItemTitle(t *testing.T) {
	item := selectItem{id: "id1", title: "Test Item", description: "desc"}
	if item.Title() != "Test Item" {
		t.Errorf("Title() = %q, want %q", item.Title(), "Test Item")
	}
}

func TestSelectItemDescription(t *testing.T) {
	item := selectItem{id: "id1", title: "Test Item", description: "some description"}
	if item.Description() != "some description" {
		t.Errorf("Description() = %q, want %q", item.Description(), "some description")
	}
}

func TestSelectItemFilterValue(t *testing.T) {
	item := selectItem{id: "id1", title: "Test Item", description: "some description"}
	fv := item.FilterValue()
	if !strings.Contains(fv, "Test Item") {
		t.Error("FilterValue should contain the title")
	}
	if !strings.Contains(fv, "some description") {
		t.Error("FilterValue should contain the description")
	}
}

// --- selectModel ---

func TestSelectModelInit(t *testing.T) {
	items := []selectItem{
		{id: "1", title: "Item 1", description: "desc 1"},
		{id: "2", title: "Item 2", description: "desc 2"},
	}
	m := newSelectModel("Test", items)

	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestSelectModelCtrlC(t *testing.T) {
	items := []selectItem{
		{id: "1", title: "Item 1"},
	}
	m := newSelectModel("Test", items)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := updated.(selectModel)

	if !result.result.cancelled {
		t.Error("ctrl+c should cancel the selection")
	}
	if !result.quitting {
		t.Error("ctrl+c should set quitting")
	}
}

func TestSelectModelEsc(t *testing.T) {
	items := []selectItem{
		{id: "1", title: "Item 1"},
	}
	m := newSelectModel("Test", items)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := updated.(selectModel)

	if !result.result.cancelled {
		t.Error("esc should cancel the selection")
	}
}

func TestSelectModelEnter(t *testing.T) {
	items := []selectItem{
		{id: "id-1", title: "First Item"},
		{id: "id-2", title: "Second Item"},
	}
	m := newSelectModel("Test", items)

	// Press enter to select the first item (which is selected by default)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(selectModel)

	if result.result.cancelled {
		t.Error("enter should not cancel")
	}
	if result.result.id != "id-1" {
		t.Errorf("selected id = %q, want %q", result.result.id, "id-1")
	}
	if result.result.title != "First Item" {
		t.Errorf("selected title = %q, want %q", result.result.title, "First Item")
	}
}

func TestSelectModelViewNotQuitting(t *testing.T) {
	items := []selectItem{
		{id: "1", title: "Item 1"},
	}
	m := newSelectModel("Test", items)

	view := m.View()
	if view == "" {
		t.Error("View() should not be empty when not quitting")
	}
	if !strings.Contains(view, "Esc") {
		t.Error("View() should contain Esc hint")
	}
}

func TestSelectModelViewQuitting(t *testing.T) {
	items := []selectItem{
		{id: "1", title: "Item 1"},
	}
	m := newSelectModel("Test", items)
	m.quitting = true

	view := m.View()
	if view != "" {
		t.Errorf("View() should be empty when quitting, got: %q", view)
	}
}

// --- interactiveOrArg ---

func TestInteractiveOrArgWithArg(t *testing.T) {
	args := []string{"my-arg"}
	result, err := interactiveOrArg(nil, args, false, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "my-arg" {
		t.Errorf("result = %q, want %q", result, "my-arg")
	}
}

func TestInteractiveOrArgNoArgNoInteractive(t *testing.T) {
	_, err := interactiveOrArg(nil, nil, false, nil, "")
	if err == nil {
		t.Fatal("expected error when no arg and not interactive")
	}
	if !strings.Contains(err.Error(), "requires an argument") {
		t.Errorf("error should mention requiring an argument, got: %v", err)
	}
}

func TestInteractiveOrArgNonTTY(t *testing.T) {
	origInteractive := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = origInteractive }()

	_, err := interactiveOrArg(nil, nil, true, func() ([]selectItem, error) {
		return []selectItem{{id: "1", title: "Item"}}, nil
	}, "Test")
	if err == nil {
		t.Fatal("expected error in non-TTY")
	}
	if !strings.Contains(err.Error(), "non-TTY") {
		t.Errorf("error should mention non-TTY, got: %v", err)
	}
}

// --- --interactive flag audit ---

// interactiveCommands lists every show command that should support --interactive.
var interactiveCommands = [][]string{
	{"issue", "show"},
	{"epic", "show"},
	{"sprint", "show"},
	{"pipeline", "show"},
	{"workspace", "show"},
}

func TestInteractiveFlagRegistered(t *testing.T) {
	for _, cmdPath := range interactiveCommands {
		name := "zh " + strings.Join(cmdPath, " ")
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find(cmdPath)
			if err != nil {
				t.Fatalf("command not found: %v", err)
			}

			flag := cmd.Flags().Lookup("interactive")
			if flag == nil {
				t.Fatalf("--interactive flag not registered on %s", name)
			}
			if flag.DefValue != "false" {
				t.Errorf("--interactive default should be false, got %s", flag.DefValue)
			}
			if flag.Shorthand != "i" {
				t.Errorf("--interactive shorthand should be 'i', got %q", flag.Shorthand)
			}
		})
	}
}

func TestInteractiveHelpText(t *testing.T) {
	for _, cmdPath := range interactiveCommands {
		name := "zh " + strings.Join(cmdPath, " ")
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find(cmdPath)
			if err != nil {
				t.Fatalf("command not found: %v", err)
			}

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			t.Cleanup(func() {
				cmd.SetOut(nil)
				cmd.SetErr(nil)
			})
			cmd.Help()

			out := buf.String()
			if !strings.Contains(out, "--interactive") {
				t.Errorf("help output should mention --interactive, got: %s", out)
			}
		})
	}
}

// --- non-TTY fallback ---

func TestIssueShowInteractiveNonTTY(t *testing.T) {
	resetIssueFlags()
	issueShowInteractive = true

	origInteractive := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = origInteractive }()

	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show", "--interactive"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue show --interactive should error in non-TTY")
	}
	if !strings.Contains(err.Error(), "non-TTY") {
		t.Errorf("error should mention non-TTY, got: %v", err)
	}
}

func TestEpicShowInteractiveNonTTY(t *testing.T) {
	resetEpicFlags()
	epicShowInteractive = true

	origInteractive := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = origInteractive }()

	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "show", "--interactive"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic show --interactive should error in non-TTY")
	}
	if !strings.Contains(err.Error(), "non-TTY") {
		t.Errorf("error should mention non-TTY, got: %v", err)
	}
}

func TestSprintShowInteractiveNonTTY(t *testing.T) {
	resetSprintFlags()
	sprintShowInteractive = true

	origInteractive := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = origInteractive }()

	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"sprint", "show", "--interactive"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("sprint show --interactive should error in non-TTY")
	}
	if !strings.Contains(err.Error(), "non-TTY") {
		t.Errorf("error should mention non-TTY, got: %v", err)
	}
}

func TestPipelineShowInteractiveNonTTY(t *testing.T) {
	resetPipelineFlags()
	pipelineShowInteractive = true

	origInteractive := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = origInteractive }()

	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "show", "--interactive"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("pipeline show --interactive should error in non-TTY")
	}
	if !strings.Contains(err.Error(), "non-TTY") {
		t.Errorf("error should mention non-TTY, got: %v", err)
	}
}

func TestWorkspaceShowInteractiveNonTTY(t *testing.T) {
	resetWorkspaceFlags()
	workspaceShowInteractive = true

	origInteractive := isInteractive
	isInteractive = func() bool { return false }
	defer func() { isInteractive = origInteractive }()

	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"workspace", "show", "--interactive"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("workspace show --interactive should error in non-TTY")
	}
	if !strings.Contains(err.Error(), "non-TTY") {
		t.Errorf("error should mention non-TTY, got: %v", err)
	}
}

// --- show commands still work without --interactive ---

func TestIssueShowRequiresArgWithoutInteractive(t *testing.T) {
	resetIssueFlags()

	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"issue", "show"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("issue show without arg or --interactive should error")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Errorf("error should mention requirement, got: %v", err)
	}
}

func TestEpicShowRequiresArgWithoutInteractive(t *testing.T) {
	resetEpicFlags()

	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"epic", "show"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("epic show without arg or --interactive should error")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Errorf("error should mention requirement, got: %v", err)
	}
}

func TestPipelineShowRequiresArgWithoutInteractive(t *testing.T) {
	resetPipelineFlags()

	configDir := t.TempDir()
	cacheDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("ZH_API_KEY", "test-key")
	t.Setenv("ZH_WORKSPACE", "ws-123")
	t.Setenv("ZH_GITHUB_TOKEN", "")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pipeline", "show"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("pipeline show without arg or --interactive should error")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Errorf("error should mention requirement, got: %v", err)
	}
}
