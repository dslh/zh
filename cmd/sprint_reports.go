package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dslh/zh/internal/api"
	"github.com/dslh/zh/internal/exitcode"
	"github.com/dslh/zh/internal/output"
	"github.com/dslh/zh/internal/resolve"
	"github.com/spf13/cobra"
)

// GraphQL queries for sprint reports

const sprintVelocityQuery = `query SprintVelocity($workspaceId: ID!, $sprintCount: Int!) {
  workspace(id: $workspaceId) {
    displayName
    averageSprintVelocity
    averageSprintVelocityWithDiff(skipDiff: false) {
      velocity
      difference
      sprintsCount
    }
    sprintConfig {
      kind
      period
      startDay
      endDay
      tzIdentifier
    }
    activeSprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
      sprintIssues(first: 0) {
        totalCount
      }
    }
    sprints(
      first: $sprintCount
      filters: { state: { eq: CLOSED } }
      orderBy: { field: END_AT, direction: DESC }
    ) {
      totalCount
      nodes {
        id
        name
        generatedName
        startAt
        endAt
        totalPoints
        completedPoints
        closedIssuesCount
        sprintIssues(first: 0) {
          totalCount
        }
      }
    }
  }
}`

const sprintScopeChangeQuery = `query SprintScopeChange($sprintId: ID!, $first: Int!, $after: String) {
  node(id: $sprintId) {
    ... on Sprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
      scopeChange(first: $first, after: $after) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          action
          effectiveAt
          estimateValue
          issue {
            id
            number
            title
            state
            estimate {
              value
            }
            repository {
              name
              ownerName
            }
          }
        }
      }
    }
  }
}`

const sprintReviewQuery = `query SprintReview($sprintId: ID!) {
  node(id: $sprintId) {
    ... on Sprint {
      id
      name
      generatedName
      state
      startAt
      endAt
      totalPoints
      completedPoints
      closedIssuesCount
      sprintReview {
        id
        title
        body
        state
        language
        lastGeneratedAt
        manuallyEdited
        createdAt
        updatedAt
        initiatedBy {
          id
          name
          githubUser {
            login
          }
        }
        sprintReviewFeatures(first: 50) {
          totalCount
          nodes {
            id
            title
            aiGeneratedIssues(first: 50) {
              totalCount
              nodes {
                id
                number
                title
                state
                estimate {
                  value
                }
                repository {
                  name
                  ownerName
                }
              }
            }
            manuallyAddedIssues(first: 50) {
              totalCount
              nodes {
                id
                number
                title
                state
                estimate {
                  value
                }
                repository {
                  name
                  ownerName
                }
              }
            }
          }
        }
        sprintReviewSchedules(first: 20) {
          totalCount
          nodes {
            id
            title
            startAt
            completedAt
          }
        }
        issuesClosedAfterSprintReview(first: 50) {
          totalCount
          nodes {
            id
            number
            title
            state
            estimate {
              value
            }
            repository {
              name
              ownerName
            }
          }
        }
      }
    }
  }
}`

// Response types

type velocitySprintEntry struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	GeneratedName   string  `json:"generatedName"`
	StartAt         string  `json:"startAt"`
	EndAt           string  `json:"endAt"`
	TotalPoints     float64 `json:"totalPoints"`
	CompletedPoints float64 `json:"completedPoints"`
	ClosedIssues    int     `json:"closedIssuesCount"`
	SprintIssues    struct {
		TotalCount int `json:"totalCount"`
	} `json:"sprintIssues"`
}

func (s *velocitySprintEntry) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.GeneratedName
}

type scopeChangeEvent struct {
	Action        string   `json:"action"`
	EffectiveAt   string   `json:"effectiveAt"`
	EstimateValue *float64 `json:"estimateValue"`
	Issue         struct {
		ID       string `json:"id"`
		Number   int    `json:"number"`
		Title    string `json:"title"`
		State    string `json:"state"`
		Estimate *struct {
			Value float64 `json:"value"`
		} `json:"estimate"`
		Repository struct {
			Name      string `json:"name"`
			OwnerName string `json:"ownerName"`
		} `json:"repository"`
	} `json:"issue"`
}

