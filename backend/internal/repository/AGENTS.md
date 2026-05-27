# Repository layer — naming conventions

## Files & types

| Element | Convention | Example |
|---|---|---|
| File | `{entity}_repository.go` | `site_repository.go` |
| Struct (unexported) | `{entity}Repository` | `siteRepository` |
| Constructor | `New{Entity}Repository(db *gorm.DB) domain.{Entity}Repository` | `NewSiteRepository` |
| Domain interface | `{Entity}Repository` in `internal/domain/` | `SiteRepository`, `StatsRepository` |
| FX registration | `fx.Provide(New{Entity}Repository)` in `module.go` | |

## Method names

| Pattern | Signature |
|---|---|
| All rows | `FindAll() ([]T, error)` |
| By primary key | `FindByID(id uint) (*T, error)` |
| Filtered list | `FindBy{Field}(val) ([]T, error)` |
| Single latest row | `FindLatestBy{Field}(val) (*T, error)` |
| Latest row per group | `FindLatestAll() ([]T, error)` |
| Insert | `Create(entity *T) error` |
| Full update | `Update(entity *T) error` |
| Remove | `Delete(id uint) error` |

## Rules

- **Not-found sentinel**: `FindByID` and `FindLatestBy*` return `nil, nil` when the record does not exist. Callers convert that to a 404; it is not a repository-level error.
- **Never-nil slices**: list methods initialise to `[]T{}` when GORM returns nil so that JSON always encodes `[]` instead of `null`.
- **Error wrapping**: always add context — `fmt.Errorf("finding site %d: %w", id, err)`.
- **No business logic**: repositories only translate between domain types and the database. Validation and defaulting belong in the service layer.

---

## Testing conventions

### File & package

| Element | Convention | Example |
|---|---|---|
| File | `{entity}_repository_test.go` | `tracker_repository_test.go` |
| Package | `repository_test` (black-box) | — |

Use the external `repository_test` package so tests can only exercise the public `domain.{Entity}Repository` interface, not internal struct fields.

### DB setup

Call `testutil.NewDB(t)` once per test or sub-test. Each call returns an isolated in-memory SQLite with all migrations applied. Never share a `*gorm.DB` across parallel tests.

```go
func TestTrackerRepository_FindAll(t *testing.T) {
    db  := testutil.NewDB(t)
    repo := repository.NewTrackerRepository(db)
    // ...
}
```

### Structure

- **One `Test*` function per repository method** — keep each function focused on a single method (e.g. `TestTrackerRepository_Create`, `TestTrackerRepository_FindByID`).
- **Sub-tests for cases** — use `t.Run("case name", ...)` for happy path, not-found, duplicate, etc.
- **Table-driven only when inputs vary** — prefer explicit sub-tests over tables when side-effects accumulate across rows.

### Assertions

Use `github.com/stretchr/testify/assert` for non-fatal checks and `require` for fatal ones (DB setup, first error that makes subsequent assertions meaningless).

```go
require.NoError(t, repo.Create(&tracker))
assert.Equal(t, "My Tracker", found.Name)
assert.Nil(t, missing) // not-found sentinel
```

### Fixture helpers

Define unexported helpers inside the test file to build minimal valid entities:

```go
func newTracker(name string) *domain.Tracker {
    return &domain.Tracker{
        Name:       name,
        ScraperKey: "generic",
        CronExpr:   "@hourly",
    }
}
```

### What to test

| Scenario | Required |
|---|---|
| Happy path (insert + read back) | yes |
| Not-found returns `nil, nil` | yes for `FindByID` / `FindLatestBy*` |
| Duplicate / constraint violation returns an error | yes |
| List returns `[]T{}` (not `nil`) when empty | yes |
| `RowsAffected == 0` error path (e.g. `Delete` on missing row) | yes |
| Database failure returns an error | **yes — every method must have an error case** |
| Business logic / HTTP concerns | **no** — belongs in service/handler tests |

### Error cases

Every repository method must have at least one sub-test that verifies an error is returned when the database is unavailable. Force this by closing the underlying `*sql.DB` before calling the method:

```go
t.Run("returns error on database failure", func(t *testing.T) {
    db := testutil.NewDB(t)
    sqlDB, err := db.DB()
    require.NoError(t, err)
    require.NoError(t, sqlDB.Close())
    repo := repository.NewTrackerRepository(db)

    _, err = repo.FindAll()

    assert.Error(t, err)
})
```

### Coverage requirement

Tests must achieve **100% statement coverage** of the repository implementation file. Run with:

```sh
go test ./internal/repository/... -coverprofile=cover.out && go tool cover -func=cover.out
```

All uncovered lines are a test gap that must be addressed before merging.
