# zh Subcommands Checklist

## zh board
- [x] `zh board` - Display all pipelines with their issues (default view)
- [x] `zh board --pipeline=<name>` - Filter to a single pipeline
- [x] `zh board --view=<name>` - Apply a saved view (filter preset)

## zh view
- [x] `zh view list` - List your saved views
- [x] `zh view show <name>` - Show the filters in a saved view
- [x] `zh view create <name>` - Create a saved view from filter flags
- [x] `zh view delete <name>` - Delete a saved view

## zh pipeline
- [x] `zh pipeline list` - List all pipelines in the workspace
- [x] `zh pipeline show <name>` - View details about a pipeline and the issues in it
- [x] `zh pipeline create <name>` - Create a new pipeline
- [x] `zh pipeline edit <name>` - Update a pipeline's name, position, or description
- [x] `zh pipeline delete <name> --into=<name>` - Delete a pipeline, moving its issues into the target pipeline
- [x] `zh pipeline alias <name> <alias>` - Set a shorthand name for the pipeline
- [x] `zh pipeline automations <name> - Display configured automations for the pipeline

## zh issue
- [x] `zh issue list` - List issues in the workspace
- [x] `zh issue show <issue>` - View issue details
- [x] `zh issue move <issue>... <pipeline>` - Move one or more issues to a pipeline
- [x] `zh issue estimate <issue> <value>` - Set the estimate on an issue
- [x] `zh issue close <issue>...` - Close one or more issues
- [x] `zh issue reopen <issue>... --pipeline=<name>` - Reopen issues into a pipeline
- [x] `zh issue connect <issue> <pr>` - Connect a PR to an issue
- [x] `zh issue disconnect <issue> <pr>` - Disconnect a PR from an issue
- [x] `zh issue block <blocker> <blocked>` - Mark blocker as blocking blocked
- [x] `zh issue priority <issue>... <priority>` - Set priority on issues
- [x] `zh issue label add <issue>... <label>...` - Add labels to issues
- [x] `zh issue label remove <issue>... <label>...` - Remove labels from issues

## zh epic
- [x] `zh epic list` - List epics in the workspace
- [x] `zh epic show <epic>` - View epic details
- [x] `zh epic create <title>` - Create an epic
- [x] `zh epic edit <epic>` - Update title/body
- [x] `zh epic delete <epic>` - Delete an epic
- [x] `zh epic set-state <epic> <state>` - Set state: open, todo, in_progress, closed
- [x] `zh epic set-dates <epic>` - Set start/end dates
- [x] `zh epic add <epic> <issue>...` - Add issues to an epic
- [x] `zh epic remove <epic> <issue>...` - Remove issues from an epic
- [x] `zh epic alias <epic> <alias>` - Set a shorthand name for the epic

## zh sprint
- [x] `zh sprint list` - List sprints (active, upcoming, recent)
- [x] `zh sprint show [sprint]` - View sprint details and issues
- [x] `zh sprint add <issue>...` - Add issues to a sprint
- [x] `zh sprint remove <issue>...` - Remove issues from a sprint
- [x] `zh sprint velocity` - Show velocity trends for recent sprints (points completed per sprint)
- [x] `zh sprint scope [sprint]` - Show scope change history (issues added/removed during sprint)
- [x] `zh sprint review [sprint]` - Show details of review associated with sprint

## zh workspace
- [x] `zh workspace list` - List available workspaces
- [x] `zh workspace show <name>` - Show current workspace details
- [x] `zh workspace switch <name>` - Switch the default workspace
- [x] `zh workspace repos` - List repos connected to the workspace
- [x] `zh workspace stats` - Detailed velocity trends, issue counts, activity metrics

## zh cache
- [x] `zh cache clear` - Clear the local cache
