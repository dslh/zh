package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/gh"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL queries and mutations for issue estimate

const issueEstimateQuery = `query GetIssueForEstimate($repositoryGhId: Int!, $issueNumber: Int!) {
  issueByInfo(repositoryGhId: $repositoryGhId, issueNumber: $issueNumber) {
    id
    number
    title
    estimate {
      value
    }
    repository {
      name
      ownerName
      estimateSet {
        values
      }
    }
  }
}`

const issueEstimateByNodeQuery = `query GetIssueForEstimateByNode($id: ID!) {
  node(id: $id) {
    ... on Issue {
      id
      number
      title
      estimate {
        value
      }
      repository {
        name
        ownerName
        estimateSet {
          values
        }
      }
    }
  }
}`

const setEstimateMutation = `mutation SetEstimate($input: SetEstimateInput!) {
  setEstimate(input: $input) {
    issue {
      id
      number
      title
      estimate {
        value
      }
      repository {
        name
        ownerName
      }
    }
  }
}`

// resolvedEstimateIssue holds the info needed to set/clear an estimate.
type resolvedEstimateIssue struct {
	IssueID         string
	Number          int
	Title           string
	RepoName        string
	RepoOwner       string
	CurrentEstimate *float64
	ValidEstimates  []float64
}

func (r *resolvedEstimateIssue) Ref() string {
	return fmt.Sprintf("%s#%d", r.RepoName, r.Number)
}

// Commands

var issueEstimateCmd = &cobra.Command{
	Use:   "estimate <issue> [value]",
	Short: "Set or clear the estimate on an issue",
	Long: `Set or clear the estimate on an issue.

Provide a value to set the estimate. Omit the value to clear it.
The value must be one of the valid estimate values configured for the
repository (typically 1, 2, 3, 5, 8, 13, 21, 40).

Examples:
  zh issue estimate task-tracker#1 5
  zh issue estimate task-tracker#1          # clears the estimate
  zh issue estimate --repo=task-tracker 1 5`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runIssueEstimate,
}

var (
	issueEstimateDryRun bool
	issueEstimateRepo   string
)

func init() {
	issueEstimateCmd.Flags().BoolVar(&issueEstimateDryRun, "dry-run", false, "Show what would be changed without executing")
	issueEstimateCmd.Flags().StringVar(&issueEstimateRepo, "repo", "", "Repository context for bare issue numbers")

	issueCmd.AddCommand(issueEstimateCmd)
}

func resetIssueEstimateFlags() {
	issueEstimateDryRun = false
	issueEstimateRepo = ""
}

func runIssueEstimate(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Parse value argument (if present)
	var newValue *float64
	if len(args) == 2 {
		v, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return exitcode.Usage(fmt.Sprintf("invalid estimate value %q — must be a number", args[1]))
		}
		newValue = &v
	}

	// Resolve the issue and fetch current estimate + valid values
	ghClient := newGitHubClient(cfg, cmd)
	resolved, err := resolveForEstimate(client, cfg.Workspace, args[0], ghClient)
	if err != nil {
		return err
	}

	// Validate estimate value against valid set
	if newValue != nil && len(resolved.ValidEstimates) > 0 {
		if !isValidEstimate(*newValue, resolved.ValidEstimates) {
			validStr := formatEstimateList(resolved.ValidEstimates)
			return exitcode.Usage(fmt.Sprintf(
				"invalid estimate value %s — valid values are: %s",
				formatEstimate(*newValue), validStr,
			))
		}
	}

	// Dry run
	if issueEstimateDryRun {
		return renderEstimateDryRun(w, resolved, newValue)
	}

	// Execute mutation
	input := map[string]any{
		"issueId": resolved.IssueID,
	}
	if newValue != nil {
		input["value"] = *newValue
	} else {
		input["value"] = nil
	}

	data, err := client.Execute(setEstimateMutation, map[string]any{"input": input})
	if err != nil {
		return exitcode.General(fmt.Sprintf("setting estimate on %s", resolved.Ref()), err)
	}

	// Parse response for JSON output
	var resp struct {
		SetEstimate struct {
			Issue struct {
				ID       string `json:"id"`
				Number   int    `json:"number"`
				Title    string `json:"title"`
				Estimate *struct {
					Value float64 `json:"value"`
				} `json:"estimate"`
				Repository struct {
					Name      string `json:"name"`
					OwnerName string `json:"ownerName"`
				} `json:"repository"`
			} `json:"issue"`
		} `json:"setEstimate"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing estimate response", err)
	}

	// JSON output
	if output.IsJSON(outputFormat) {
		jsonResp := map[string]any{
			"issue": map[string]any{
				"id":         resp.SetEstimate.Issue.ID,
				"number":     resp.SetEstimate.Issue.Number,
				"repository": fmt.Sprintf("%s/%s", resp.SetEstimate.Issue.Repository.OwnerName, resp.SetEstimate.Issue.Repository.Name),
				"title":      resp.SetEstimate.Issue.Title,
				"estimate": map[string]any{
					"previous": formatEstimateJSON(resolved.CurrentEstimate),
					"current":  formatEstimateJSON(newValue),
				},
			},
		}
		return output.JSON(w, jsonResp)
	}

	// Render confirmation
	if newValue != nil {
		output.MutationSingle(w, output.Green(fmt.Sprintf(
			"Set estimate on %s to %s.",
			resolved.Ref(), formatEstimate(*newValue),
		)))
	} else {
		output.MutationSingle(w, output.Green(fmt.Sprintf(
			"Cleared estimate from %s.",
			resolved.Ref(),
		)))
	}

	return nil
}

// resolveForEstimate resolves an issue identifier and fetches current estimate + valid values.
func resolveForEstimate(client *api.Client, workspaceID, identifier string, ghClient *gh.Client) (*resolvedEstimateIssue, error) {
	parsed, parseErr := resolve.ParseIssueRef(identifier)

	// If it's a ZenHub ID, use the node query directly
	if parseErr == nil && parsed.ZenHubID != "" {
		return resolveEstimateByNode(client, parsed.ZenHubID)
	}

	// Resolve to get repo GH ID and issue number
	result, err := resolve.Issue(client, workspaceID, identifier, &resolve.IssueOptions{
		RepoFlag:     issueEstimateRepo,
		GitHubClient: ghClient,
	})
	if err != nil {
		return nil, err
	}

	return resolveEstimateByInfo(client, result.RepoGhID, result.Number)
}

func resolveEstimateByInfo(client *api.Client, repoGhID, issueNumber int) (*resolvedEstimateIssue, error) {
	data, err := client.Execute(issueEstimateQuery, map[string]any{
		"repositoryGhId": repoGhID,
		"issueNumber":    issueNumber,
	})
	if err != nil {
		return nil, exitcode.General("fetching issue for estimate", err)
	}

	var resp struct {
		IssueByInfo *struct {
			ID       string `json:"id"`
			Number   int    `json:"number"`
			Title    string `json:"title"`
			Estimate *struct {
				Value float64 `json:"value"`
			} `json:"estimate"`
			Repository struct {
				Name        string `json:"name"`
				OwnerName   string `json:"ownerName"`
				EstimateSet *struct {
					Values []float64 `json:"values"`
				} `json:"estimateSet"`
			} `json:"repository"`
		} `json:"issueByInfo"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing issue estimate response", err)
	}

	if resp.IssueByInfo == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue #%d not found", issueNumber))
	}

	var estVal *float64
	if resp.IssueByInfo.Estimate != nil {
		v := resp.IssueByInfo.Estimate.Value
		estVal = &v
	}
	var validEst []float64
	if resp.IssueByInfo.Repository.EstimateSet != nil {
		validEst = resp.IssueByInfo.Repository.EstimateSet.Values
	}

	return buildResolvedEstimate(resp.IssueByInfo.ID, resp.IssueByInfo.Number, resp.IssueByInfo.Title,
		resp.IssueByInfo.Repository.Name, resp.IssueByInfo.Repository.OwnerName,
		estVal, validEst), nil
}

