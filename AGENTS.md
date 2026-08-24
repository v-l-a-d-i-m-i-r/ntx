# Agent Instructions

## Project Structure
```
ntx/
├── cmd/
│   ├── ntx-cli/            # UUIDv7 helper tool
│   ├── ntx-dbus-forwarder/ # listens on D-Bus, forwards notifications to ntx-server over RPC
│   └── ntx-server/         # opens SQLite, runs migrations, serves RPC requests
├── internal/
│   ├── db/                 # connection setup (open.go), migration runner (db.go)
│   │   └── migrations/     # individual migration files
│   ├── dbusforwarder/      # D-Bus notification listener/forwarder logic
│   ├── notification/       # domain logic (validate, create, get) used by RPC handlers
│   ├── server/             # ntx-server wiring: RPC router (router.go), startup (server.go)
│   └── shared/
│       ├── dto/            # RPC request/response DTOs shared between forwarder and server
│       ├── frame/          # length-prefixed framing protocol used under the RPC layer
│       ├── log/            # structured logger construction (slog)
│       ├── rpc/            # JSON-RPC 2.0 request/response encoding and error codes
│       └── transport/      # transport interfaces
│           └── unixsock/   # Unix-socket implementation
└── tools/                  # supporting scripts/tooling, not application code
```
- When you add, remove, or move a directory or package, update this Project Structure section to match.

## Code Conventions
- When a type implements an interface, add a compile-time guard for it, for example `var _ Migration = m20260824083247487CreateNotifications{}` in `internal/db/migrations/20260824083247487_create_notifications.go`.
- Put the guard directly above the type definition.

## Task Execution
- **Always prefer `just` recipes over low-level commands.** This project defines a `justfile` at the repo root; run `just --list` to see available recipes (test, lint, build, run, install, init, etc.) before falling back to raw `go`/`golangci-lint`/shell invocations.
- Only use the underlying low-level command when no matching `just` recipe exists, and prefer adding a new recipe to the `justfile` over repeatedly invoking the raw command.
