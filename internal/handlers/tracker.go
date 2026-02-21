package handlers

import (
	"embed"
	"os"
	"strings"
	"sync"

	"github.com/pocketbase/pocketbase/core"
)

//go:embed static/tracker.min.js
var trackerFS embed.FS

var (
	trackerScript     string
	trackerScriptOnce sync.Once
)

// loadTrackerScript loads the minified tracker script from embedded filesystem
func loadTrackerScript() string {
	trackerScriptOnce.Do(func() {
		content, err := trackerFS.ReadFile("static/tracker.min.js")
		if err != nil {
			trackerScript = "console.error('Failed to load tracker script');"
			return
		}
		trackerScript = string(content)
	})
	return trackerScript
}

// GetPublicURL returns the public URL for the application
func GetPublicURL(e *core.RequestEvent) string {
	if publicURL := os.Getenv("PUBLIC_URL"); publicURL != "" {
		return strings.TrimSuffix(publicURL, "/")
	}

	scheme := "https"
	if e.Request.TLS == nil {
		forwardedProto := e.Request.Header.Get("X-Forwarded-Proto")
		if forwardedProto != "" {
			scheme = forwardedProto
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + e.Request.Host
}

// HandleTrackerScript serves the JavaScript tracker with the correct endpoint
func (h *Handlers) HandleTrackerScript(e *core.RequestEvent) error {
	endpoint := GetPublicURL(e)
	script := strings.ReplaceAll(loadTrackerScript(), "{{ENDPOINT}}", endpoint)

	e.Response.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	e.Response.Header().Set("Cache-Control", "public, max-age=86400")

	_, err := e.Response.Write([]byte(script))
	return err
}
