package web

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Frontend struct {
	publicDir           string
	sharedDir           string
	vendorDir           string
	vendorMarkedDir     string
	vendorFontAwesome   string
	sharedHandler       http.Handler
	vendorHandler       http.Handler
	vendorMarkedHandler http.Handler
	vendorFontHandler   http.Handler
}

func NewFrontend(rootDir string) *Frontend {
	sharedDir := filepath.Join(rootDir, "src", "shared")
	vendorDir := filepath.Join(rootDir, "node_modules", "redom", "dist")
	vendorMarkedDir := filepath.Join(rootDir, "node_modules", "marked", "lib")
	vendorFontAwesome := filepath.Join(rootDir, "node_modules", "@fortawesome", "fontawesome-free")

	return &Frontend{
		publicDir:           filepath.Join(rootDir, "src", "public"),
		sharedDir:           sharedDir,
		vendorDir:           vendorDir,
		vendorMarkedDir:     vendorMarkedDir,
		vendorFontAwesome:   vendorFontAwesome,
		sharedHandler:       http.StripPrefix("/shared/", http.FileServer(http.Dir(sharedDir))),
		vendorHandler:       http.StripPrefix("/vendor/", http.FileServer(http.Dir(vendorDir))),
		vendorMarkedHandler: http.StripPrefix("/vendor-marked/", http.FileServer(http.Dir(vendorMarkedDir))),
		vendorFontHandler:   http.StripPrefix("/vendor-fontawesome/", http.FileServer(http.Dir(vendorFontAwesome))),
	}
}

func (frontend *Frontend) TryServeStatic(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case r.Method != http.MethodGet && r.Method != http.MethodHead:
		return false
	case r.URL.Path == "/app.js":
		http.ServeFile(w, r, filepath.Join(frontend.publicDir, "app.js"))
		return true
	case r.URL.Path == "/styles.css":
		http.ServeFile(w, r, filepath.Join(frontend.publicDir, "styles.css"))
		return true
	case strings.HasPrefix(r.URL.Path, "/shared/"):
		frontend.sharedHandler.ServeHTTP(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, "/vendor/"):
		frontend.vendorHandler.ServeHTTP(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, "/vendor-marked/"):
		frontend.vendorMarkedHandler.ServeHTTP(w, r)
		return true
	case strings.HasPrefix(r.URL.Path, "/vendor-fontawesome/"):
		frontend.vendorFontHandler.ServeHTTP(w, r)
		return true
	default:
		return false
	}
}

func (frontend *Frontend) ServeIndex(w http.ResponseWriter, statusCode int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	io.WriteString(w, indexHTML)
}

func (frontend *Frontend) Validate() error {
	paths := []string{
		filepath.Join(frontend.publicDir, "app.js"),
		filepath.Join(frontend.publicDir, "styles.css"),
		frontend.sharedDir,
		frontend.vendorDir,
		frontend.vendorMarkedDir,
		frontend.vendorFontAwesome,
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return err
		}
	}

	return nil
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>OpenAI Batch Web App</title>
    <link rel="stylesheet" href="/vendor-fontawesome/css/all.min.css" />
    <link rel="stylesheet" href="/styles.css" />
  </head>
  <body>
    <div class="page-shell">
      <main id="app-root"></main>
    </div>
    <script type="module" src="/app.js"></script>
  </body>
</html>
`
