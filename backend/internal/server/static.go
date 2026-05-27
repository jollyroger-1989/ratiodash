package server

import "embed"

// staticFiles holds the embedded Vue SPA assets produced by `npm run build`.
// The dist/ directory is populated at build time (Dockerfile or `make build`).
// In development the directory contains only a .gitkeep placeholder, so
// NewRouter detects the missing index.html and skips static-file serving.
//
//go:embed all:dist
var staticFiles embed.FS
