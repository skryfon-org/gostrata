# Contributing to goclean

Thanks for helping. Everyone who takes part agrees to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

- **Bugs and ideas**: open an [issue](https://github.com/skryfon-org/goclean/issues/new/choose).
- **Security issues**: never open a public issue. See [SECURITY.md](SECURITY.md).
- **Bigger changes** (a new router, database or ORM, a new command, a change to the
  generated project layout): open an issue first so we can agree on scope before you
  write code.

Issues labelled `good first issue` are a good place to start.

## Development setup

You need Go (the version in `go.mod` or newer) and `make`.

```bash
git clone https://github.com/<you>/goclean && cd goclean
make build   # ./bin/goclean
make lint    # gofmt check
make vet
make test    # unit tests with the race detector
make smoke   # scaffold a project, add a module and compile it (needs network)
```

Run one test with `go test ./pkg/generator -run TestStackMatrixGeneratesValidGo -v`.

## How the code is laid out

- `cmd/`: Cobra commands and the interactive wizard.
- `templates/project/`: the files `goclean new` generates. Every file is rendered with
  `text/template`; router and database variants are `{{if}}` branches inside one file.
- `templates/stubs/`: per-entity templates used by the `make:*` commands.
- `pkg/generator/`: all generation logic. Every generated `.go` file is run through
  `go/format`, so a template must produce valid Go or generation fails.

Two things are easy to break:

- **Marker comments.** `make:module` registers new modules by inserting code above the
  `// goclean:*` comments in the generated `container.go` and `routes.go`. If you rename
  or remove a marker in the templates, update `pkg/generator/patch.go` too.
- **The stack matrix.** `TestStackMatrixGeneratesValidGo` generates a project and a
  module for every database and router combination. A template change must work for
  all of them, and a new stack option must be added to the `Supported*` values in
  `pkg/generator/project.go`, the project templates and the stubs.

## Pull requests

1. Fork the repository and create a branch from `main`.
2. Keep the change focused. Add or update tests for it.
3. Make sure `make lint vet test` pass (and `make smoke` if you touched templates).
4. Use a [Conventional Commits](https://www.conventionalcommits.org/) prefix in commit
   messages: `feat:`, `fix:`, `docs:`, `chore:`, `test:`. Release notes are built from
   them.
5. Open the pull request and fill in the template. A maintainer reviews it. CI must be
   green before it is merged.

By contributing, you agree that your contribution is licensed under the
[MIT License](LICENSE).

## Releases

Maintainers release by pushing a `vX.Y.Z` tag. GoReleaser builds the binaries and
publishes the GitHub release with checksums and build provenance.
