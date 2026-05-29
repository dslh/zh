We've just finished building `zh`, a CLI tool for ZenHub. Now, we need to make sure that it works.

Read checklist.md and pick the next unchecked item from the list. Use the built binary at ./zh to run the subcommand and ensure that it's working properly. The local configuration is already set up with credentials for test GitHub and ZenHub accounts, the same ones you have configured for your ZenHub and GitHub GraphQL MCP tools. So, it is fine to run write commands, and you can use the MCP tools (or ./zh itself) to verify that the intended changes have taken effect.

Try out any flags or parameters supported by the command to ensure that they function as intended. Try out any different types of parameters that the command supports, and try different identifier types that are available to ensure they work.

If you encounter any bugs or errors, fix them and update any failing tests accordingly. Be sure that the full test suite and the linter pass. When you are done, write a summary of findings and fixes to a markdown file in journal/testing/ that corresponds to the subcommand that was tested (e.g. journal/testing/issue/list.md for `zh issue list`). Then, check off the item in checklist.md and commit any changes that were made along with the markdown report.

SPEC.md has some information about the test environment. You can create additional issues or PRs in the test repositories as needed. You do not need to test interactive mode on any commands.