type reviewIssueNode struct {
	ID       string `json:"id"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Estimate *struct {
		Value float64 `json:"value"`
	} `json:"estimate"`
	Repository struct {
		Name      string `json:"name"`
		OwnerName string `json:"ownerName"`
	} `json:"repository"`
}

type reviewFeatureNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	AI    struct {
		TotalCount int               `json:"totalCount"`
		Nodes      []reviewIssueNode `json:"nodes"`
	} `json:"aiGeneratedIssues"`
	Manual struct {
		TotalCount int               `json:"totalCount"`
		Nodes      []reviewIssueNode `json:"nodes"`
	} `json:"manuallyAddedIssues"`
}

type reviewScheduleNode struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	StartAt     string  `json:"startAt"`
	CompletedAt *string `json:"completedAt"`
}

// Commands

var sprintVelocityCmd = &cobra.Command{
	Use:   "velocity",
	Short: "Show velocity trends for recent sprints",
	Long: `Show velocity trends for recent sprints, including points completed
per sprint and the workspace average velocity with trend.

Displays closed sprints and optionally the active sprint (in progress).
The average velocity is calculated by ZenHub over the last 3 closed sprints.

Examples:
  zh sprint velocity
  zh sprint velocity --sprints=10
  zh sprint velocity --no-active`,
	Args: cobra.NoArgs,
	RunE: runSprintVelocity,
}

var sprintScopeCmd = &cobra.Command{
	Use:   "scope [sprint]",
	Short: "Show scope change history for a sprint",
	Long: `Show scope change history for a sprint — issues added and removed over
the sprint's lifetime. Defaults to the active sprint.

Displays a chronological event log and a summary of initial scope,
mid-sprint additions, and removals.

The sprint can be specified as:
  - ZenHub ID
  - sprint name or unique name substring
  - relative reference: current, next, previous

Examples:
  zh sprint scope
  zh sprint scope previous
  zh sprint scope --summary`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSprintScope,
}

var sprintReviewCmd = &cobra.Command{
	Use:   "review [sprint]",
	Short: "View sprint review",
	Long: `View the sprint review associated with a sprint. Defaults to the active sprint.

Sprint reviews are AI-generated summaries of sprint accomplishments. Not all
sprints will have a review — one must be generated in ZenHub first.

The sprint can be specified as:
  - ZenHub ID
  - sprint name or unique name substring
  - relative reference: current, next, previous

Examples:
  zh sprint review
  zh sprint review previous
  zh sprint review --features`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSprintReview,
}

// Flag variables

var (
	velocitySprints  int
	velocityNoActive bool

	scopeSummary bool
	scopeLimit   int
	scopeAll     bool

	reviewFeatures   bool
	reviewSchedules  bool
	reviewLateCloses bool
	reviewRaw        bool
)

func init() {
	sprintVelocityCmd.Flags().IntVar(&velocitySprints, "sprints", 6, "Number of recent closed sprints to include")
	sprintVelocityCmd.Flags().BoolVar(&velocityNoActive, "no-active", false, "Exclude the active sprint from output")

	sprintScopeCmd.Flags().BoolVar(&scopeSummary, "summary", false, "Show only the net summary without the event log")
	output.AddPaginationFlags(sprintScopeCmd, &scopeLimit, &scopeAll)

	sprintReviewCmd.Flags().BoolVar(&reviewFeatures, "features", false, "Show feature breakdown with grouped issues")
	sprintReviewCmd.Flags().BoolVar(&reviewSchedules, "schedules", false, "Show associated review schedules")
	sprintReviewCmd.Flags().BoolVar(&reviewLateCloses, "late-closes", false, "Show issues closed after the review was generated")
	sprintReviewCmd.Flags().BoolVar(&reviewRaw, "raw", false, "Output review body as raw text without markdown rendering")

	sprintCmd.AddCommand(sprintVelocityCmd)
	sprintCmd.AddCommand(sprintScopeCmd)
	sprintCmd.AddCommand(sprintReviewCmd)
}

func resetSprintReportFlags() {
	velocitySprints = 6
	velocityNoActive = false

	scopeSummary = false
	scopeLimit = 100
	scopeAll = false

	reviewFeatures = false
	reviewSchedules = false
	reviewLateCloses = false
	reviewRaw = false
}

// ── sprint velocity ──────────────────────────────────────────────────────

func runSprintVelocity(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	data, err := client.Execute(sprintVelocityQuery, map[string]any{
		"workspaceId": cfg.Workspace,
		"sprintCount": velocitySprints,
	})
	if err != nil {
		return exitcode.General("fetching sprint velocity", err)
	}

	var resp struct {
		Workspace struct {
			DisplayName               string   `json:"displayName"`
			AverageSprintVelocity     *float64 `json:"averageSprintVelocity"`
			AverageSprintVelocityDiff *struct {
				Velocity     float64  `json:"velocity"`
				Difference   *float64 `json:"difference"`
				SprintsCount int      `json:"sprintsCount"`
			} `json:"averageSprintVelocityWithDiff"`
			SprintConfig *struct {
				Kind         string `json:"kind"`
				Period       int    `json:"period"`
				StartDay     string `json:"startDay"`
				EndDay       string `json:"endDay"`
				TzIdentifier string `json:"tzIdentifier"`
			} `json:"sprintConfig"`
			ActiveSprint *velocitySprintEntry `json:"activeSprint"`
			Sprints      struct {
				TotalCount int                   `json:"totalCount"`
				Nodes      []velocitySprintEntry `json:"nodes"`
			} `json:"sprints"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing sprint velocity response", err)
	}

	ws := resp.Workspace

	if output.IsJSON(outputFormat) {
		return output.JSON(w, map[string]any{
			"workspace":        ws.DisplayName,
			"averageVelocity":  ws.AverageSprintVelocity,
			"velocityWithDiff": ws.AverageSprintVelocityDiff,
			"sprintConfig":     ws.SprintConfig,
			"activeSprint":     ws.ActiveSprint,
			"closedSprints":    ws.Sprints.Nodes,
		})
	}

	if ws.SprintConfig == nil {
		fmt.Fprintln(w, "Sprints are not configured for this workspace.")
		return nil
	}

	// Header
	d := output.NewDetailWriter(w, "VELOCITY", ws.DisplayName)

	// Sprint cadence
	cadence := formatSprintCadence(ws.SprintConfig.Period, ws.SprintConfig.StartDay, ws.SprintConfig.EndDay)
	fields := []output.KeyValue{
		output.KV("Sprint cadence", cadence),
	}

	// Average velocity with trend
	if ws.AverageSprintVelocityDiff != nil {
		vd := ws.AverageSprintVelocityDiff
		velStr := formatEstimate(vd.Velocity) + " pts"
		velStr += fmt.Sprintf(" (last %d sprints", vd.SprintsCount)
		if vd.Difference != nil && *vd.Difference != 0 {
			if *vd.Difference > 0 {
				velStr += fmt.Sprintf(", trending %s", output.Green("+"+formatEstimate(*vd.Difference)))
			} else {
				velStr += fmt.Sprintf(", trending %s", output.Red(formatEstimate(*vd.Difference)))
			}
		}
		velStr += ")"
		fields = append(fields, output.KV("Avg velocity", velStr))
	} else if ws.AverageSprintVelocity != nil {
		fields = append(fields, output.KV("Avg velocity", formatEstimate(*ws.AverageSprintVelocity)+" pts"))
	}

	d.Fields(fields)

	// No sprints at all
	if len(ws.Sprints.Nodes) == 0 && (ws.ActiveSprint == nil || velocityNoActive) {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No closed sprints found.")
		return nil
	}

	// Sprint table
	lw := output.NewListWriter(w, "SPRINT", "DATES", "PTS DONE", "PTS TOTAL", "ISSUES", "VELOCITY")

	// Active sprint first (if included)
	if ws.ActiveSprint != nil && !velocityNoActive {
		s := ws.ActiveSprint
		lw.Row(
			output.Green("▶ "+s.DisplayName()),
			formatSprintDates(s.StartAt, s.EndAt),
			formatEstimate(s.CompletedPoints),
			formatEstimate(s.TotalPoints),
			fmt.Sprintf("%d/%d", s.ClosedIssues, s.SprintIssues.TotalCount),
			output.Dim("(in progress)"),
		)
	}

	// Closed sprints
	for _, s := range ws.Sprints.Nodes {
		lw.Row(
			"  "+s.DisplayName(),
			formatSprintDates(s.StartAt, s.EndAt),
			formatEstimate(s.CompletedPoints),
			formatEstimate(s.TotalPoints),
			fmt.Sprintf("%d/%d", s.ClosedIssues, s.SprintIssues.TotalCount),
			formatEstimate(s.CompletedPoints),
		)
	}

	// Footer with average
	footer := ""
	if ws.AverageSprintVelocityDiff != nil {
		footer = fmt.Sprintf("avg (last %d): %s", ws.AverageSprintVelocityDiff.SprintsCount, formatEstimate(ws.AverageSprintVelocityDiff.Velocity))
	}
	if footer != "" {
		lw.FlushWithFooter(footer)
	} else {
		lw.Flush()
	}

	return nil
}

