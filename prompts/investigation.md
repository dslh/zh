Take a look at SPEC.md. Today's task is to investigate ZenHub's API, to verify the feasibility of each subcommand and to work out the exact query or queries we'll need to run to implement them.

Read checklist.md and pick the first subcommand from the list that hasn't yet been checked off. Use your MCP connection to introspect ZenHub's API and determine which entities, fields, and mutations are available that could be used to implement the subcommand. Feel free to run queries if you think it might be helpful. Do not run any mutations.

You should then write a file to the research/ directory named after the subcommand you have investigated. research/board.md for `zh board`, research/view/list.md for `zh view list`, and so on. The file should include:
 - One or more examples of the query/queries/mutations that should be run against ZenHub's API to implement the subcommand.
 - A note about any information that would be useful to have cached to run the subcommand. Common stuff that doesn't change often such as workspace and pipeline names and IDs.
 - Any flags or parameters it might make sense for the subcommand to support based on what's available in the API.
 - Anything not available in the API that might instead be supplied by GitHub's API or CLI.
 - Anything not available at all that might limit the usefulness or viability of the subcommand.
 - Anything of interest adjacent to the relevant data available from ZenHub's API that might lend itself to a related subcommand not already listed in the spec.

Don't feel like you have to include a section for each of these concerns if there is nothing worth mentioning. You may also want to introspect or query GitHub's API using the tools available to you, if there's something vital you can't find in ZenHub's API. Again, don't run any migrations.

As a last step, check off the subcommand you have investigated in checklist.md. Thanks for your help.

If you need a workspace ID for any queries, try 69866ab95c14bf002977146b.
