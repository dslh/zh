package cmd

import "github.com/spf13/cobra"

// This file registers dynamic shell completions for all commands.
// It uses a single init() to wire up ValidArgsFunction on commands that
// take entity identifiers as positional args, and RegisterFlagCompletionFunc
// on flags that accept entity names.

func init() {
	// --- ValidArgsFunction for positional arguments ---

	// Pipeline commands: first arg is a pipeline name
	pipelineShowCmd.ValidArgsFunction = completePipelineNames
	pipelineEditCmd.ValidArgsFunction = completePipelineNames
	pipelineDeleteCmd.ValidArgsFunction = completePipelineNames
	pipelineAliasCmd.ValidArgsFunction = completePipelineNames
	pipelineAutomationsCmd.ValidArgsFunction = completePipelineNames

	// Sprint commands: first arg is a sprint name
	sprintShowCmd.ValidArgsFunction = completeSprintNames
	sprintScopeCmd.ValidArgsFunction = completeSprintNames
	sprintReviewCmd.ValidArgsFunction = completeSprintNames

	// Epic commands: first arg is an epic identifier
	epicShowCmd.ValidArgsFunction = completeEpicNames
	epicEditCmd.ValidArgsFunction = completeEpicNames
	epicDeleteCmd.ValidArgsFunction = completeEpicNames
	epicSetDatesCmd.ValidArgsFunction = completeEpicNames
	epicProgressCmd.ValidArgsFunction = completeEpicNames
	epicEstimateCmd.ValidArgsFunction = completeEpicNames
	epicAliasCmd.ValidArgsFunction = completeEpicNames
	epicAddCmd.ValidArgsFunction = completeEpicNames
	epicRemoveCmd.ValidArgsFunction = completeEpicNames
	epicAssigneeAddCmd.ValidArgsFunction = completeEpicNames
	epicAssigneeRemoveCmd.ValidArgsFunction = completeEpicNames
	epicLabelAddCmd.ValidArgsFunction = completeEpicNames
	epicLabelRemoveCmd.ValidArgsFunction = completeEpicNames
	epicKeyDateListCmd.ValidArgsFunction = completeEpicNames
	epicKeyDateAddCmd.ValidArgsFunction = completeEpicNames
	epicKeyDateRemoveCmd.ValidArgsFunction = completeEpicNames

	// epic set-state: first arg is epic, second is state
	epicSetStateCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeEpicNames(cmd, args, toComplete)
		}
		if len(args) == 1 {
			return completeEpicStates(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Workspace commands: first arg is a workspace name
	workspaceShowCmd.ValidArgsFunction = completeWorkspaceNames
	workspaceSwitchCmd.ValidArgsFunction = completeWorkspaceNames

	// --- RegisterFlagCompletionFunc for flags ---

	// Global output flag
	_ = rootCmd.RegisterFlagCompletionFunc("output", completeOutputFormats)

	// Pipeline flags
	registerFlagCompletion(boardCmd, "pipeline", completePipelineNames)
	registerFlagCompletion(issueListCmd, "pipeline", completePipelineNames)
	registerFlagCompletion(issueReopenCmd, "pipeline", completePipelineNames)
	registerFlagCompletion(pipelineDeleteCmd, "into", completePipelineNames)

	// Sprint flags
	registerFlagCompletion(issueListCmd, "sprint", completeSprintNames)
	registerFlagCompletion(sprintAddCmd, "sprint", completeSprintNames)
	registerFlagCompletion(sprintRemoveCmd, "sprint", completeSprintNames)

	// Epic flags
	registerFlagCompletion(issueListCmd, "epic", completeEpicNames)

	// Repo flags
	registerFlagCompletion(issueListCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueShowCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueCloseCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueMoveCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueEstimateCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueReopenCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueConnectCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueDisconnectCmd, "repo", completeRepoNames)
	registerFlagCompletion(issuePriorityCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueLabelAddCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueLabelRemoveCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueBlockCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueBlockersCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueBlockingCmd, "repo", completeRepoNames)
	registerFlagCompletion(issueActivityCmd, "repo", completeRepoNames)
	registerFlagCompletion(sprintAddCmd, "repo", completeRepoNames)
	registerFlagCompletion(sprintRemoveCmd, "repo", completeRepoNames)
	registerFlagCompletion(epicCreateCmd, "repo", completeRepoNames)
	registerFlagCompletion(epicAddCmd, "repo", completeRepoNames)
	registerFlagCompletion(epicRemoveCmd, "repo", completeRepoNames)

	// Position flags
	registerFlagCompletion(issueMoveCmd, "position", completePositionValues)
	registerFlagCompletion(issueReopenCmd, "position", completePositionValues)
}

// registerFlagCompletion is a helper that registers a flag completion function,
// silently ignoring errors (e.g. if the flag doesn't exist).
func registerFlagCompletion(cmd *cobra.Command, flag string, fn func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)) {
	_ = cmd.RegisterFlagCompletionFunc(flag, fn)
}