func formatSprintCadence(period int, startDay, endDay string) string {
	unit := "week"
	if period > 1 {
		unit = fmt.Sprintf("%d-week", period)
	}
	if startDay != "" && endDay != "" {
		return fmt.Sprintf("%s (%s - %s)", unit, startDay, endDay)
	}
	return unit
}

// ── sprint scope ─────────────────────────────────────────────────────────

func runSprintScope(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve sprint
	identifier := "current"
	if len(args) > 0 {
		identifier = args[0]
	}

	resolved, err := resolve.Sprint(client, cfg.Workspace, identifier)
	if err != nil {
		return err
	}

	// Fetch scope changes with pagination
	limit := output.EffectiveLimit(scopeLimit, scopeAll)

	// For JSON output, respect the limit directly
	if output.IsJSON(outputFormat) {
		events, sprint, totalCount, err := fetchScopeChanges(client, resolved.ID, limit)
		if err != nil {
			return err
		}
		return output.JSON(w, map[string]any{
			"sprint": map[string]any{
				"id":              sprint.ID,
				"name":            sprint.DisplayName(),
				"state":           sprint.State,
				"startAt":         sprint.StartAt,
				"endAt":           sprint.EndAt,
				"totalPoints":     sprint.TotalPoints,
				"completedPoints": sprint.CompletedPoints,
			},
			"totalEvents": totalCount,
			"events":      events,
		})
	}

	// For human-readable output, fetch all events so the summary is accurate
	allEvents, sprint, totalCount, err := fetchScopeChanges(client, resolved.ID, 0)
	if err != nil {
		return err
	}

	// Display events are limited for the event log
	displayEvents := allEvents
	if limit > 0 && limit < len(allEvents) {
		displayEvents = allEvents[:limit]
	}

	// Header
	d := output.NewDetailWriter(w, "SCOPE CHANGES", sprint.DisplayName())
	fields := []output.KeyValue{
		output.KV("Dates", formatSprintDates(sprint.StartAt, sprint.EndAt)),
	}
	if sprint.TotalPoints > 0 {
		fields = append(fields, output.KV("Points", output.FormatProgress(int(sprint.CompletedPoints), int(sprint.TotalPoints))))
	}
	fields = append(fields, output.KV("Changes", formatScopeChangeSummaryLine(allEvents, totalCount)))
	d.Fields(fields)

	if totalCount == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No scope changes recorded for this sprint.")
		return nil
	}

	// Compute summary from all events (not limited)
	summary := computeScopeSummary(allEvents, sprint.StartAt)

	if !scopeSummary {
		// Event log table (uses limited display events)
		d.Section("EVENT LOG")
		lw := output.NewListWriter(w, "DATE", "ACTION", "PTS", "REPO", "#", "TITLE")
		for _, e := range displayEvents {
			t, _ := time.Parse(time.RFC3339, e.EffectiveAt)
			date := output.FormatDate(t)

			action := output.Green("+ added")
			if e.Action == "ISSUE_REMOVED" {
				action = output.Red("- removed")
			}

			pts := output.TableMissing
			if e.EstimateValue != nil {
				pts = formatEstimate(*e.EstimateValue)
			}

			title := e.Issue.Title
			if len(title) > 40 {
				title = title[:37] + "..."
			}

			lw.Row(
				date,
				action,
				pts,
				e.Issue.Repository.Name,
				fmt.Sprintf("#%d", e.Issue.Number),
				title,
			)
		}

		footer := fmt.Sprintf("Showing %d", len(displayEvents))
		if totalCount > len(displayEvents) {
			footer += fmt.Sprintf(" of %d", totalCount)
		}
		footer += " event(s)"
		lw.FlushWithFooter(footer)
	}

	// Summary section
	d.Section("SUMMARY")
	fmt.Fprintf(w, "Initial scope (at sprint start):  %s\n", formatScopeCount(summary.initialIssues, summary.initialPts))
	fmt.Fprintf(w, "Added mid-sprint:                 %s\n", formatScopeCount(summary.addedIssues, summary.addedPts))
	fmt.Fprintf(w, "Removed mid-sprint:               %s\n", formatScopeCount(summary.removedIssues, summary.removedPts))
	fmt.Fprintf(w, "Net scope change:                 %s\n", formatScopeCount(summary.addedIssues-summary.removedIssues, summary.addedPts-summary.removedPts))
	fmt.Fprintf(w, "Current scope:                    %s\n", formatScopeCount(
		summary.initialIssues+summary.addedIssues-summary.removedIssues,
		summary.initialPts+summary.addedPts-summary.removedPts,
	))

	return nil
}

