# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`gostrata` is a Cobra CLI that scaffolds Go Clean Architecture projects (`gostrata new`) and generates per-entity layers inside them (`gostrata make:module`, `make:entity`, `make:repo`, ...).

## Commands

```bash
make build        # ./bin/gostrata with version stamped via -ldflags (cmd.AppVersion)
make test         # go test -race ./...
make vet          # go vet ./...
make lint         # fails if gofmt -l reports anything
make smoke        # GOSTRATA_SMOKE=1: scaffold a project, add a module, go build it (needs network)
make snapshot     # local goreleaser multi-platform build

go test ./pkg/generator -run TestStackMatrixGeneratesValidGo -v   # single test
```

CI (`.github/workflows/ci.yml`) runs gofmt check, vet, race tests and the smoke test. Pushing a `v*` tag triggers GoReleaser and then dispatches a `goclean-release` event to the `skryfon-org/goclean-desktop` repo.

## Architecture

- `cmd/` — Cobra commands. `root.go` registers everything and holds the shared persistent flags (`--force`, `--dry-run`). `wizard.go` holds the `huh` arrow-key prompts used by `new` only when stdin is a TTY and the flag wasn't passed.
- `templates/` — embedded via `templates/fs.go` (`//go:embed all:project all:stubs`).
  - `templates/project/` is walked file-by-file by `generator.GenerateProject`; every file (not just `.tmpl`) is rendered with `text/template` against `ProjectOptions`, `.tmpl` is stripped from the output name, and **a file that renders to whitespace is skipped** (that's how `docker-compose.yml` disappears for SQLite). Router/database variants live inside the same template via `{{if eq .Router "chi"}}` / `.Database` branches, not separate files.
  - `templates/stubs/` are per-entity templates rendered against `StubData` by `pkg/generator/module.go` (`layerFiles` maps layer name → stub/destination; `handlerStub` picks `handler_chi.tmpl` vs `handler.tmpl`).
- `pkg/generator/` — all generation logic.
  - Every generated `.go` file passes through `formatGo` (`go/format`), so template whitespace doesn't matter but templates **must produce valid Go** or generation fails.
  - `gostrata new` writes `.gostrata.json` (database/router/orm). `make:*` reads it via `loadProjectConfig` (missing file ⇒ postgres+gin+gorm) so stubs, SQL types (`sqlTypeFor` in `fields.go`) and route snippets match the project's stack.
  - Auto-registration (`patch.go`): `make:module` inserts snippets above `// gostrata:*` marker comments in the generated `container.go` and `routes.go`. All-or-nothing per file, idempotent (skips if already present), re-gofmt'd; if markers are missing it prints manual steps instead. Renaming/removing a marker in the project templates breaks this.
  - Migrations are numbered by scanning `migrations/` for the next sequence (`nextMigrationNumber`).
  - `runner` records created/overwritten/skipped files for the end-of-run summary; it honours `Force` and `DryRun`.
- `pkg/strutil/` — case conversions and pluralization used to build `StubData` names.
- `pkg/ui/` — colored terminal output helpers.

## Adding a stack option

Supported values are the `Supported*` slices/constants in `pkg/generator/project.go`. `TestStackMatrixGeneratesValidGo` iterates every database × router combination, generates a project + module, and checks the Go parses and stack-specific output, so a new router/database must be handled in both project templates and stubs to pass. Per the README roadmap, more routers (Echo, Fiber) and ORMs need templates split into per-router/per-ORM layers first.
