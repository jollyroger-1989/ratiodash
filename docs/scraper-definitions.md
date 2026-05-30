# Writing YAML Scraper Definitions

This guide explains how to create and maintain scraper definition files for RatioDash.

A scraper definition is a YAML file that tells the backend:

- which credentials the user must provide
- how to log in (if needed)
- where to fetch stats
- how to extract uploaded, downloaded, and ratio values

Definitions are loaded from `backend/scrapers/*.yml` at startup.

## 1. Start with a minimal working definition

Create a new file named after your scraper key, for example:

- `backend/scrapers/mytracker.yml`

Use this skeleton:

```yaml
id: mytracker
name: MyTracker
description: >-
  Private tracker example definition.
language: en-US
type: private
encoding: UTF-8
links:
  - https://tracker.example/

settings:
  - name: username
    type: text
    label: Username
    required: true
  - name: password
    type: password
    label: Password
    required: true

login:
  method: form
  path: /login
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"

stats:
  path: /api/me
  response:
    type: json
  fields:
    uploaded:
      selector: uploaded
      filters:
        - name: parsebytes
    downloaded:
      selector: downloaded
      filters:
        - name: parsebytes
    ratio:
      selector: ratio
      filters:
        - name: parsefloat
```

Required fields:

- top-level `id`
- `stats.path`

## 2. Top-level fields

- `id`: unique key used in UI and database (`scraper_key`)
- `name`: display name
- `description`: human-readable details
- `language`: language/locale hint (example `en-US`, `fr-FR`)
- `type`: usually `private`
- `encoding`: usually `UTF-8`
- `links`: optional fallback base URLs

How base URL is chosen:

1. `credentials.url` if user provided it
2. first entry in `links`

Only `http` and `https` URLs are allowed.

## 3. settings: credential form fields

`settings` defines fields shown in the frontend credential form.

Each setting supports:

- `name`: key used in templates (`.Config.<name>`)
- `type`: `text` or `password`
- `label`: UI label
- `required`: whether backend must reject missing value

Example:

```yaml
settings:
  - name: token
    type: password
    label: API Token
    required: true
```

## 4. login block (optional)

Skip `login` when stats endpoint only needs a token/header and no session login.

When login is needed, choose one method:

- `form`: GET page, extract values, POST submit
- `json`: POST JSON directly
- `post`: POST form body directly

### 4.1 form login

Use for classic login pages with CSRF.

```yaml
login:
  method: form
  path: /login
  submitpath: /api/auth/login
  contenttype: application/json
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
  selectorinputs:
    csrf_token:
      selector: "input[name=csrf_token]"
      attribute: value
  selectorheaders:
    csrf-token:
      selector: "meta[name=csrf-token]"
      attribute: content
```

- `selectorinputs` adds scraped values to POST body
- `selectorheaders` adds scraped values to request headers

### 4.2 json login with captures

Use when API returns a token in JSON.

```yaml
login:
  method: json
  path: /api/v1/auth/login
  inputs:
    username: "{{ .Config.username }}"
    password: "{{ .Config.password }}"
  response:
    type: json
  captures:
    token:
      selector: token
```

Then reuse in stats headers:

```yaml
stats:
  headers:
    Authorization: "Bearer {{ .Captures.token }}"
```

### 4.3 login error detection

Use `error` to detect failed auth explicitly.

For JSON responses:

```yaml
error:
  - selector: success
    value: "false"
```

For HTML responses, use a selector for an error banner/message.

## 5. stats block

`stats` controls where stats are fetched and how values are extracted.

Fields:

- `path`: endpoint path (can be template)
- `headers`: optional templated headers
- `response.type`: `json` or `html` (default is `html`)
- `fields`: extraction map, in order

Example:

```yaml
stats:
  path: /api/user
  headers:
    Authorization: "Bearer {{ .Config.token }}"
  response:
    type: json
  fields:
    uploaded:
      selector: uploaded
    downloaded:
      selector: downloaded
    ratio:
      selector: ratio
      filters:
        - name: parsefloat
```

