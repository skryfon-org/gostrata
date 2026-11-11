# gostrata 🚀

**A modern, production-grade Go Clean Architecture scaffolding and code generator CLI tool.**

[![Go Reference](https://pkg.go.dev/badge/github.com/skryfon-org/gostrata.svg)](https://pkg.go.dev/github.com/skryfon-org/gostrata)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Website](https://img.shields.io/badge/website-goclean.cloud-22d3ee)](https://www.goclean.cloud)

📖 **[Documentation](https://www.goclean.cloud/docs/introduction/)** · 🌐 **[Website](https://www.goclean.cloud)** · 📝 **[Changelog](https://www.goclean.cloud/changelog/)** · 📦 **[Download the desktop installer](https://www.goclean.cloud/#download)**

Bakes in enterprise-tested architecture from real-world Go backend production systems (Gin, GORM, PostgreSQL, Viper, Zerolog, migrations, Docker).

---

## ⚡ Worldwide Installation

Anyone with Go 1.22+ (gostrata and generated projects target Go 1.23; Go will fetch that toolchain automatically) installed can install `gostrata` globally with one command:

```bash
go install github.com/skryfon-org/gostrata@latest
```

Ensure your `$(go env GOPATH)/bin` is in your system's `PATH`:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

---

## 🛠 Usage

### 1. Create a New Project

```bash
# Interactive mode (arrow-key menus for framework and database, prompts for module name)
gostrata new my-saas

# Or non-interactive with flags
gostrata new my-saas --module github.com/username/my-saas --db saas_db

# Pick the stack (defaults: gin + postgres)
gostrata new my-api --router chi --database sqlite
gostrata new my-shop --router gin --database mysql
```

| Flag | Values | Notes |
|------|--------|-------|
| `--router` | `gin` (default), `chi` | chi uses plain `net/http` handlers; request validation uses the same `binding:"..."` tags |
| `--database` | `postgres` (default), `mysql`, `sqlite` | SQLite is pure Go (no cgo) and skips `docker-compose.yml` |
| `--orm` | `gorm` (default) | Saved in `.gostrata.json`; more ORMs are planned (see Roadmap) |

In a terminal, any of `--router`, `--database` that you don't pass is asked with an arrow-key menu. Passing the flag skips its question, and piped or CI runs never prompt.

The choice is saved in `.gostrata.json`, so `make:*` generators emit matching handlers, routes, SQL types and migrations.

This bootstraps a complete Clean Architecture project in seconds:
- ✅ Scaffolds full Go Clean Architecture (Domain, Usecase, Delivery, Infrastructure)
- ✅ Auto-replaces all import paths with your Go module path
- ✅ Configures `.env` and `.env.example`
- ✅ Sets up `docker-compose.yml` (PostgreSQL)
- ✅ Configures `.air.toml` for live hot-reload
- ✅ Initializes Git repository (`git init`)
- ✅ Resolves and downloads Go dependencies (`go mod tidy`)

---

### 2. Scaffold Features (`make:*` generators)

Inside your project directory, generate complete Clean Architecture components instantly:

#### Generate a Complete Module (All 7 layers in 1 second)
```bash
gostrata make:module Product
```

This creates:
- `internal/domain/entity/product.go`
- `internal/types/product.go`
- `internal/domain/repository/product_repository.go`
- `internal/infrastructure/persistence/product_repository.go`
- `internal/usecase/interface/product_usecase.go`
- `internal/usecase/implementation/product_usecase.go`
- `internal/delivery/http/handlers/product_handler.go`
- `migrations/<NNNNNN>_create_products_table.up.sql`
- `migrations/<NNNNNN>_create_products_table.down.sql`

#### Define fields (`name:type`)
```bash
gostrata make:module Product name:string price:float stock:int active:bool
```
Fields flow into the entity, DTOs (create/update/response), usecase, handler and migration columns.
Types: `string`, `text`, `int`, `int64`, `float`, `bool`, `time`, `uuid` (default `string`).
Update requests use pointers, so omitted fields stay unchanged.

#### Auto-registration
`make:module` patches `container.go` and `routes.go` through the `// gostrata:*` marker comments in
projects created by `gostrata new`. It is idempotent. Use `--no-register` to skip it; for older projects
without markers it prints the manual steps instead.

#### Common flags
| Flag | Applies to | Effect |
|------|-----------|--------|
| `--force` | all generators, `new` | overwrite existing files (`new`: allow a non-empty directory) |
| `--dry-run` | all generators, `new` | list what would be written, change nothing |
| `--no-register` | `make:module` | don't patch container/routes |
| `--no-git`, `--no-tidy` | `new` | skip `git init` / `go mod tidy` |

Every run ends with a created / overwritten / skipped summary.

#### Check your environment
```bash
gostrata doctor   # verifies Go 1.22+ (1.23+ recommended), git and docker
```

#### Granular Generators
```bash
# Generate only an Entity
gostrata make:entity Order

# Generate Request, Input, and Response DTO types
gostrata make:types Order

# Generate Repository interface + GORM implementation
gostrata make:repo Order

# Generate Usecase interface + implementation
gostrata make:usecase Order

# Generate Gin HTTP Handler
gostrata make:handler Order

# Generate SQL migration files (up/down)
gostrata make:migration Order
```

---

## 📁 Architecture Overview

Every project generated by `gostrata` follows strict **Clean Architecture (Dependency Inversion)**:

```
├── cmd/
│   ├── api/                 # Server startup, graceful shutdown, logger, DI container
│   └── migrate/             # Database migration runner (up/down/version)
├── config/                  # Viper configuration loader (env + defaults)
├── internal/
│   ├── delivery/
│   │   └── http/
│   │       ├── handlers/    # Gin HTTP handlers
│   │       ├── middleware/  # Logger, Recovery, CORS, RequestID
│   │       ├── response/    # Standard JSON envelope (Success, Error, Paginated)
│   │       └── routes.go    # Route definitions & grouping
│   ├── domain/
│   │   ├── entity/          # Pure business entities (zero external dependencies)
│   │   ├── errors/          # Standard domain errors (ErrNotFound, ErrConflict, etc.)
│   │   ├── repository/      # Repository interface contracts
│   │   └── service/         # Domain service contracts
│   ├── infrastructure/
│   │   ├── container/       # Dependency Injection container (AppContainer)
│   │   ├── database/        # GORM PostgreSQL connection pool & fail-fast
│   │   ├── persistence/     # GORM implementations of repository interfaces
│   │   └── server/          # HTTP server lifecycle with graceful shutdown
│   ├── types/               # Request DTOs, Usecase Input/Query DTOs, and Response DTOs
│   └── usecase/
│       ├── interface/       # Usecase contracts / interactors
│       └── implementation/  # Concrete business logic implementations
├── pkg/
│   └── logger/              # Zerolog structured logging
└── migrations/              # PostgreSQL SQL migration files
```

---

## 🏃 Running Generated Projects

```bash
cd my-saas

# 1. Start local PostgreSQL
docker compose up -d

# 2. Run database migrations
make migrate

# 3. Start development server (hot-reload with Air)
make dev

# 4. Check API health
curl http://localhost:8080/health
```

---

## 🗺 Roadmap

Planned for a later version:

- **Frameworks:** Echo, Fiber
- **ORMs / data access:** sqlx, Bun, Ent (GORM stays the default)

These need the project templates split into per-router and per-ORM layers, so the wizard will list them once they are implemented.

---

## 🤝 Contributing

Contributions, issues and feature requests are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) for setup and the pull request flow, and the [Code of Conduct](CODE_OF_CONDUCT.md). To report a vulnerability, follow [SECURITY.md](SECURITY.md).

---

## 📄 License

MIT License.


---

## 🧪 Development & Releases

```bash
make test    # unit tests
make smoke   # scaffold a project, add a module, go build (needs network)
make build   # ./bin/gostrata with the version stamped in
```

CI (`.github/workflows/ci.yml`) runs format, vet, tests and the smoke test. Pushing a `v*` tag runs GoReleaser
(`.goreleaser.yaml`) to publish binaries. `go install github.com/skryfon-org/gostrata@vX.Y.Z` reports its
module version automatically.

---

## 📚 Learn more

- [Quick Start](https://www.goclean.cloud/docs/quick-start/): create your first project in minutes
- [Project Structure](https://www.goclean.cloud/docs/project-structure/): what gostrata generates and why
- [Available Commands](https://www.goclean.cloud/docs/available-commands/): every command and flag
- [Troubleshooting](https://www.goclean.cloud/docs/troubleshooting/): common problems and fixes
