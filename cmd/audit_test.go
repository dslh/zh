package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// allCommands lists every command and subcommand path in the CLI.
// Used to verify --help works on all commands.
var allCommands = [][]string{
	// Root and top-level
	{},
	{"version"},
	{"cache"},
	{"cache", "clear"},

	// Workspace
	{"workspace"},
	{"workspace", "list"},
	{"workspace", "show"},
	{"workspace", "switch"},
	{"workspace", "repos"},
	{"workspace", "stats"},

	// Pipeline
	{"pipeline"},
	{"pipeline", "list"},
	{"pipeline", "show"},
	{"pipeline", "create"},
	{"pipeline", "edit"},
	{"pipeline", "delete"},
	{"pipeline", "alias"},
	{"pipeline", "automations"},

	// Board
	{"board"},

	// Issue
	{"issue"},
	{"issue", "list"},
	{"issue", "show"},
	{"issue", "move"},
	{"issue", "estimate"},
	{"issue", "close"},
	{"issue", "reopen"},
	{"issue", "connect"},
	{"issue", "disconnect"},
	{"issue", "block"},
	{"issue", "blockers"},
	{"issue", "blocking"},
	{"issue", "priority"},
	{"issue", "label"},
	{"issue", "label", "add"},
	{"issue", "label", "remove"},
	{"issue", "activity"},

	// Epic
	{"epic"},
	{"epic", "list"},
	{"epic", "show"},
	{"epic", "create"},
	{"epic", "edit"},
	{"epic", "delete"},
	{"epic", "set-state"},
	{"epic", "set-dates"},
	{"epic", "add"},
	{"epic", "remove"},
	{"epic", "alias"},
	{"epic", "progress"},
	{"epic", "estimate"},
	{"epic", "assignee"},
	{"epic", "assignee", "add"},
	{"epic", "assignee", "remove"},
	{"epic", "label"},
	{"epic", "label", "add"},
	{"epic", "label", "remove"},
	{"epic", "key-date"},
	{"epic", "key-date", "list"},
	{"epic", "key-date", "add"},
	{"epic", "key-date", "remove"},

	// Sprint
	{"sprint"},
	{"sprint", "list"},
	{"sprint", "show"},
	{"sprint", "add"},
	{"sprint", "remove"},
	{"sprint", "velocity"},
	{"sprint", "scope"},
	{"sprint", "review"},

	// Utility
	{"label"},
	{"label", "list"},
	{"priority"},
	{"priority", "list"},
}

func TestAllHelpText(t *testing.T) {
	for _, cmdPath := range allCommands {
		name := "zh"
		if len(cmdPath) > 0 {
			name += " " + strings.Join(cmdPath, " ")
		}
		t.Run(name, func(t *testing.T) {
			// Find the target command without executing it
			var cmd = rootCmd
			if len(cmdPath) > 0 {
				var err error
				cmd, _, err = rootCmd.Find(cmdPath)
				if err != nil {
					t.Fatalf("command not found: %v", err)
				}
			}

			// Generate help text into a buffer, then reset
			// to avoid polluting other tests that share the global command tree
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			t.Cleanup(func() {
				cmd.SetOut(nil)
				cmd.SetErr(nil)
			})
			cmd.Help()

			out := buf.String()
			if out == "" {
				t.Fatal("help produced no output")
			}
			if !strings.Contains(out, "Usage:") {
				t.Errorf("help output should contain 'Usage:', got: %s", out)
			}
		})
	}
}

// dryRunCommands lists every command that should support --dry-run per SPEC.md.
var dryRunCommands = [][]string{
	// Pipeline mutations
	{"pipeline", "create"},
	{"pipeline", "edit"},
	{"pipeline", "delete"},

	// Issue mutations
	{"issue", "move"},
	{"issue", "estimate"},
	{"issue", "close"},
	{"issue", "reopen"},
	{"issue", "connect"},
	{"issue", "disconnect"},
	{"issue", "block"},
	{"issue", "priority"},
	{"issue", "label", "add"},
	{"issue", "label", "remove"},

	// Epic mutations
	{"epic", "create"},
	{"epic", "edit"},
	{"epic", "delete"},
	{"epic", "set-state"},
	{"epic", "set-dates"},
	{"epic", "add"},
	{"epic", "remove"},
	{"epic", "estimate"},
	{"epic", "assignee", "add"},
	{"epic", "assignee", "remove"},
	{"epic", "label", "add"},
	{"epic", "label", "remove"},
	{"epic", "key-date", "add"},
	{"epic", "key-date", "remove"},

	// Sprint mutations
	{"sprint", "add"},
	{"sprint", "remove"},
}

func TestDryRunFlagRegistered(t *testing.T) {
	for _, cmdPath := range dryRunCommands {
		name := "zh " + strings.Join(cmdPath, " ")
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find(cmdPath)
			if err != nil {
				t.Fatalf("command not found: %v", err)
			}

			flag := cmd.Flags().Lookup("dry-run")
			if flag == nil {
				t.Fatalf("--dry-run flag not registered on %s", name)
			}
			if flag.DefValue != "false" {
				t.Errorf("--dry-run default should be false, got %s", flag.DefValue)
			}
		})
	}
}

// commandsThatShouldNotHaveDryRun lists read-only commands to verify
// they do NOT have --dry-run.
var commandsThatShouldNotHaveDryRun = [][]string{
	{"workspace", "list"},
	{"workspace", "show"},
	{"workspace", "repos"},
	{"workspace", "stats"},
	{"pipeline", "list"},
	{"pipeline", "show"},
	{"pipeline", "automations"},
	{"issue", "list"},
	{"issue", "show"},
	{"issue", "blockers"},
	{"issue", "blocking"},
	{"issue", "activity"},
	{"epic", "list"},
	{"epic", "show"},
	{"epic", "progress"},
	{"sprint", "list"},
	{"sprint", "show"},
	{"sprint", "velocity"},
	{"sprint", "scope"},
	{"sprint", "review"},
	{"label", "list"},
	{"priority", "list"},
	{"board"},
	{"cache", "clear"},
}

func TestNoDryRunOnReadOnlyCommands(t *testing.T) {
	for _, cmdPath := range commandsThatShouldNotHaveDryRun {
		name := "zh " + strings.Join(cmdPath, " ")
		t.Run(name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find(cmdPath)
			if err != nil {
				t.Fatalf("command not found: %v", err)
			}

			flag := cmd.Flags().Lookup("dry-run")
			if flag != nil {
				t.Errorf("read-only command %s should not have --dry-run", name)
			}
		})
	}
}
