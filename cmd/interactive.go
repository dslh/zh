package cmd

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/spf13/cobra"
)

// selectItem represents a selectable entity in the interactive list.
type selectItem struct {
	id          string
	title       string
	description string
}

func (i selectItem) Title() string       { return i.title }
func (i selectItem) Description() string { return i.description }
func (i selectItem) FilterValue() string { return i.title + " " + i.description }

// selectResult holds the outcome of an interactive selection.
type selectResult struct {
	id        string
	title     string
	cancelled bool
}

// selectModel is the Bubble Tea model for interactive entity selection.
type selectModel struct {
	list     list.Model
	result   selectResult
	quitting bool
}

func newSelectModel(title string, items []selectItem) selectModel {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	height := min(len(items)*3+8, 25)
	l := list.New(listItems, delegate, 70, height)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)

	return selectModel{list: l}
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't intercept keys while filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			m.result.cancelled = true
			m.quitting = true
			return m, tea.Quit
		case "enter":
			selected, ok := m.list.SelectedItem().(selectItem)
			if !ok {
				return m, nil
			}
			m.result.id = selected.id
			m.result.title = selected.title
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Faint(true).Render("Press Enter to select, / to filter, Esc to cancel"))
	b.WriteString("\n")
	return b.String()
}

// runInteractiveSelect runs a Bubble Tea list selection and returns the
// selected item's ID and title. Returns an error if the user cancels or
// the terminal is not interactive.
func runInteractiveSelect(cmd *cobra.Command, title string, items []selectItem) (*selectResult, error) {
	if !isInteractive() {
		return nil, exitcode.Usage("--interactive requires a terminal — cannot run in non-TTY environment")
	}

	if len(items) == 0 {
		return nil, exitcode.NotFoundError("no items to select from")
	}

	m := newSelectModel(title, items)
	p := tea.NewProgram(m, tea.WithOutput(cmd.ErrOrStderr()))
	finalModel, err := p.Run()
	if err != nil {
		return nil, exitcode.General("interactive selection", err)
	}

	result := finalModel.(selectModel).result
	if result.cancelled {
		return nil, exitcode.General("selection cancelled", nil)
	}

	return &result, nil
}

// interactiveOrArg checks whether --interactive is set and returns the
// selected identifier, or falls back to the positional argument. Returns
// true if interactive mode was used.
func interactiveOrArg(cmd *cobra.Command, args []string, interactive bool, fetchItems func() ([]selectItem, error), listTitle string) (string, error) {
	if interactive {
		if !isInteractive() {
			return "", exitcode.Usage("--interactive requires a terminal — cannot run in non-TTY environment")
		}
		items, err := fetchItems()
		if err != nil {
			return "", err
		}
		result, err := runInteractiveSelect(cmd, listTitle, items)
		if err != nil {
			return "", err
		}
		return result.id, nil
	}

	if len(args) < 1 {
		return "", exitcode.Usage("requires an argument or --interactive flag")
	}
	return args[0], nil
}