func resolveEstimateByNode(client *api.Client, nodeID string) (*resolvedEstimateIssue, error) {
	data, err := client.Execute(issueEstimateByNodeQuery, map[string]any{
		"id": nodeID,
	})
	if err != nil {
		return nil, exitcode.General("fetching issue for estimate", err)
	}

	var resp struct {
		Node *struct {
			ID       string `json:"id"`
			Number   int    `json:"number"`
			Title    string `json:"title"`
			Estimate *struct {
				Value float64 `json:"value"`
			} `json:"estimate"`
			Repository struct {
				Name        string `json:"name"`
				OwnerName   string `json:"ownerName"`
				EstimateSet *struct {
					Values []float64 `json:"values"`
				} `json:"estimateSet"`
			} `json:"repository"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, exitcode.General("parsing issue estimate response", err)
	}

	if resp.Node == nil {
		return nil, exitcode.NotFoundError(fmt.Sprintf("issue %q not found", nodeID))
	}

	var estVal *float64
	if resp.Node.Estimate != nil {
		v := resp.Node.Estimate.Value
		estVal = &v
	}
	var validEst []float64
	if resp.Node.Repository.EstimateSet != nil {
		validEst = resp.Node.Repository.EstimateSet.Values
	}

	return buildResolvedEstimate(resp.Node.ID, resp.Node.Number, resp.Node.Title,
		resp.Node.Repository.Name, resp.Node.Repository.OwnerName,
		estVal, validEst), nil
}

func buildResolvedEstimate(id string, number int, title, repoName, repoOwner string, estimateValue *float64, validEstimates []float64) *resolvedEstimateIssue {
	return &resolvedEstimateIssue{
		IssueID:         id,
		Number:          number,
		Title:           title,
		RepoName:        repoName,
		RepoOwner:       repoOwner,
		CurrentEstimate: estimateValue,
		ValidEstimates:  validEstimates,
	}
}

func isValidEstimate(value float64, validValues []float64) bool {
	for _, v := range validValues {
		if v == value {
			return true
		}
	}
	return false
}

func formatEstimateList(values []float64) string {
	strs := make([]string, len(values))
	for i, v := range values {
		strs[i] = formatEstimate(v)
	}
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func formatEstimateJSON(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func renderEstimateDryRun(w writerFlusher, resolved *resolvedEstimateIssue, newValue *float64) error {
	var header string
	var ctx string

	if resolved.CurrentEstimate != nil {
		ctx = fmt.Sprintf("(currently: %s)", formatEstimate(*resolved.CurrentEstimate))
	} else {
		ctx = "(currently: none)"
	}

	if newValue != nil {
		header = fmt.Sprintf("Would set estimate on %s to %s", resolved.Ref(), formatEstimate(*newValue))
	} else {
		header = fmt.Sprintf("Would clear estimate from %s", resolved.Ref())
	}

	items := []output.MutationItem{
		{
			Ref:     resolved.Ref(),
			Title:   truncateTitle(resolved.Title),
			Context: ctx,
		},
	}

	output.MutationDryRun(w, header, items)
	return nil
}
