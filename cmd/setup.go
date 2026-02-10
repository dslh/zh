package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dslh/zh/internal/config"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/spf13/cobra"
)

// setupCmd is exposed as `zh setup` for manual re-configuration.
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure zh interactively",
	Long:  `Run the setup wizard to configure your ZenHub API key, default workspace, and GitHub access method.`,
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

// needsSetup reports whether the cold start wizard should run.
func needsSetup() bool {
	cfg, err := config.Load()
	if err != nil {
		return true
	}
	return cfg.APIKey == ""
}

// isInteractive reports whether the terminal is interactive (both stdin and
// stdout are TTYs). It's a variable so tests can override it.
var isInteractive = func() bool {
	for _, f := range []*os.File{os.Stdin, os.Stdout} {
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		if stat.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	return true
}

// runSetup implements the cold start wizard.
func runSetup(cmd *cobra.Command, args []string) error {
	if !isInteractive() {
		return exitcode.General("setup requires an interactive terminal — set ZH_API_KEY and ZH_WORKSPACE environment variables for non-interactive use", nil)
	}

	m := newSetupModel()
	p := tea.NewProgram(m, tea.WithOutput(cmd.ErrOrStderr()))
	finalModel, err := p.Run()
	if err != nil {
		return exitcode.General("setup wizard", err)
	}

	result := finalModel.(setupModel)
	if result.cancelled {
		fmt.Fprintln(cmd.OutOrStdout(), "Setup cancelled.")
		return nil
	}

	if result.err != nil {
		return result.err
	}

	// Write config
	cfg := &config.Config{
		APIKey:    result.apiKey,
		Workspace: result.workspaceID,
		GitHub: config.GitHubConfig{
			Method: result.githubMethod,
			Token:  result.githubToken,
		},
	}
	if err := config.Write(cfg); err != nil {
		return exitcode.General("saving config", err)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintln(w)
	fmt.Fprintln(w, output.Green("Configuration saved."))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Workspace: %s\n", result.workspaceName)
	fmt.Fprintf(w, "  GitHub:    %s\n", result.githubMethod)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run "+output.Bold("zh workspace list")+" to see all workspaces.")
	fmt.Fprintln(w, "Run "+output.Bold("zh --help")+" to see available commands.")
	return nil
}

// --- Bubble Tea model ---

type setupStep int

const (
	stepAPIKey setupStep = iota
	stepValidating
	stepSelectWorkspace
	stepGitHubMethod
	stepGitHubPAT
	stepValidatingGitHub
	stepDone
)

type setupModel struct {
	step setupStep
	err  error

	// Text inputs
	apiKeyInput textinput.Model
	patInput    textinput.Model
	activeInput *textinput.Model
	statusMsg   string

	// Workspace selection
	workspaces  []workspaceChoice
	wsListModel list.Model

	// GitHub method selection
	githubChoices []string
	githubCursor  int

	// Results
	apiKey        string
	workspaceID   string
	workspaceName string
	githubMethod  string
	githubToken   string
	cancelled     bool
}

type workspaceChoice struct {
	id      string
	name    string
	orgName string
}

func (w workspaceChoice) Title() string {
	if w.orgName != "" {
		return w.name + "  " + lipgloss.NewStyle().Faint(true).Render("("+w.orgName+")")
	}
	return w.name
}
func (w workspaceChoice) Description() string { return "" }
func (w workspaceChoice) FilterValue() string { return w.name + " " + w.orgName }

// Messages

type apiKeyValidatedMsg struct {
	workspaces []workspaceChoice
	err        error
}

type githubValidatedMsg struct {
	err error
}

func newSetupModel() setupModel {
	apiKey := textinput.New()
	apiKey.Placeholder = "zh_xxx..."
	apiKey.Focus()
	apiKey.CharLimit = 256
	apiKey.Width = 50

	pat := textinput.New()
	pat.Placeholder = "ghp_xxx..."
	pat.CharLimit = 256
	pat.Width = 50

	return setupModel{
		step:          stepAPIKey,
		apiKeyInput:   apiKey,
		patInput:      pat,
		activeInput:   &apiKey,
		githubChoices: []string{"gh", "pat", "none"},
		githubCursor:  0,
	}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	case apiKeyValidatedMsg:
		if msg.err != nil {
			m.step = stepAPIKey
			m.statusMsg = output.Red("Invalid API key: " + msg.err.Error())
			m.apiKeyInput.Focus()
			m.activeInput = &m.apiKeyInput
			return m, textinput.Blink
		}
		m.workspaces = msg.workspaces
		m.statusMsg = ""
		m.step = stepSelectWorkspace

		// Create list model
		items := make([]list.Item, len(m.workspaces))
		for i, ws := range m.workspaces {
			items[i] = ws
		}
		delegate := list.NewDefaultDelegate()
		delegate.ShowDescription = false
		m.wsListModel = list.New(items, delegate, 60, min(len(items)+8, 20))
		m.wsListModel.Title = "Select a workspace"
		m.wsListModel.SetShowStatusBar(false)
		m.wsListModel.SetShowHelp(true)
		return m, nil
	case githubValidatedMsg:
		if msg.err != nil {
			m.step = stepGitHubMethod
			m.statusMsg = output.Red("GitHub validation failed: " + msg.err.Error())
			return m, nil
		}
		m.step = stepDone
		return m, tea.Quit
	}

	switch m.step {
	case stepAPIKey:
		return m.updateAPIKey(msg)
	case stepSelectWorkspace:
		return m.updateWorkspaceSelect(msg)
	case stepGitHubMethod:
		return m.updateGitHubMethod(msg)
	case stepGitHubPAT:
		return m.updateGitHubPAT(msg)
	}

	return m, nil
}

func (m setupModel) updateAPIKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		value := strings.TrimSpace(m.apiKeyInput.Value())
		if value == "" {
			m.statusMsg = output.Red("API key is required")
			return m, nil
		}
		m.apiKey = value
		m.step = stepValidating
		m.statusMsg = "Validating API key..."
		return m, m.validateAPIKey
	}

	var cmd tea.Cmd
	m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	m.activeInput = &m.apiKeyInput
	return m, cmd
}

