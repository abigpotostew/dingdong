package app

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/abigpotostew/stewstats/internal/handlers"
	"github.com/abigpotostew/stewstats/internal/migrations"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
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

		// Add custom CORS middleware for /api/ping that runs before everything else
		e.Router.Use(pingCORSMiddleware(app))

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

// pingCORSMiddleware creates a middleware that handles CORS for the /api/ping endpoint
// This runs before Pocketbase's default CORS to prevent conflicts
func pingCORSMiddleware(app *pocketbase.PocketBase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Only handle /api/ping endpoint
			if !strings.HasPrefix(c.Request().URL.Path, "/api/ping") {
				return next(c)
			}

			origin := c.Request().Header.Get("Origin")
			if origin == "" {
				return next(c)
			}

			// For OPTIONS preflight, we need to handle it completely here
			// to prevent Pocketbase's CORS from interfering
			if c.Request().Method == http.MethodOptions {
				c.Response().Header().Set("Access-Control-Allow-Origin", origin)
				c.Response().Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type")
				c.Response().Header().Set("Access-Control-Max-Age", "86400")
				return c.NoContent(http.StatusNoContent)
			}

			// For POST requests, set headers before the handler runs
			c.Response().Header().Set("Access-Control-Allow-Origin", origin)
			c.Response().Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type")

			return next(c)
		}
	}
}

// Use the middleware package's CORS if needed
var _ = middleware.CORS