## 6. fields: extraction rules

Each field can use one of two sources:

- `selector`: extract from JSON or HTML
- `text`: compute using template

Supported field options:

- `selector`
- `attribute` (HTML only)
- `match` (HTML only: `first` or `last`)
- `text`
- `optional`
- `default`
- `filters`

### 6.1 JSON selectors

JSON selectors use gjson dot paths:

```yaml
selector: user.uploaded
```

### 6.2 HTML selectors and match behavior

HTML selectors use CSS selectors.

When selector can match both outer container and inner value element, set:

```yaml
match: last
```

This is especially useful for nested blocks like label/value grids.

### 6.3 optional + default

Use when a field may not exist on some accounts:

```yaml
_bonus_downloaded:
  selector: bonus_downloaded
  optional: true
  default: "0"
```

## 7. Templates and computed fields

Template context:

- `.Config.<name>` from credentials
- `.Captures.<name>` from login captures
- `.Result.<field>` from already computed stats fields

Important: `fields` are evaluated in YAML order, so computed fields must appear after dependencies.

Example (sum and ratio computation):

```yaml
fields:
  _uploaded:
    selector: total_uploaded_bytes
  _bonus_uploaded:
    selector: bonus_uploaded
    optional: true
    default: "0"
  uploaded:
    text: "{{ isum .Result._uploaded .Result._bonus_uploaded }}"
  _downloaded:
    selector: total_downloaded_bytes
  _bonus_downloaded:
    selector: bonus_downloaded
    optional: true
    default: "0"
  downloaded:
    text: "{{ isum .Result._downloaded .Result._bonus_downloaded }}"
  ratio:
    text: "{{ fratio .Result.uploaded .Result.downloaded }}"
```

Built-in template functions:

- `isum a b`: integer sum as string
- `fratio uploaded downloaded`: safe division (returns `0` if downloaded is zero)
- `re_replace s pattern replacement`

## 8. Filters

Filters run in order and transform the extracted value.

Common filters:

- `parsebytes`: converts `2.00 Go`, `1.5 GB`, `700 MiB` to raw bytes
- `parsefloat`: parses ratio-like values and maps `∞`, `N/A`, `-` to `0`
- `trim`, `replace`, `re_replace`, `regexp`, `tolower`, `toupper`
- `urlencode`, `urldecode`, `querystring`, `split`, `append`, `prepend`, `htmldecode`

Example:

```yaml
filters:
  - name: re_replace
    args: [",", "."]
  - name: parsefloat
```

## 9. Selector compatibility note

Prefer this style inside `:has(...)`:

```yaml
selector: 'div:has(div.mt-1:contains("Upload")) > div:first-child'
```

Avoid direct-child combinator inside `:has(...)` for compatibility.

If matching is still ambiguous, use `match: last`.

## 10. Logging behavior

The scraper engine logs key steps using standard library logging:

- fetch start
- login requests and status
- captures
- stats request
- final parsed values

Do not put secrets in templates or logs.

## 11. Validation checklist before commit

- `id` matches intended `scraper_key`
- required credentials declared in `settings`
- login flow works with real response format (`html` or `json`)
- `uploaded`, `downloaded`, `ratio` are present (or ratio is computed)
- selectors tested against realistic nested HTML/JSON
- optional fields include sane defaults
- filters produce numeric values expected by backend

Then run tests:

- `go test ./internal/scraper/...`
- `go test ./...`

## 12. Real examples in this repository

- `backend/scrapers/unit3d.yml`: token-based JSON endpoint, no login block
- `backend/scrapers/torr9.yml`: JSON login + token capture + computed totals
- `backend/scrapers/c411.yml`: form login with CSRF header extraction
- `backend/scrapers/yggreborn.yml`: HTML selectors with `match: last`
