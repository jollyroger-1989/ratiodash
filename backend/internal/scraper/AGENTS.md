# Scraper Layer — YAML Engine Conventions

## Purpose

The scraper package is YAML-first. Each tracker scraper is defined by a `.yml`
file under `backend/scrapers/` and loaded at startup into a runtime
`YAMLScraper` implementing `domain.TrackerScraper`.

Code-defined scrapers are optional and are only needed for tracker-specific
behavior that cannot be expressed in YAML.

---

## Architecture

- `loader.go`: discovers and parses `*.yml` files from `SCRAPERS_DIR`
- `definition.go`: YAML schema (`Definition`, `LoginDef`, `StatsDef`, etc.)
- `yaml_scraper.go`: execution engine (login + stats fetch + extraction)
- `filter.go`: built-in field filters (parsebytes, parsefloat, regex, etc.)
- `template.go`: Go template rendering with helper funcs (`isum`, `fratio`, `re_replace`)
- `html_extractor.go`: CSS extraction via goquery
- `json_extractor.go`: JSON extraction via gjson
- `registry.go`: combines optional code scrapers (FX group) with loaded YAML scrapers
- `module.go`: wires `NewLoader` and `NewRegistry`

---

## Runtime Wiring

`Module` provides `NewLoader` and `NewRegistry`, then exports registry as
`domain.ScraperRegistry`.

Registry population flow:

1. collect code scrapers from FX group `group:"scrapers"` (if any)
2. load YAML scrapers from `SCRAPERS_DIR` (default `./scrapers`)
3. merge both into a key → scraper map

If `SCRAPERS_DIR` does not exist, loader returns an empty list (not an error).

---

## YAML Definition Rules

### Required fields

- `id`
- `stats.path`

### Common sections

- `settings`: credential fields shown in UI (`name`, `type`, `label`, `required`)
- `login`: optional auth flow
- `stats`: endpoint + extraction map for `uploaded`, `downloaded`, `ratio`

### Login methods

- `method: form`: GET login page, extract dynamic inputs/headers, POST submit
- `method: json`: POST JSON body
- `method: post`: POST form body without preflight GET

### Login extraction features

- `selectorinputs`: extract values into POST body
- `selectorheaders`: extract values into HTTP headers (for CSRF-header flows)
- `captures`: save response values into template context (`.Captures.*`)
- `error`: failure indicators for HTML selectors or JSON path values

---

## Extraction Behavior

`stats.fields` is order-preserving (`OrderedFields`) so later template fields can
reference earlier results via `.Result`.

Field behavior:

- `selector`: CSS selector (HTML) or gjson path (JSON)
- `attribute`: HTML attribute to read, defaults to element text
- `text`: Go template value source (bypasses selector)
- `optional` + `default`: fallback behavior
- `filters`: transform pipeline
- `match`: HTML-only selector disambiguation

`match` values:

- `first` (default): use first matched element
- `last`: use last matched element (useful when selector matches both container and leaf)

---

## Selector Notes

goquery/cascadia does not reliably support direct-child combinators inside
`:has(...)` for this project setup. Prefer:

- `div:has(div.mt-1:contains("Upload"))`

over:

- `div:has(> div.mt-1:contains("Upload"))`

If a selector still matches both outer and inner nodes, set `match: last`.

---

## Credentials And URL Safety

Credentials are parsed from tracker JSON into a string map.

- required `settings` fields must be present before requests start
- base URL resolution prefers `credentials.url`, then falls back to definition `links[0]`
- only `http` and `https` URLs are allowed

---

## Logging

Use `logrus` with a `scraper` field:

- `scraper <id>: fetching stats ...`
- `scraper <id>: GET/POST ...`
- `scraper <id>: login successful`
- `scraper <id>: stats -> uploaded=... downloaded=... ratio=...`

Do not log secrets (passwords, tokens, cookie values).

---

## Adding Or Updating A Scraper

1. Add or edit `backend/scrapers/<key>.yml`.
2. Keep `id` aligned with DB/UI `scraper_key`.
3. Add/adjust tests in `yaml_scraper_test.go` with `httptest.NewServer` fixtures.
4. Validate HTML selectors against realistic nested markup.
5. Run scraper package tests, then `go test ./...`.

Only add a code-defined scraper when YAML cannot express the required behavior.

---

## Testing Conventions

- no live HTTP calls; use `httptest.NewServer`
- cover happy path + auth failures + extraction failures + malformed responses
- include edge cases for bytes/float parsing filters
- for nested HTML selectors, include container-wrapping fixtures and assert `match: last` behavior where needed

Useful test helpers are exported via `export_test.go` for parser/template/filter
coverage.
