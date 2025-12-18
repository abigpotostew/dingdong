package app

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/abigpotostew/stewstats/internal/handlers"
	"github.com/abigpotostew/stewstats/internal/migrations"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed templates/*
var templatesFS embed.FS

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

	// Setup routes before serving
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// Create handlers
		h := handlers.New(app, tmpl)

		// PING API endpoint - public with CORS validation
		e.Router.POST("/api/ping", h.HandlePing)

		// Handle CORS preflight for ping endpoint
		e.Router.OPTIONS("/api/ping", h.HandlePingPreflight)

		// Tracker script endpoint - serves JavaScript with proper content-type
		e.Router.GET("/tracker.js", h.HandleTrackerScript)

		// Admin portal routes - publicly readable stats view
		e.Router.GET("/", h.HandleDashboard)
		e.Router.GET("/sites", h.HandleSites)
		e.Router.GET("/sites/:siteId", h.HandleSiteStats)

		// Serve static files for tracker script
		e.Router.GET("/static/*", echo.StaticDirectoryHandler(echo.MustSubFS(templatesFS, "templates"), false))

		log.Println("StewStats server started")
		return nil
	})

	// Check if running with arguments, otherwise use default serve
	args := os.Args
	if len(args) == 1 {
		os.Args = append(os.Args, "serve", "--http=0.0.0.0:8090")
	}

	return app.Start()
}

// getClientIP extracts the real client IP from request headers
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs; the first is the client
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
