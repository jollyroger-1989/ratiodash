# RatioDash — AGENTS.md

RatioDash is a self-hosted dashboard for tracking your upload/download ratio across multiple torrent sites. A Go REST API scrapes and stores per-site stats; a Vue 3 SPA presents them.

---

## Repository layout

```
ratiodash/
├── Procfile                  # overmind: starts api + web together
├── Makefile                  # convenience targets (install, build, dev-*)
├── backend/                  # Go API
│   ├── cmd/api/main.go       # FX application entrypoint
│   ├── pkg/
│   │   ├── config/           # env-based config (SERVER_ADDR, DATABASE_URL)
│   │   └── database/         # Goose migrations + GORM connection
│   ├── migrations/           # *.sql files embedded in the binary via go:embed
│   └── internal/
│       ├── domain/           # entities, repository/service interfaces
│       ├── repository/       # GORM implementations (see AGENTS.md inside)
│       ├── service/          # business logic
│       ├── scraper/          # scraper registry + per-site adapters
│       ├── scheduler/        # cron scheduler (robfig/cron)
│       ├── handler/          # HTTP handlers via Huma v2 (see AGENTS.md inside)
│       └── server/           # chi router, CORS, lifecycle hook
└── frontend/                 # Vue 3 SPA (see frontend layout below)
```

For a deeper breakdown of the backend internals, see [backend/AGENTS.md](backend/AGENTS.md).

---

## Backend stack

| Concern | Library |
|---|---|
| Dependency injection | `go.uber.org/fx` — every package exports a `Module` variable |
| HTTP router | `go-chi/chi/v5` with RequestID, Logger, Recoverer, CORS |
| OpenAPI / handlers | `danielgtaylor/huma/v2` — auto-generates OpenAPI 3.1; Swagger UI at `/docs` |
| ORM | `gorm.io/gorm` + `gorm.io/driver/sqlite` (CGo — requires `gcc`) |
| Migrations | `pressly/goose/v3` — SQL files embedded via `//go:embed *.sql`; run at startup |
| Scheduler | `robfig/cron/v3` — per-site cron expressions; scheduler implements `domain.Refresher` |

### FX module wiring order (`cmd/api/main.go`)

```
config → database → repository → scraper → service → scheduler → handler → server
```

Each package exports `var Module = fx.Options(…)`. To add a new concern, create a package, define `Module`, and append it here.

### Adding a new scraper

1. Copy `internal/scraper/generic.go`, rename the struct, implement `Key() string` and `Fetch(ctx, domain.Site) (*domain.SiteStats, error)`.
2. Add one annotated `fx.Provide` block to `internal/scraper/module.go`:

```go
fx.Provide(
    fx.Annotate(
        NewMyScraper,
        fx.As(new(domain.SiteScraper)),
        fx.ResultTags(`group:"scrapers"`),
    ),
),
```

3. Register a site in the UI with `scraper_key` set to the value returned by `Key()`.

### Adding a new entity (domain object)

1. Define the struct and repository/service interfaces in `internal/domain/{entity}.go`.
2. Implement the repository in `internal/repository/{entity}_repository.go` — follow [repository/AGENTS.md](backend/internal/repository/AGENTS.md).
3. Implement the service in `internal/service/{entity}_service.go`.
4. Write handlers in `internal/handler/{entity}_handler.go` — follow [handler/AGENTS.md](backend/internal/handler/AGENTS.md).
5. Add a migration in `migrations/000N_create_{entities}.sql`.
6. Wire constructors in the relevant `module.go` files.

---

## Frontend stack

| Concern | Library |
|---|---|
| Build tool | Vite 5 |
| Framework | Vue 3 (Composition API + `<script setup>`) |
| State management | Pinia |
| Routing | Vue Router 4 |
| HTTP client | Axios (wrapped in `src/services/api.ts`) |

Vite proxies `/api/*` → `http://localhost:8080` in dev.

---

## Running locally

```bash
# First-time setup
make install          # npm install in frontend/

# Start everything (requires overmind)
overmind start        # reads Procfile: api + web

# Or separately
make dev-backend      # Go API on :8080
make dev-frontend     # Vite on :5173
```

Environment variables (backend, all optional):

| Variable | Default |
|---|---|
| `SERVER_ADDR` | `:8080` |
| `DATABASE_URL` | `ratiodash.db` |

---

## Key URLs

| URL | Description |
|---|---|
| `http://localhost:5173` | Vue frontend |
| `http://localhost:8080/api/v1/sites` | Sites CRUD |
| `http://localhost:8080/api/v1/dashboard` | Latest stats per site |
| `http://localhost:8080/docs` | Swagger / OpenAPI UI |
| `http://localhost:8080/health` | Health check |
