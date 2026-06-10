# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.
It is also auto-detected by CodeRabbit as code-review guidelines, so keep the conventions below accurate.

## Repository layout

This repository (`go-rss-reader`) is the application root. Holds `docker-compose.yaml`, `backend/`, `frontend/`, `.github/`.

- `backend/` — the Go module (`module rss_reader`, Go 1.25). **All Go commands must be run from this directory**, not the repo root. Contains `.golangci.yaml`, `sqlc.yaml`, `cmd/`, `internal/`.
- `frontend/` — frontend application (all of this code is implemented by AI agent).
- Design notes live **outside this repository**, in the parent working tree's `docs/` (e.g. `../docs/internal/arch.md` and `../docs/internal/directory.md` are the canonical architecture references; `../docs/internal/error_handling_current.md` describes the error strategy). They are not part of this repo and are not visible to repo-scoped tools (e.g. CodeRabbit) — the essential conventions are inlined below.
- The parent working tree also holds `../practice/` (scratch Go files) and a legacy `../rss_reader/`; both are outside this repo — ignore unless asked.

## Common commands

Run from the repository root (compose) or `backend/` (Go tooling):

- Run locally with live reload (from repo root): `docker compose up` (brings up Postgres 18 + app via `air`; reads `.env`).
- Run without Docker (from `backend/`): `go run ./cmd` (requires `DATABASE_URL` in env; see `.env`).
- Build (from `backend/`): `go build -buildvcs=false -o ./tmp/main ./cmd` (matches `.air.toml`).
- Test everything (from `backend/`): `go test ./...`.
- Single test: `go test ./internal/apperror -run TestWrapPreservesAppErrorMetadata -v`.
- Lint (from `backend/`): `golangci-lint run` (config in `.golangci.yaml` — enables `revive`, `errorlint`, `gocritic`, `gosec`, `errname`, `lll` with 120-char lines).
- Format (from `backend/`): `golangci-lint fmt` (uses the `formatters` section of `.golangci.yaml`; supports `--stdin` for editor integration).
- Regenerate sqlc code after touching `internal/infra/persistence/postgres/sql/*.sql`: `sqlc generate` (config in `sqlc.yaml`, output `internal/infra/persistence/postgres/generated/`). Never hand-edit the `generated/` files.

The server listens on `:8080`. Routes are under `/api/v1` (see `internal/infra/router/router.go`). Config comes from env vars only — `DATABASE_URL` is required; `LOG_LEVEL` (debug/info/warn/error) and `LOG_FORMAT` (text/json) are optional.

## Architecture

Paths below are relative to `backend/` (the Go module root).

Clean-architecture-inspired layering with dependencies pointing inward. Composition root is `internal/di/container.go` — it wires Config → pgxpool → repositories/gateway/txManager → interactors → handlers.

- `internal/domain/model` — entities (`Feed`, `Article`, `FetchStatus`, `FeedCursor`). IDs are UUIDv7 generated in constructors. No outward dependencies.
- `internal/domain/repository` — repository **interfaces** consumed by usecase. Concrete implementations live in `infra/persistence/postgres`.
- `internal/usecase` — `FeedInteractor`, `ArticleInteractor`. Depend on repository interfaces plus two ports: `RSSFetcher` (in `RSS_usecase.go`) and `TransactionManager` (in `transaction.go`). Interactors know nothing about HTTP or SQL.
- `internal/handler` — Echo handlers. Do input binding/validation (`go-playground/validator` via the package-level `requestValidator`) and delegate to usecase. They return `apperror.AppError` on failure — they never call `echo.NewHTTPError` or set status codes directly.
- `internal/infra/gateway/rss_gateway.go` — implements `usecase.RSSFetcher` using `gofeed` + a tuned `http.Client` (10s total, 5s TLS/response header). Extracts `ETag`/`Last-Modified` into `FeedCursor` and falls back to a sha256 of `title|publishedAt` when `GUID`/`Link` are missing (`resolveExternalID`).
- `internal/infra/persistence/postgres` — sqlc-generated queries wrapped by hand-written repositories. See transactions below.
- `internal/infra/middleware` — `NewGlobalErrorHandler`, `RequestIDContext`, `RequestLogger`. Wired in `cmd/main.go` after `middleware.RequestID()`.
- `internal/infra/router/router.go` — declares routes; the only place handlers are bound to paths.
- `internal/job` — background scheduler (`Scheduler` using `sync.WaitGroup.Go`) plus `DueFeedCollector`/`DueFeedFecher` skeletons for periodic feed refresh. `duefeed_fetcher.go` is currently a stub — the `Exec` body is missing.

### Transactions

`TransactionManager.WithTransaction(ctx, fn)` starts a pgx tx, stores it on `ctx` under an unexported key, and commits/rolls back around `fn`. Every postgres repo defines `querier(ctx)` that does:

```go
if tx := TxFromContext(ctx); tx != nil { return generated.New(tx) }
return generated.New(r.pool)
```

So call sites inside `txManager.WithTransaction(ctx, func(txCtx context.Context) error { ... })` automatically flow through the transaction, while calls outside go through the pool. Usecases must pass the `txCtx` received by the callback (not the outer `ctx`) down into repos.

### Error handling

Canonical type is `apperror.AppError` with `Code` ∈ {`not_found`, `invalid_argument`, `conflict`, `external_unavailable`, `internal`}, plus `Op` (operation breadcrumb) and `Err` (cause). Conventions:

- Every public method starts with `const op = "Pkg.Method"` and uses it when constructing/wrapping.
- `apperror.Wrap(err, op)` preserves the existing code and prepends the op — it does **not** reclassify. Use `apperror.NewXxx` only at the boundary where the classification first becomes known.
- Infra→apperror classification happens in `postgres/error.go`:`classifyDBError` (pgx `ErrNoRows` → not_found; PG error `23505` → conflict; else internal). Always route DB errors through `wrapAndLogDBError`, which also logs non-not-found failures.
- `gateway/rss_gateway.go` classifies network/parse failures as `external_unavailable` and non-2xx HTTP as `external_unavailable`.
- `middleware/error_handler.go` is the only place that converts `AppError`/`echo.HTTPError` to HTTP status + JSON. 5xx public messages are always replaced with `"internal server error"`.

### Logging

`applog.WithContext(ctx, logger)` attaches `request_id` (put on the context by `RequestIDContext` middleware) to any `*slog.Logger`. Always log via `applog.WithContext(...).XxxContext(ctx, ...)` rather than the raw `slog.Default()` so request correlation works. The logger is built from config in `infra/logger/logger.go` and installed globally in `main.go` before DI.

### Data model notes

- `articles` has `UNIQUE(feed_id, external_id)` and `SaveArticle` uses `ON CONFLICT ... DO NOTHING`. New articles from a refetch are idempotent.
- `feed_fetch_status` is upserted via `ON CONFLICT (feed_id) DO UPDATE` in `SaveFeedFetchStatus`.
- `GetDueFeedFetchStatuses` takes `now` and `limit` and is the entry point for the scheduled refresh loop. The generated param struct names the limit `Column2` (sqlc quirk with `$2::integer`).
- `sqlc.yaml` overrides make `pg_catalog.int4` → `int` (not `int32`) and nullable `text`/`timestamptz` → `*string`/`*time.Time`.
