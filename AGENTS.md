# Project conventions

`CLAUDE.md` is a symlink to this file. Keep technical artifacts in English.

## Architecture

Use [Go With The Domain](https://threedots.tech/go-with-the-domain/) as the conceptual architecture reference.
Keep this application self-contained and domain-focused; add only behavior required by the project.

- Domain: standard library only.
- Adapter root: repository contracts and domain types only.
- Application: adapter-root contracts, domain and common helpers; no concrete adapter imports.
- Transport ports: application and domain, with standard-library HTTP/templates/logging.
- SQLite adapter: adapter-root contracts, domain, bob, opt, dbmate and the SQLite driver.
- Service: assembles handlers; only `cmd/web` assembles infrastructure and the server.
- Common helpers: standard library, local trace context and zerolog.

Depguard enforces these boundaries. Repository interfaces belong in `internal/adapter/repository.go`.

## Behavior and tools

Methods receive `context.Context` first. Commands return errors; queries return data without modifying state.
The web handler generates new IDs. Preserve existing log fields, metric keys and typed error behavior.
Logger configuration lives in `internal/common/logger`, independent of the environment adapter; `cmd/web`
only invokes it with parsed values and the output writer.

The web UI swaps `#users-section`. Validation/conflict responses preserve submitted values; other errors target
`#flash` without exposing their cause. Templates and assets are embedded. Preserve required third-party notices.

Use `make test`, `make lint`, `make coverage`, `make format`, `make mock` and `make generate`.
Tools use `go tool -modfile=./tools/go.tool.mod`. Never edit generated mocks or `*.bob.go` files manually.
Migrations are embedded SQL with dbmate up/down sections. Optional seeds run transactionally in filename order;
new datasets must be repeatable without overwriting existing users. Defaults and environment variables are
documented in `env.dist`. Building needs CGO and a C compiler; Docker supplies the build environment.
