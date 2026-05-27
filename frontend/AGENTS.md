# RatioDash Frontend — Agent Guidelines

## Stack

- **Vue 3** with `<script setup>` and Composition API (no Options API)
- **TypeScript** — strict, no `any` unless unavoidable
- **Vite** build tool (`npm run build` runs `vue-tsc && vite build`)
- **Pinia** for global state (`src/stores/`)
- **vue-router** for navigation (`src/router/index.ts`)
- **vue-i18n** for all user-facing strings (`src/i18n/`)
- **Axios** HTTP client wrapped in `src/services/api.ts`
- **Chart.js + vue-chartjs** for data visualisation

## Verification

Always run before finishing:

```bash
npm run build        # type-checks (vue-tsc) then builds
# or for type-check only:
npm run type-check
```

Zero errors are required. Do not leave TypeScript or template compilation errors.

## Project Structure

```
src/
  assets/main.css          # global CSS variables + utility classes
  components/              # reusable SFCs
  composables/             # pure TS helpers (no Vue reactivity at module level)
  i18n/locales/            # en.ts, fr.ts — translation dictionaries
  router/index.ts
  services/api.ts          # all HTTP calls + shared TS interfaces
  stores/                  # Pinia stores
  views/                   # route-level components
```

## Component Conventions

### Single-File Components (SFCs)

- Order: `<template>` → `<script setup lang="ts">` → `<style scoped>`
- Use `<script setup lang="ts">` exclusively — no `export default`
- Props typed with `defineProps<{}>()`, emits with `defineEmits<{}>()`

### Reusable Infrastructure

Before adding logic that already exists, check these first:

| Need | Use |
|---|---|
| Modal dialog | `<BaseModal>` — props: `modelValue`, `titleId`, `size` (`sm`/`md`/`lg`) |
| Tracker stats display | `<StatsGrid>` — props: `stats: TrackerStats`, `showDate?: boolean` |
| History rows with delete | `<HistoryTable>` — props: `history: TrackerStats[]`, emits `delete` |
| Cron schedule picker | `<CronSelect>` — `v-model` string |
| Page title + action slot | `<PageHeader>` — prop `title: string`, default slot for action button |
| Format bytes | `import { formatBytes } from '@/composables/formatBytes'` |
| Ratio CSS class | `import { ratioClass } from '@/composables/ratioClass'` |
| Cron preset list | `import { CRON_PRESETS } from '@/composables/cronPresets'` |

### Do Not Duplicate

- Never copy `formatBytes`, `ratioClass`, or `CRON_PRESETS` inline in a component
- Never write raw Teleport/backdrop/modal-header boilerplate — use `BaseModal`
- Never write `.ratio-good / .ratio-warn / .ratio-bad` in scoped CSS — they are global

## Styling Rules

### Global CSS (`src/assets/main.css`)

Design tokens are CSS custom properties on `:root`. Always use them:

```
--bg, --bg-surface, --bg-surface-hover
--border, --border-bright
--text, --text-muted, --text-label
--accent, --accent-2, --accent-glow
--ratio-good, --ratio-warn, --ratio-bad
--shadow, --shadow-glow
```

**Global utility classes available in every component (no import needed):**

- `.ratio-good / .ratio-warn / .ratio-bad` — ratio colour
- `.muted / .muted-small` — de-emphasised text
- `.field` — form field wrapper (label + input/select/textarea)
- `.field-optional` — inline label suffix
- `.fields-row / .fields-row--config` — 2-col / 1-col form grid
- `.form-actions` — button row at bottom of form
- `.form-error / .form-success` — feedback boxes
- `.btn-primary / .btn-secondary / .btn-test` — standard buttons
- `.btn-icon / .btn-icon.btn-edit / .btn-icon.btn-delete` — small icon buttons

### Scoped Styles

- Only put styles in `<style scoped>` that are **specific to that component**
- Do not redeclare any of the global utility classes above in scoped blocks
- Use `var(--token)` instead of hardcoded hex values wherever a token exists

## Internationalisation

- Every user-visible string must use `$t('key')` in templates or `t('key')` in script
- Dates must use `$d(date, 'short')` in templates or `d(date, 'short')` in script
- Add keys to **both** `src/i18n/locales/en.ts` and `src/i18n/locales/fr.ts`
- Follow the existing nested key structure (e.g. `common.cancel`, `dashboard.modal.title`)

## State Management (Pinia)

- Stores use the **setup store** pattern (`defineStore('id', () => { ... })`)
- Expose `loading: Ref<boolean>`, `error: Ref<string | null>` on every store that fetches data
- Call `error.value = null` at the start of each async action
- Use `finally` to reset `loading`

## API Layer (`src/services/api.ts`)

- All HTTP calls go through the shared `http` Axios instance — never create a second one
- Define request/response types as TypeScript interfaces in `api.ts` and export them
- The interceptor handles 401 → redirect to `/login` automatically

## Router

- Public routes must have `meta: { public: true }`
- The setup-only route must also have `meta: { setupOnly: true }`
- Lazy-load all routes except `HomeView` (already eager)

## Things to Avoid

- Do not add `eslint-disable` comments — fix the issue instead
- Do not use `!important` in CSS
- Do not store sensitive data (tokens, credentials) anywhere except `localStorage` under the existing `auth_token` key, which the API interceptor reads
- Do not bypass the shared `http` instance to call the API directly
- Do not add new dependencies without a clear reason — the current stack covers all common needs
