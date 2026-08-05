# Agent Instructions

## Task Execution
- **Always prefer `just` recipes over low-level commands.** This project defines a `justfile` at the repo root; run `just --list` to see available recipes (test, lint, build, run, install, init, etc.) before falling back to raw `go`/`golangci-lint`/shell invocations.
- Only use the underlying low-level command when no matching `just` recipe exists, and prefer adding a new recipe to the `justfile` over repeatedly invoking the raw command.