type scopeChangeSummary struct {
	initialIssues int
	initialPts    float64
	addedIssues   int
	addedPts      float64
	removedIssues int
	removedPts    float64
}

func computeScopeSummary(events []scopeChangeEvent, startAt string) scopeChangeSummary {
	var s scopeChangeSummary
	sprintStart, _ := time.Parse(time.RFC3339, startAt)

	for _, e := range events {
		t, _ := time.Parse(time.RFC3339, e.EffectiveAt)
		pts := float64(0)
		if e.EstimateValue != nil {
			pts = *e.EstimateValue
		}

		isInitial := !t.After(sprintStart)

		switch e.Action {
		case "ISSUE_ADDED":
			if isInitial {
				s.initialIssues++
				s.initialPts += pts
			} else {
				s.addedIssues++
				s.addedPts += pts
			}
		case "ISSUE_REMOVED":
			if isInitial {
				// Removals before sprint start shouldn't normally happen,
				// but handle gracefully
				s.initialIssues--
				s.initialPts -= pts
			} else {
				s.removedIssues++
				s.removedPts += pts
			}
		}
	}

	return s
}

func formatScopeCount(issues int, pts float64) string {
	ptsStr := formatEstimate(pts)
	if issues == 1 {
		return fmt.Sprintf("1 issue, %s pts", ptsStr)
	}
	return fmt.Sprintf("%d issues, %s pts", issues, ptsStr)
}

