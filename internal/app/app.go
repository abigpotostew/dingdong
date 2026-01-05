package app

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/abigpotostew/dingdong/internal/handlers"
	"github.com/abigpotostew/dingdong/internal/migrations"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Run initializes and starts the Pocketbase application
func Run() error {
	app := pocketbase.New()

	// Parse templates
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return err
	}

	// Register migrations for database schema
	migrations.Register(app)

	// Setup routes
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Create handlers
		h := handlers.New(app, tmpl)

		// PING API endpoint with custom CORS handling
		e.Router.POST("/api/ping", func(re *core.RequestEvent) error {
			if err := handlePingCORS(app, re); err != nil {
				return err
			}
			return h.HandlePing(re)
		})

		e.Router.OPTIONS("/api/ping", func(re *core.RequestEvent) error {
			return handlePingPreflight(app, re)
		})

		// Tracker script endpoint
		e.Router.GET("/tracker.js", func(re *core.RequestEvent) error {
			return h.HandleTrackerScript(re)
		})

		// Robots.txt - block all crawlers
		e.Router.GET("/robots.txt", func(re *core.RequestEvent) error {
			content, err := staticFS.ReadFile("static/robots.txt")
			if err != nil {
				return re.Error(500, "Failed to read robots.txt", err)
			}
			re.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
			re.Response.Header().Set("Cache-Control", "public, max-age=86400")
			_, err = re.Response.Write(content)
			return err
		})

		// Admin portal routes
		e.Router.GET("/", func(re *core.RequestEvent) error {
			return h.HandleDashboard(re)
		})
		e.Router.GET("/sites", func(re *core.RequestEvent) error {
			return h.HandleSites(re)
		})
		e.Router.GET("/sites/{siteId}", func(re *core.RequestEvent) error {
			return h.HandleSiteStats(re)
		})
		e.Router.GET("/admin", func(re *core.RequestEvent) error {
			return h.HandleAdmin(re)
		})

		log.Println("DingDong server started")
		return e.Next()
	})

	// Check if running with arguments, otherwise use default serve
	args := os.Args
	if len(args) == 1 {
		os.Args = append(os.Args, "serve", "--http=0.0.0.0:8090")
	}

	return app.Start()
}

// handlePingCORS sets CORS headers for POST requests to /api/ping
func handlePingCORS(app *pocketbase.PocketBase, e *core.RequestEvent) error {
	origin := e.Request.Header.Get("Origin")
	if origin == "" {
		return nil
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return nil
	}

	domain := handlers.ExtractDomain(parsedOrigin.Host)

	// Check if domain is registered
	_, err = handlers.FindSiteByDomain(app, domain)
	if err != nil {
		return nil // Let the handler deal with unregistered domains
	}

	// Set CORS headers
	e.Response.Header().Set("Access-Control-Allow-Origin", origin)
	e.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	e.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	e.Response.Header().Set("Access-Control-Max-Age", "86400")

	return nil
}

// handlePingPreflight handles CORS preflight requests for /api/ping
func handlePingPreflight(app *pocketbase.PocketBase, e *core.RequestEvent) error {
	origin := e.Request.Header.Get("Origin")
	if origin == "" {
		return e.NoContent(http.StatusNoContent)
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return e.NoContent(http.StatusNoContent)
	}

	domain := handlers.ExtractDomain(parsedOrigin.Host)

	// Check if domain is registered
	_, err = handlers.FindSiteByDomain(app, domain)
	if err != nil {
		return e.NoContent(http.StatusForbidden)
	}

	// Set CORS headers
	e.Response.Header().Set("Access-Control-Allow-Origin", origin)
	e.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	e.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	e.Response.Header().Set("Access-Control-Max-Age", "86400")

	return e.NoContent(http.StatusNoContent)
}
