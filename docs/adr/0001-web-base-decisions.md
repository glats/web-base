# ADR 0001: Domain-focused Go web application

Status: implemented

## Context

The application needs a small user maintainer with persistent storage and server-rendered interactions,
without coupling domain rules to HTTP or a database implementation.

## Decisions

1. Keep a single Go module. Domain entities contain validation and typed errors, not persistence interfaces.
2. Put repository contracts in the adapter root; isolate concrete implementations in subpackages.
3. Separate commands and queries, assemble handlers in `service.NewApp`, and wire infrastructure in `cmd/web`.
4. Use bob v0.48.0 and dbmate v2.33.0 with SQLite. Embed migrations and apply pending versions at startup.
5. Make demo seeds opt-in and transactional. Repeated seed runs must neither duplicate nor overwrite users.
6. Use standard-library routing, HTML templates and embedding. Vendor htmx 4.0.0 and Pico.css 2.1.1 with their
   required notices, avoiding a frontend build pipeline and CDN dependencies.
7. Keep one section-swap target; render validation errors with submitted values and retarget unexpected errors
   to a flash region. Do not expose internal failure causes to users.
8. Protect writes with CrossOriginProtection and a 1 MiB body limit. Generate IDs before invoking commands.
9. Configure logging in `internal/common/logger`; `cmd/web` only calls that helper at startup. Apply LOG_LEVEL
   and LOG_PRETTY consistently to the server and handler-observation logs without a config-adapter dependency.
10. Package the CGO binary in a compatible non-root container. SQLite uses a writable persistent volume;
    no separate database service or Compose stack is needed.

## Consequences

Building requires CGO and a C compiler. Generated SQL code must be regenerated after schema changes, and its
constraint identifiers can affect error mapping. Startup migrations are simple for one process; deployments with
multiple replicas require a coordinated migration step. The UI requires JavaScript. Runtime tests can start the
real HTTP server using `run(ctx, cfg, ready)` without invoking main.

## Reference

[Go With The Domain](https://threedots.tech/go-with-the-domain/) is the conceptual architecture reference.
Third-party code licenses and generated-code notices are recorded separately and are not architecture sources.