func formatScopeChangeSummaryLine(events []scopeChangeEvent, totalCount int) string {
	added := 0
	removed := 0
	for _, e := range events {
		switch e.Action {
		case "ISSUE_ADDED":
			added++
		case "ISSUE_REMOVED":
			removed++
		}
	}
	return fmt.Sprintf("%d events (%d added, %d removed)", totalCount, added, removed)
}

type scopeSprintDetail struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	GeneratedName   string  `json:"generatedName"`
	State           string  `json:"state"`
	StartAt         string  `json:"startAt"`
	EndAt           string  `json:"endAt"`
	TotalPoints     float64 `json:"totalPoints"`
	CompletedPoints float64 `json:"completedPoints"`
	ClosedIssues    int     `json:"closedIssuesCount"`
}

func (s *scopeSprintDetail) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.GeneratedName
}

func fetchScopeChanges(client *api.Client, sprintID string, limit int) ([]scopeChangeEvent, *scopeSprintDetail, int, error) {
	var allEvents []scopeChangeEvent
	var cursor *string
	var sprint *scopeSprintDetail
	totalCount := 0
	pageSize := 100

	for {
		if limit > 0 {
			remaining := limit - len(allEvents)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		vars := map[string]any{
			"sprintId": sprintID,
			"first":    pageSize,
		}
		if cursor != nil {
			vars["after"] = *cursor
		}

		data, err := client.Execute(sprintScopeChangeQuery, vars)
		if err != nil {
			return nil, nil, 0, exitcode.General("fetching scope changes", err)
		}

		var resp struct {
			Node *struct {
				ID              string  `json:"id"`
				Name            string  `json:"name"`
				GeneratedName   string  `json:"generatedName"`
				State           string  `json:"state"`
				StartAt         string  `json:"startAt"`
				EndAt           string  `json:"endAt"`
				TotalPoints     float64 `json:"totalPoints"`
				CompletedPoints float64 `json:"completedPoints"`
				ClosedIssues    int     `json:"closedIssuesCount"`
				ScopeChange     struct {
					TotalCount int                `json:"totalCount"`
					PageInfo   pageInfoNode       `json:"pageInfo"`
					Nodes      []scopeChangeEvent `json:"nodes"`
				} `json:"scopeChange"`
			} `json:"node"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, nil, 0, exitcode.General("parsing scope changes response", err)
		}

		if resp.Node == nil {
			return nil, nil, 0, exitcode.NotFoundError(fmt.Sprintf("sprint %q not found", sprintID))
		}

		if sprint == nil {
			sprint = &scopeSprintDetail{
				ID:              resp.Node.ID,
				Name:            resp.Node.Name,
				GeneratedName:   resp.Node.GeneratedName,
				State:           resp.Node.State,
				StartAt:         resp.Node.StartAt,
				EndAt:           resp.Node.EndAt,
				TotalPoints:     resp.Node.TotalPoints,
				CompletedPoints: resp.Node.CompletedPoints,
				ClosedIssues:    resp.Node.ClosedIssues,
			}
		}

		totalCount = resp.Node.ScopeChange.TotalCount
		allEvents = append(allEvents, resp.Node.ScopeChange.Nodes...)

		if !resp.Node.ScopeChange.PageInfo.HasNextPage {
			break
		}
		if limit > 0 && len(allEvents) >= limit {
			break
		}

		cursor = &resp.Node.ScopeChange.PageInfo.EndCursor
	}

	return allEvents, sprint, totalCount, nil
}

// ── sprint review ────────────────────────────────────────────────────────

func runSprintReview(cmd *cobra.Command, args []string) error {
	cfg, err := requireWorkspace()
	if err != nil {
		return err
	}

	client := newClient(cfg, cmd)
	w := cmd.OutOrStdout()

	// Resolve sprint
	identifier := "current"
	if len(args) > 0 {
		identifier = args[0]
	}

	resolved, err := resolve.Sprint(client, cfg.Workspace, identifier)
	if err != nil {
		return err
	}

	// Fetch sprint review
	data, err := client.Execute(sprintReviewQuery, map[string]any{
		"sprintId": resolved.ID,
	})
	if err != nil {
		return exitcode.General("fetching sprint review", err)
	}

	var resp struct {
		Node *struct {
			ID              string  `json:"id"`
			Name            string  `json:"name"`
			GeneratedName   string  `json:"generatedName"`
			State           string  `json:"state"`
			StartAt         string  `json:"startAt"`
			EndAt           string  `json:"endAt"`
			TotalPoints     float64 `json:"totalPoints"`
			CompletedPoints float64 `json:"completedPoints"`
			ClosedIssues    int     `json:"closedIssuesCount"`
			SprintReview    *struct {
				ID              string `json:"id"`
				Title           string `json:"title"`
				Body            string `json:"body"`
				State           string `json:"state"`
				Language        string `json:"language"`
				LastGeneratedAt string `json:"lastGeneratedAt"`
				ManuallyEdited  bool   `json:"manuallyEdited"`
				CreatedAt       string `json:"createdAt"`
				UpdatedAt       string `json:"updatedAt"`
				InitiatedBy     *struct {
					ID         string `json:"id"`
					Name       string `json:"name"`
					GithubUser *struct {
						Login string `json:"login"`
					} `json:"githubUser"`
				} `json:"initiatedBy"`
				Features struct {
					TotalCount int                 `json:"totalCount"`
					Nodes      []reviewFeatureNode `json:"nodes"`
				} `json:"sprintReviewFeatures"`
				Schedules struct {
					TotalCount int                  `json:"totalCount"`
					Nodes      []reviewScheduleNode `json:"nodes"`
				} `json:"sprintReviewSchedules"`
				LateCloses struct {
					TotalCount int               `json:"totalCount"`
					Nodes      []reviewIssueNode `json:"nodes"`
				} `json:"issuesClosedAfterSprintReview"`
			} `json:"sprintReview"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return exitcode.General("parsing sprint review response", err)
	}

	if resp.Node == nil {
		return exitcode.NotFoundError(fmt.Sprintf("sprint %q not found", identifier))
	}

	sprint := resp.Node
	sprintName := sprint.Name
	if sprintName == "" {
		sprintName = sprint.GeneratedName
	}

	if output.IsJSON(outputFormat) {
		return output.JSON(w, sprint)
	}

	review := sprint.SprintReview

	// No review
	if review == nil {
		d := output.NewDetailWriter(w, "SPRINT REVIEW", sprintName)
		d.Fields([]output.KeyValue{
			output.KV("Dates", formatSprintDates(sprint.StartAt, sprint.EndAt)),
		})
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No review has been generated for this sprint.")
		return nil
	}

	// Review exists but not completed
	if review.State == "INITIAL" {
		d := output.NewDetailWriter(w, "SPRINT REVIEW", sprintName)
		d.Fields([]output.KeyValue{
			output.KV("Dates", formatSprintDates(sprint.StartAt, sprint.EndAt)),
			output.KV("State", output.Dim("INITIAL")),
		})
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Review has not been generated yet.")
		return nil
	}

	if review.State == "IN_PROGRESS" {
		d := output.NewDetailWriter(w, "SPRINT REVIEW", sprintName)
		d.Fields([]output.KeyValue{
			output.KV("Dates", formatSprintDates(sprint.StartAt, sprint.EndAt)),
			output.KV("State", output.Yellow("IN_PROGRESS")),
		})
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Review is currently being generated. Check back shortly.")
		return nil
	}

	// Completed review
	d := output.NewDetailWriter(w, "SPRINT REVIEW", sprintName)

	reviewFields := []output.KeyValue{
		output.KV("State", output.Green("COMPLETED")),
	}

	// Generated timestamp
	if review.LastGeneratedAt != "" {
		genStr := formatReviewTimestamp(review.LastGeneratedAt)
		if review.ManuallyEdited {
			genStr += " (manually edited)"
		}
		reviewFields = append(reviewFields, output.KV("Generated", genStr))
	}

	// Initiated by
	if review.InitiatedBy != nil {
		initiator := review.InitiatedBy.Name
		if review.InitiatedBy.GithubUser != nil && review.InitiatedBy.GithubUser.Login != "" {
			initiator = "@" + review.InitiatedBy.GithubUser.Login
		}
		if initiator != "" {
			reviewFields = append(reviewFields, output.KV("Initiated", initiator))
		}
	}

	d.Fields(reviewFields)

	// Progress section
	d.Section("PROGRESS")
	if sprint.TotalPoints > 0 {
		fmt.Fprintf(w, "Points:  %s\n", output.FormatProgress(int(sprint.CompletedPoints), int(sprint.TotalPoints)))
	}
	fmt.Fprintf(w, "Issues:  %d closed\n", sprint.ClosedIssues)

	// Review body
	d.Section("REVIEW")
	if review.Body != "" {
		if reviewRaw {
			fmt.Fprintln(w, review.Body)
		} else {
			_ = output.RenderMarkdown(w, review.Body, 80)
		}
	}

	// Features (optional)
	if reviewFeatures && review.Features.TotalCount > 0 {
		d.Section(fmt.Sprintf("FEATURES (%d)", review.Features.TotalCount))
		for _, f := range review.Features.Nodes {
			fmt.Fprintln(w)
			fmt.Fprintln(w, output.Bold(f.Title))

			// Collect and deduplicate issues
			issues := deduplicateReviewIssues(f.AI.Nodes, f.Manual.Nodes)
			if len(issues) == 0 {
				fmt.Fprintln(w, "  (no issues)")
				continue
			}
			for _, iss := range issues {
				est := output.TableMissing
				if iss.Estimate != nil {
					est = formatEstimate(iss.Estimate.Value) + " pts"
				}
				title := iss.Title
				if len(title) > 40 {
					title = title[:37] + "..."
				}
				fmt.Fprintf(w, "  %-16s #%-5d %-40s %s  %s\n",
					iss.Repository.Name, iss.Number, title, est, strings.ToLower(iss.State))
			}

			// Note if there are more manually added issues
			if f.Manual.TotalCount > len(f.Manual.Nodes) {
				fmt.Fprintf(w, "  + %d more manually added issue(s)\n", f.Manual.TotalCount-len(f.Manual.Nodes))
			}
		}
	}

	// Schedules (optional)
	if reviewSchedules && review.Schedules.TotalCount > 0 {
		d.Section(fmt.Sprintf("SCHEDULES (%d)", review.Schedules.TotalCount))
		lw := output.NewListWriter(w, "TITLE", "DATE", "STATUS")
		for _, s := range review.Schedules.Nodes {
			t, _ := time.Parse(time.RFC3339, s.StartAt)
			date := output.FormatDate(t)

			status := output.Yellow("pending")
			if s.CompletedAt != nil {
				status = output.Green("completed")
			}

			lw.Row(s.Title, date, status)
		}
		lw.Flush()
	}

	// Late closes (optional)
	if reviewLateCloses && review.LateCloses.TotalCount > 0 {
		d.Section(fmt.Sprintf("ISSUES CLOSED AFTER REVIEW (%d)", review.LateCloses.TotalCount))
		lw := output.NewListWriter(w, "REPO", "#", "TITLE", "EST")
		for _, iss := range review.LateCloses.Nodes {
			est := output.TableMissing
			if iss.Estimate != nil {
				est = formatEstimate(iss.Estimate.Value)
			}
			title := iss.Title
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			lw.Row(iss.Repository.Name, fmt.Sprintf("#%d", iss.Number), title, est)
		}
		lw.Flush()
	}

	// Hints for optional flags
	hints := []string{}
	if !reviewFeatures && review.Features.TotalCount > 0 {
		hints = append(hints, fmt.Sprintf("Use --features to see %d feature group(s).", review.Features.TotalCount))
	}
	if !reviewSchedules && review.Schedules.TotalCount > 0 {
		hints = append(hints, fmt.Sprintf("Use --schedules to see %d review schedule(s).", review.Schedules.TotalCount))
	}
	if !reviewLateCloses && review.LateCloses.TotalCount > 0 {
		hints = append(hints, fmt.Sprintf("Use --late-closes to see %d issue(s) closed after review.", review.LateCloses.TotalCount))
	}
	if len(hints) > 0 {
		fmt.Fprintln(w)
		for _, h := range hints {
			fmt.Fprintln(w, h)
		}
	}

	return nil
}

func formatReviewTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return output.FormatDate(t)
}

func deduplicateReviewIssues(ai, manual []reviewIssueNode) []reviewIssueNode {
	seen := make(map[string]bool)
	var result []reviewIssueNode
	for _, iss := range ai {
		if !seen[iss.ID] {
			seen[iss.ID] = true
			result = append(result, iss)
		}
	}
	for _, iss := range manual {
		if !seen[iss.ID] {
			seen[iss.ID] = true
			result = append(result, iss)
		}
	}
	return result
}
