# Notifier layer — conventions

Notifiers deliver `domain.Notification` values to external channels (HTTP push, e-mail, webhooks, etc.). The `domain.Notifier` interface is the only dependency callers import; concrete implementations are invisible to the rest of the application.

---

## Files & types

| Element | Convention | Example |
|---|---|---|
| File | `{transport}.go` | `ntfy.go`, `mail.go` |
| Struct (unexported) | `{transport}Notifier` | `ntfyNotifier`, `mailNotifier` |
| Constructor | `New{Transport}Notifier(cfg …) (domain.Notifier, error)` | `NewNtfyNotifier` |
| FX registration | `fx.Annotate(New…, fx.As(new(domain.Notifier)), fx.ResultTags(`group:"notifiers"`))` in `module.go` | |

The `multiNotifier` in `multi.go` fans out to all group members and is the single `domain.Notifier` exposed to the rest of the application. **Do not add a second `fx.Provide` for `domain.Notifier`.**

---

## Adding a new backend

1. Create `internal/notifier/{transport}.go`.
2. Define an unexported struct implementing `domain.Notifier`:

```go
type ntfyNotifier struct{ topic string }

func NewNtfyNotifier(cfg config.Config) (domain.Notifier, error) {
    if cfg.NtfyTopic == "" {
        return nil, nil // disabled — MultiNotifier tolerates nils
    }
    return &ntfyNotifier{topic: cfg.NtfyTopic}, nil
}

func (n *ntfyNotifier) Notify(ctx context.Context, notif domain.Notification) error {
    // POST to ntfy …
}
```

3. Append to `module.go`:

```go
fx.Provide(
    fx.Annotate(
        NewNtfyNotifier,
        fx.As(new(domain.Notifier)),
        fx.ResultTags(`group:"notifiers"`),
    ),
),
```

---

## domain.Notification fields

| Field | Purpose |
|---|---|
| `Event` | What triggered the notification (`report`, `ratio_alert`, `sync_error`). Use to filter or route. |
| `Level` | Urgency: `info`, `warning`, `error`. Map to ntfy priority, e-mail subject prefix, etc. |
| `Title` | Short summary — use as subject line or push headline. |
| `Body` | Full human-readable message. |
| `Tags` | Free-form labels (e.g. tracker name). Surface as metadata or ignore. |

---

## Rules

- **Constructors return `nil, nil` when disabled** — check a config flag (topic URL, SMTP host, etc.) and return `(nil, nil)` rather than an error when a backend is not configured. `MultiNotifier` skips nil entries.
- **One responsibility** — a notifier only formats and delivers; routing, filtering, and retry logic belong in the caller or a wrapper.
- **Context propagation** — always forward `ctx` to HTTP clients, SMTP dials, etc. to respect request deadlines and cancellation.
- **No business logic** — notifiers must not query the database or call services.
- **Error wrapping** — wrap errors with context: `fmt.Errorf("ntfy notify: %w", err)`.

---

## Wiring in `cmd/api/main.go`

Insert `notifier.Module` after `service` and before `handler` in the FX options list:

```
config → database → repository → scraper → service → notifier → scheduler → handler → server
```

---

## Testing conventions

Notifier tests are **unit tests**: they spin up a local `httptest.Server` (for HTTP-based backends) or stub collaborators directly, and assert on outgoing requests or return values — no real external services are contacted.

### Files & packages

| Element | Convention | Example |
|---|---|---|
| File | `{transport}_test.go` | `ntfy_test.go`, `multi_test.go` |
| Package | `notifier_test` (external black-box) | — |
| Test func | `Test{Transport}Notifier_{Method}` | `TestNtfyNotifier_Notify` |
| Subtest | `t.Run("…descriptive phrase…", …)` | `t.Run("sets Authorization header when token is set", …)` |

### HTTP backends — test server pattern

Use `net/http/httptest.NewServer` to intercept outgoing requests without contacting a real service. Capture the request inside the handler, then assert on headers and body after `Notify` returns.

```go
func TestNtfyNotifier_Notify(t *testing.T) {
    t.Run("sets correct priority for each level", func(t *testing.T) {
        var got *http.Request
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            got = r.Clone(context.Background())
            w.WriteHeader(http.StatusOK)
        }))
        defer srv.Close()

        n, err := notifier.NewNtfyNotifier(&config.Config{NtfyURL: srv.URL})
        require.NoError(t, err)
        require.NotNil(t, n)

        err = n.Notify(context.Background(), domain.Notification{
            Event: domain.EventSyncError,
            Level: domain.LevelError,
            Title: "test title",
            Body:  "test body",
        })

        require.NoError(t, err)
        assert.Equal(t, "5", got.Header.Get("Priority"))
        assert.Equal(t, "test title", got.Header.Get("Title"))
    })
}
```

### What to test for every HTTP backend

| Scenario | Required |
|---|---|
| Constructor returns `nil, nil` when URL/host is empty | yes |
| Constructor returns a non-nil `domain.Notifier` when configured | yes |
| Correct HTTP method and path reached the server | yes |
| `Title` header matches `Notification.Title` | yes |
| Priority / urgency header matches each `NotificationLevel` | yes — one sub-test per level |
| `Tags` header contains the event name | yes |
| Auth header is set when a token/password is configured | yes |
| Auth header is absent when no token/password is configured | yes |
| Non-2xx response from the server returns an error | yes |
| Unreachable server (close srv before Notify) returns an error | yes |
| Cancelled context returns an error | yes |

### multiNotifier tests (`multi_test.go`)

| Scenario | Required |
|---|---|
| All registered backends are called | yes |
| A failure in one backend does not prevent the rest from being called | yes |
| Errors from multiple backends are all returned (joined) | yes |
| `nil` entries in the group are silently skipped | yes |
| Empty notifier list is a no-op and returns nil | yes |

```go
func TestMultiNotifier_Notify(t *testing.T) {
    t.Run("calls all backends", func(t *testing.T) {
        called := 0
        stub := stubNotifier(func(_ context.Context, _ domain.Notification) error {
            called++
            return nil
        })

        n := notifier.NewMultiNotifier(notifier.MultiParams{Notifiers: []domain.Notifier{stub, stub}})
        require.NoError(t, n.Notify(context.Background(), domain.Notification{}))
        assert.Equal(t, 2, called)
    })
}
```

Define a minimal `stubNotifier` helper inside the test file:

```go
type stubNotifier func(context.Context, domain.Notification) error

func (s stubNotifier) Notify(ctx context.Context, n domain.Notification) error { return s(ctx, n) }
```

### Assertions

Use `github.com/stretchr/testify/require` for setup steps and the first error check that would make subsequent assertions meaningless. Use `assert` for everything else.

```go
require.NoError(t, err)           // fatal — no point continuing
assert.Equal(t, "5", priority)    // non-fatal header check
assert.NotNil(t, notif)           // non-fatal nil guard
```

### Coverage requirement

Tests must achieve **100% statement coverage** of the implementation file. Run with:

```bash
go test -coverprofile=cover.out ./internal/notifier/...
go tool cover -func=cover.out
```
