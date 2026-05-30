# Scraper layer — conventions

## Purpose

Each scraper is a site-specific adapter that fetches upload / download / ratio stats from one private tracker. All scrapers implement `domain.TrackerScraper` and are registered in `module.go` via FX value groups.

---

## Files & types

| Element | Convention | Example |
|---|---|---|
| File | `{site}.go` (lower-case, no underscores) | `unit3d.go`, `c411.go` |
| Struct (exported) | `{Site}Scraper` | `Unit3DScraper`, `C411Scraper` |
| Constructor | `New{Site}Scraper() *{Site}Scraper` | `NewUnit3DScraper` |
| Embedding | Embed `BaseScraper` unless the site needs its own HTTP client (e.g. cookie jar) | |
| Key | Short lower-case identifier matching the `scraper_key` stored in the database | `"unit3d"`, `"c411"` |

---

## Interface

Every scraper must satisfy `domain.TrackerScraper`:

```go
type TrackerScraper interface {
    Key()              string
    CredentialFields() []CredentialField
    Fetch(ctx context.Context, tracker Tracker) (*TrackerStats, error)
}
```

- `Key()` — unique registry key; must match the value users enter in the UI.
- `CredentialFields()` — describes which JSON fields are required; used by the frontend to render the credential form. Return `nil` for generic/unstructured scrapers.
- `Fetch()` — performs the HTTP request(s) and returns parsed stats. Always propagate context so the caller can cancel long-running requests.

---

## BaseScraper

`BaseScraper` provides a pre-configured `http.Client` (30-second timeout) and two helpers:

| Method | When to use |
|---|---|
| `Get(ctx, rawCredentials string)` | Simple GET to the URL in credentials; sets Cookie, Bearer, X-Api-Key, and custom headers automatically |
| `DoRequest(ctx, method, url string, creds Credentials)` | Custom method or URL — e.g. for POST login flows or API endpoints different from the tracker URL |

Scrapers that need their own cookie jar (like `C411Scraper`) do **not** embed `BaseScraper` and manage their own `http.Client`.

---

## Credentials

`ParseCredentials(raw string) (Credentials, error)` decodes the `tracker.Credentials` JSON field. The `Credentials` struct has these fields:

| Field | Use |
|---|---|
| `URL` | Base URL of the tracker |
| `Cookie` | Session cookie |
| `Username` / `Password` | Username/password for login flows |
| `APIKey` | Sent as `X-Api-Key` header |
| `Token` | Sent as `Authorization: Bearer <token>` |
| `Headers` | Arbitrary extra headers (`map[string]string`) |

Call `ParseCredentials` at the top of `Fetch` and validate required fields immediately — return a descriptive error before making any HTTP request.

**Security**: `DoRequest` validates that the URL scheme is `http` or `https` to prevent SSRF via `file://`, `gopher://`, etc.

---

## Parse helpers (package-level)

| Helper | Purpose |
|---|---|
| `ParseJSON(body []byte, dst any) error` | Decode JSON response |
| `ParseHTML(body []byte) (*html.Node, error)` | Parse HTML into a node tree |
| `WalkHTML(root *html.Node, fn func(*html.Node) bool)` | Walk HTML tree; return `false` to stop |
| `ParseXML(body []byte, dst any) error` | Decode XML response |

---

## Adding a new scraper

1. Copy `generic.go`, rename the file and the struct.
2. Implement `Key()`, `CredentialFields()`, and `Fetch()`.
3. Register in `module.go`:

```go
fx.Provide(
    fx.Annotate(
        NewMyScraper,
        fx.As(new(domain.TrackerScraper)),
        fx.ResultTags(`group:"scrapers"`),
    ),
),
```

4. Add a test file `{site}_test.go` (see Testing conventions below).

---

## Error wrapping

Prefix every error with the scraper key:

```go
return nil, fmt.Errorf("unit3d: %w", err)
return nil, fmt.Errorf("unit3d: credentials must contain a \"token\" field")
```

---

## Testing conventions

### File & package

| Element | Convention | Example |
|---|---|---|
| File | `{site}_test.go` | `unit3d_test.go`, `scraper_test.go` |
| Package | `package scraper_test` (external black-box) | — |

Use `package scraper_test` so tests can only access exported types and functions.

### No real HTTP calls

Scrapers must never make live HTTP requests in tests. Use `net/http/httptest.NewServer` to spin up a fake tracker endpoint:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintln(w, `{"uploaded":1073741824,"downloaded":536870912,"ratio":2.0}`)
}))
defer srv.Close()

tracker := domain.Tracker{
    Credentials: fmt.Sprintf(`{"url":%q,"token":"test-token"}`, srv.URL),
}
stats, err := scraper.NewUnit3DScraper().Fetch(t.Context(), tracker)
require.NoError(t, err)
assert.Equal(t, int64(1073741824), stats.Uploaded)
```

### Structure

- **One `Test*` function per scraper** — e.g. `TestUnit3DScraper_Fetch`.
- **Sub-tests for cases** — happy path, missing credential fields, bad JSON response, HTTP error, etc.
- `TestParseCredentials` lives in `scraper_test.go` alongside `BaseScraper` / `GenericScraper` tests.

### What to test

| Scenario | Required |
|---|---|
| Happy path — correct stats returned | yes |
| Missing required credential field returns error | yes for each required field |
| Invalid credentials JSON returns error | yes |
| HTTP error (non-2xx) returns error | yes |
| Malformed / unexpected response body returns error | yes |
| `Key()` returns the expected string | yes |
| `CredentialFields()` returns the expected fields | yes |
| SSRF via `file://` or other non-http/https scheme returns error | yes (for scrapers using `DoRequest`) |

### Assertions

Use `testify/require` for the first error in a chain (makes the rest of the test skip on failure) and `testify/assert` for field comparisons:

```go
stats, err := s.Fetch(context.Background(), tracker)
require.NoError(t, err)
assert.Equal(t, int64(1073741824), stats.Uploaded)
assert.Equal(t, int64(536870912), stats.Downloaded)
assert.InDelta(t, 2.0, stats.Ratio, 0.001)
```

### Credential fixture helper

Define a small helper per test file to build credential JSON without string noise:

```go
func creds(url, token string) string {
    b, _ := json.Marshal(map[string]string{"url": url, "token": token})
    return string(b)
}
```
