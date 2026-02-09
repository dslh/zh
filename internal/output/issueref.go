package output

import (
	"fmt"
	"strings"
)

// IssueRefFormatter formats issue references as short form (repo#number)
// or long form (owner/repo#number) depending on whether any repo names
// in the workspace are ambiguous (same name under different owners).
type IssueRefFormatter struct {
	ambiguous map[string]bool // repo names that appear under multiple owners
}

// NewIssueRefFormatter creates a formatter that knows which repo names are
// ambiguous. Pass all repo full names (owner/repo) in the workspace.
func NewIssueRefFormatter(repoFullNames []string) *IssueRefFormatter {
	// Count how many distinct owners each repo name has.
	owners := make(map[string]map[string]bool) // repo name → set of owners
	for _, full := range repoFullNames {
		parts := strings.SplitN(full, "/", 2)
		if len(parts) != 2 {
			continue
		}
		owner, repo := parts[0], parts[1]
		if owners[repo] == nil {
			owners[repo] = make(map[string]bool)
		}
		owners[repo][owner] = true
	}

	ambiguous := make(map[string]bool)
	for repo, ownerSet := range owners {
		if len(ownerSet) > 1 {
			ambiguous[repo] = true
		}
	}

	return &IssueRefFormatter{ambiguous: ambiguous}
}

// FormatRef formats an issue or PR reference. If the repo name is ambiguous
// within the workspace, returns "owner/repo#number"; otherwise "repo#number".
func (f *IssueRefFormatter) FormatRef(owner, repo string, number int) string {
	if f.ambiguous[repo] {
		return fmt.Sprintf("%s/%s#%d", owner, repo, number)
	}
	return fmt.Sprintf("%s#%d", repo, number)
}
