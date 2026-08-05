// Package webui embeds the built React client (apps/client-beta). Run
// `pnpm --filter client-beta build` before building the Go binaries.
package webui

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// FS exposes the built client under dist/ (index.html + hashed assets).
var FS, _ = fs.Sub(distFS, "dist")

// IndexHTML returns the client entry page.
func IndexHTML() ([]byte, error) {
	return fs.ReadFile(FS, "index.html")
}

// Asset reads a built client asset by path relative to dist/ (e.g.
// "assets/index-abc.js").
func Asset(name string) ([]byte, error) {
	return fs.ReadFile(FS, name)
}

// ContentType returns a best-effort HTTP content type for a built asset.
func ContentType(name string) string {
	switch {
	case strings.HasSuffix(name, ".js"):
		return "application/javascript"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(name, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(name, ".json"):
		return "application/json"
	case strings.HasSuffix(name, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(name, ".webp"):
		return "image/webp"
	case strings.HasSuffix(name, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(name, ".woff"):
		return "font/woff"
	case strings.HasSuffix(name, ".map"):
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
