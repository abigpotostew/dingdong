package app

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

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

		// Add custom CORS middleware for /api/ping using Pre() to run BEFORE Pocketbase's CORS
		e.Router.Pre(pingCORSMiddleware(app))

		// PING API endpoint - public with CORS validation
		e.Router.POST("/api/ping", h.HandlePing)

		// Handle CORS preflight for ping endpoint (backup, Pre middleware should handle it)
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
// Using Pre() ensures this runs BEFORE Pocketbase's default CORS middleware
func pingCORSMiddleware(app *pocketbase.PocketBase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Only handle /api/ping endpoint
			if path != "/api/ping" {
				return next(c)
			}

			origin := c.Request().Header.Get("Origin")
			if origin == "" {
				return next(c)
			}

			// Validate the origin against registered domains
			parsedOrigin, err := url.Parse(origin)
			if err != nil {
				return next(c)
			}

			domain := parsedOrigin.Host
			// Remove port if present
			if colonIdx := strings.LastIndex(domain, ":"); colonIdx != -1 {
				if !strings.Contains(domain, "]") || strings.LastIndex(domain, "]") < colonIdx {
					domain = domain[:colonIdx]
				}
			}

			// Check if domain is registered
			_, err = app.Dao().FindFirstRecordByFilter("sites", "domain = {:domain} && active = true", map[string]any{
				"domain": domain,
			})
			if err != nil {
				// Domain not registered - don't set CORS headers, browser will block
				if c.Request().Method == http.MethodOptions {
					return c.NoContent(http.StatusForbidden)
				}
				return next(c)
			}

			// Set CORS headers for allowed origin (not wildcard!)
			c.Response().Header().Set("Access-Control-Allow-Origin", origin)
			c.Response().Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type")
			c.Response().Header().Set("Access-Control-Max-Age", "86400")

			// For OPTIONS preflight, respond immediately and don't continue
			// This prevents Pocketbase's CORS from overriding our headers
			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	}
}