func (m setupModel) updateWorkspaceSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		selected, ok := m.wsListModel.SelectedItem().(workspaceChoice)
		if !ok {
			return m, nil
		}
		m.workspaceID = selected.id
		m.workspaceName = selected.name
		m.step = stepGitHubMethod
		m.statusMsg = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.wsListModel, cmd = m.wsListModel.Update(msg)
	return m, cmd
}

func (m setupModel) updateGitHubMethod(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.githubCursor > 0 {
				m.githubCursor--
			}
		case "down", "j":
			if m.githubCursor < len(m.githubChoices)-1 {
				m.githubCursor++
			}
		case "enter":
			m.githubMethod = m.githubChoices[m.githubCursor]
			m.statusMsg = ""
			switch m.githubMethod {
			case "gh":
				m.step = stepValidatingGitHub
				m.statusMsg = "Checking gh CLI..."
				return m, m.validateGhCLI
			case "pat":
				m.step = stepGitHubPAT
				m.patInput.Focus()
				m.activeInput = &m.patInput
				return m, textinput.Blink
			case "none":
				m.step = stepDone
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m setupModel) updateGitHubPAT(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "enter" {
		value := strings.TrimSpace(m.patInput.Value())
		if value == "" {
			m.statusMsg = output.Red("Token is required (or go back and select 'none')")
			return m, nil
		}
		m.githubToken = value
		m.step = stepDone
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.patInput, cmd = m.patInput.Update(msg)
	m.activeInput = &m.patInput
	return m, cmd
}

// Commands

func (m setupModel) validateAPIKey() tea.Msg {
	client := apiNewFunc(m.apiKey)
	workspaces, err := fetchAllWorkspaces(client)
	if err != nil {
		return apiKeyValidatedMsg{err: err}
	}

	var choices []workspaceChoice
	for _, ws := range workspaces {
		orgName := ""
		if ws.Organization != nil {
			orgName = ws.Organization.Name
		}
		choices = append(choices, workspaceChoice{
			id:      ws.ID,
			name:    ws.DisplayName,
			orgName: orgName,
		})
	}

	return apiKeyValidatedMsg{workspaces: choices}
}

// ghAuthCheckFunc is replaceable for testing.
var ghAuthCheckFunc = func() error {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run()
}

func (m setupModel) validateGhCLI() tea.Msg {
	if err := ghAuthCheckFunc(); err != nil {
		return githubValidatedMsg{err: fmt.Errorf("'gh auth status' failed — ensure the gh CLI is installed and authenticated")}
	}
	return githubValidatedMsg{}
}

// View

func (m setupModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true)
	b.WriteString(titleStyle.Render("zh setup"))
	b.WriteString("\n\n")

	switch m.step {
	case stepAPIKey:
		b.WriteString("Enter your ZenHub API key:\n")
		b.WriteString("(Find yours at https://app.zenhub.com/settings/tokens)\n\n")
		b.WriteString(m.apiKeyInput.View())
		b.WriteString("\n")
		if m.statusMsg != "" {
			b.WriteString("\n" + m.statusMsg + "\n")
		}
	case stepValidating:
		b.WriteString(m.statusMsg)
		b.WriteString("\n")
	case stepSelectWorkspace:
		b.WriteString(m.wsListModel.View())
		b.WriteString("\n")
	case stepGitHubMethod:
		b.WriteString("How should zh access GitHub?\n\n")
		for i, choice := range m.githubChoices {
			cursor := "  "
			if i == m.githubCursor {
				cursor = "> "
			}
			label := githubMethodLabel(choice)
			if i == m.githubCursor {
				b.WriteString(lipgloss.NewStyle().Bold(true).Render(cursor + label))
			} else {
				b.WriteString(cursor + label)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Faint(true).Render(githubMethodDescription(m.githubChoices[m.githubCursor])))
		b.WriteString("\n")
		if m.statusMsg != "" {
			b.WriteString("\n" + m.statusMsg + "\n")
		}
	case stepGitHubPAT:
		b.WriteString("Enter your GitHub personal access token:\n\n")
		b.WriteString(m.patInput.View())
		b.WriteString("\n")
		if m.statusMsg != "" {
			b.WriteString("\n" + m.statusMsg + "\n")
		}
	case stepValidatingGitHub:
		b.WriteString(m.statusMsg)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("Press Esc to cancel"))
	b.WriteString("\n")

	return b.String()
}

func githubMethodLabel(method string) string {
	switch method {
	case "gh":
		return "gh CLI (recommended)"
	case "pat":
		return "Personal access token"
	case "none":
		return "No GitHub access"
	}
	return method
}

func githubMethodDescription(method string) string {
	switch method {
	case "gh":
		return "Uses the gh CLI tool for GitHub access. Requires gh to be installed and authenticated."
	case "pat":
		return "Uses a GitHub personal access token. You'll need a token with repo scope."
	case "none":
		lines := []string{
			"Some features will not work without GitHub access:",
			"",
			"  - Legacy epic operations (edit, state, add/remove)",
			"  - Issue activity from GitHub (--github flag)",
			"  - Branch name resolution (--repo flag)",
			"  - PR review/merge/CI status in issue show",
			"  - Issue author, reactions, and participants",
			"  - Repo description, language, and stars in workspace repos",
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

// runRoot is the RunE for the bare `zh` command.
// On first run (no config), it launches the setup wizard.
// Otherwise, it shows help.
func runRoot(cmd *cobra.Command, args []string) error {
	if needsSetup() {
		if !isInteractive() {
			return cmd.Help()
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Welcome to zh! Let's get you set up.")
		return runSetup(cmd, nil)
	}
	return cmd.Help()
}

// setupPersistentPreRun is installed as PersistentPreRunE on the root command.
// It checks whether setup is needed and, if so, runs the wizard before any
// subcommand executes — unless the subcommand is one that doesn't need config
// (version, help, setup, completion).
func setupPersistentPreRun(cmd *cobra.Command, args []string) error {
	// Don't trigger setup for commands that don't need config.
	path := cmd.CommandPath() // e.g. "zh version", "zh cache clear"
	for _, skip := range []string{"zh version", "zh help", "zh setup", "zh completion", "zh cache"} {
		if path == skip || strings.HasPrefix(path, skip+" ") {
			return nil
		}
	}
	// Skip the bare root command (shows help).
	if path == "zh" {
		return nil
	}

	// Skip if API key is already set (via config or env).
	if !needsSetup() {
		return nil
	}

	// Non-interactive: just tell the user what to do.
	if !isInteractive() {
		return exitcode.Auth("no API key configured — set ZH_API_KEY or run 'zh setup' in a terminal", nil)
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "Welcome to zh! Let's get you set up.")
	return runSetup(cmd, nil)
}

// validateAPIKeyNonInteractive validates an API key by making a test API call.
// It's exported for use by tests of the model logic.
func validateAPIKeyNonInteractive(apiKey string) ([]workspaceChoice, error) {
	client := apiNewFunc(apiKey)
	data, err := client.Execute(listWorkspacesQuery, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Viewer struct {
			ZenhubOrganizations struct {
				Nodes []orgWithWorkspaces `json:"nodes"`
			} `json:"zenhubOrganizations"`
		} `json:"viewer"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	var choices []workspaceChoice
	for _, org := range resp.Viewer.ZenhubOrganizations.Nodes {
		for _, ws := range org.Workspaces.Nodes {
			choices = append(choices, workspaceChoice{
				id:      ws.ID,
				name:    ws.DisplayName,
				orgName: org.Name,
			})
		}
	}
	return choices, nil
}
