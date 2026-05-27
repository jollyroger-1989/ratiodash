# Handler layer — naming conventions

## Files & types

| Element | Convention | Example |
|---|---|---|
| File | `{entity}_handler.go` | `site_handler.go` |
| Struct (exported) | `{Entity}Handler` | `SiteHandler`, `StatsHandler` |
| Constructor | `New{Entity}Handler(deps...) *{Entity}Handler` | `NewSiteHandler` |
| Route registrar | `Register{Entity}Routes(api huma.API, h *{Entity}Handler)` | `RegisterSiteRoutes` |
| FX registration | `fx.Provide(New{Entity}Handler)` + `fx.Invoke(Register{Entity}Routes)` in `module.go` | |

## I/O type names

Each Huma operation gets its own pair of input/output structs in the same file.

| Pattern | Name |
|---|---|
| List all | `List{Entities}Output` |
| Get one | `Get{Entity}Input` / `Get{Entity}Output` |
| Create | `Create{Entity}Input` / `Create{Entity}Output` |
| Update (partial) | `Update{Entity}Input` / `Update{Entity}Output` |
| Delete | `Delete{Entity}Input` |
| Custom action | `{Action}{Entity}Input` / `{Action}{Entity}Output` |

- Path parameters go on the **Input** struct with `path:"param"` tags.
- Query parameters go on the **Input** struct with `query:"param"` tags.
- Request body is a nested `Body` field typed to the matching domain input struct.
- Response body is a nested `Body` field typed to the domain model or a slice of it.

## Handler methods

- Named after the action: `ListSites`, `GetSite`, `CreateSite`, `UpdateSite`, `DeleteSite`, `RefreshSite`.
- Signature: `func (h *{Entity}Handler) {Action}(ctx context.Context, input *{Action}{Entity}Input) (*{Action}{Entity}Output, error)`
- No-body responses (204): return `(*struct{}, error)` and set `DefaultStatus: http.StatusNoContent` on the operation.

## Error responses

| Situation | Helper |
|---|---|
| Record not found | `huma.Error404NotFound("…")` |
| Bad input / validation | `huma.Error422UnprocessableEntity("…")` |
| Unexpected server error | `huma.Error500InternalServerError("…")` |

Never leak raw error messages to the client; log them server-side and return a generic message.

## Operation IDs & tags

- `OperationID`: kebab-case action + entity — `"list-sites"`, `"create-site"`, `"refresh-site"`.
- `Tags`: one tag matching the entity name in lower-case plural — `[]string{"sites"}`.
- All routes are prefixed `/api/v1/`.

## Rules

- Handlers contain **no business logic** — delegate everything to domain service interfaces.
- Inject domain interfaces (not concrete types) so handlers remain testable in isolation.
- Call `refresher.Schedule` / `refresher.Unschedule` on create / update / delete when the entity affects the scheduler.

---

## Testing handlers

Handler tests are **integration tests**: they wire real repositories and services against an in-memory SQLite database, but mock external I/O (scrapers and the scheduler).

### Files & packages

| Element | Convention | Example |
|---|---|---|
| File | `{entity}_handler_test.go` | `tracker_handler_test.go` |
| Package | `package handler_test` (external black-box) | |
| Test func | `Test{Entity}Handler_{Action}` | `TestTrackerHandler_Create` |
| Subtest | `t.Run("…descriptive phrase…", …)` | `t.Run("returns 404 when not found", …)` |

### Wiring

Each test file declares a private `setup` helper that:
1. Calls `testutil.NewDB(t)` for an isolated, migrated SQLite database.
2. Creates real repositories and services from that DB.
3. Creates mock stubs for `domain.RefreshService` and `domain.Refresher` (scheduler) with `mocks.NewMock*`.
4. Creates the handler with `New{Entity}Handler(…)`.
5. Calls `testutil.NewAPI(t)` and registers routes with `Register{Entity}Routes`.

```go
type trackerTestEnv struct {
    api     humatest.TestAPI
    refresh *mocks.MockRefreshService
    sched   *mocks.MockRefresher
}

func setupTrackerHandler(t *testing.T) trackerTestEnv {
    t.Helper()
    db      := testutil.NewDB(t)
    repo    := repository.NewTrackerRepository(db)
    svc     := service.NewTrackerService(repo)
    refresh := mocks.NewMockRefreshService(t)
    sched   := mocks.NewMockRefresher(t)
    h       := NewTrackerHandler(svc, refresh, sched)
    api     := testutil.NewAPI(t)
    RegisterTrackerRoutes(api, h)
    return trackerTestEnv{api: api, refresh: refresh, sched: sched}
}
```

Handlers that only depend on a service with no external I/O (e.g. `StatsHandler`, `ScraperHandler`) do not need mocks — wire the real service directly.

### Making requests

Use `env.api.Do` with a method, path, optional headers, and an optional body reader.

```go
resp := env.api.Do(http.MethodGet, "/api/v1/trackers", nil, nil)
assert.Equal(t, http.StatusOK, resp.Code)

body := strings.NewReader(`{"name":"Alpha","scraper_key":"generic"}`)
resp  = env.api.Do(http.MethodPost, "/api/v1/trackers",
    http.Header{"Content-Type": []string{"application/json"}}, body)
assert.Equal(t, http.StatusOK, resp.Code)
```

### Asserting responses

Decode the response body into a typed struct. Use `require` for structural assertions (decode errors, status code) and `assert` for field-level checks.

```go
var out struct {
    ID   uint   `json:"id"`
    Name string `json:"name"`
}
require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
assert.Equal(t, "Alpha", out.Name)
```

For error responses, assert the Huma error envelope:

```go
var errBody struct {
    Status int    `json:"status"`
    Title  string `json:"title"`
}
require.NoError(t, json.NewDecoder(resp.Body).Decode(&errBody))
assert.Equal(t, http.StatusNotFound, errBody.Status)
```

### Mock expectations

Set expectations before issuing the request. Use `.EXPECT()` for strict, typed expectations:

```go
env.refresh.EXPECT().RefreshTracker(mock.Anything, uint(1)).Return(nil)
env.sched.EXPECT().Schedule(mock.AnythingOfType("domain.Tracker")).Return(nil)
```

When a route must **not** call the scraper (e.g. `GET /trackers`), add no expectations — testify/mock will fail the test on any unexpected call.

### What to cover

For each route, write at minimum:

| Scenario | Expected status |
|---|---|
| Happy path (valid input, data exists) | `200` / `204` |
| Not found (valid ID, missing record) | `404` |
| Invalid input / scrape failure | `422` |
| Unexpected service error | `500` |

The goal is to target **100% code coverage** on the handler method itself, which means covering all branches including error handling and scheduler calls.

Routes that trigger a scrape (`POST /trackers`, `PATCH /trackers/{id}`, `POST /trackers/{id}/refresh`) must cover both the **scrape-succeeds** and **scrape-fails** branches.

### Rules

- Call `testutil.NewDB(t)` inside the setup helper — never share a database between tests.
- Seed fixture data through the real repository or service, not by raw SQL.
- Never assert on `Credentials` field values in response bodies — the field is excluded from serialization.
- Do not mock `domain.TrackerService` or `domain.StatsService` in handler tests; the integration value comes from exercising the real service + repository stack.
