We're building `zh`, a CLI tool for ZenHub. See SPEC.md for the full specification.

Read ROADMAP.md and find the first unchecked item. Starting from there, decide on a reasonable scope for this session — it could be a single item or a small group of related items, depending on their size and complexity. Err on the side of doing less well rather than more poorly. Complete your chosen scope and check off each item as you finish it.

A few guidelines:

- Consult the research/ directory for API details relevant to your task. Files are named by subcommand, e.g. research/issue/list.md for `zh issue list`.
- Follow existing patterns in the codebase. If you're unsure how something should be structured, look at how similar things have already been done.
- Prefer tasks that concern fixing or improving existing functionality, but otherwise don't skip ahead in the roadmap.

If you notice things that should be addressed but are outside your current scope — technical debt, missing functionality, workarounds that need revisiting — add them as new items to the appropriate phase in ROADMAP.md.

When you're finished, write a concise, bullet-point summary of the work done to a markdown file in the journal/ directory with a sequential name such as 001-scaffolding.md. Then check off completed items in ROADMAP.md and commit your changes.
