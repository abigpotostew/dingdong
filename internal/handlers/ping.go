package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/models"
)

// PingRequest represents the incoming ping data from the tracker
type PingRequest struct {
	Path         string `json:"path"`
	Referrer     string `json:"referrer"`
	ScreenWidth  int    `json:"screen_width"`
	ScreenHeight int    `json:"screen_height"`
}

// setCORSHeaders sets the CORS headers for the ping endpoint
func setCORSHeaders(c echo.Context, origin string) {
	c.Response().Header().Set("Access-Control-Allow-Origin", origin)
	c.Response().Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.Response().Header().Set("Access-Control-Max-Age", "86400")
	// Explicitly do NOT set Access-Control-Allow-Credentials
}

// HandlePingPreflight handles CORS preflight requests for the ping endpoint
func (h *Handlers) HandlePingPreflight(c echo.Context) error {
	origin := c.Request().Header.Get("Origin")
	if origin == "" {
		return c.NoContent(http.StatusNoContent)
	}

	// Parse the origin to get the domain
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return c.NoContent(http.StatusNoContent)
	}
	domain := ExtractDomain(parsedOrigin.Host)

	// Look up the site by domain (checks primary domain and additional_domains)
	_, err = FindSiteByDomain(h.app, domain)
	if err != nil {
		// Return forbidden without CORS headers - browser will block
		return c.NoContent(http.StatusForbidden)
	}

	// Set CORS headers for the allowed origin
	setCORSHeaders(c, origin)

	return c.NoContent(http.StatusNoContent)
}

// HandlePing processes incoming pageview pings from the JavaScript tracker
func (h *Handlers) HandlePing(c echo.Context) error {
	// Get the Origin header for CORS validation
	origin := c.Request().Header.Get("Origin")
	if origin == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing Origin header",
		})
	}

	// Parse the origin to get the domain
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid Origin header",
		})
	}
	domain := ExtractDomain(parsedOrigin.Host)

	// Look up the site by domain (checks primary domain and additional_domains)
	site, err := FindSiteByDomain(h.app, domain)
	if err != nil {
		// Site not registered - reject
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Domain not registered",
		})
	}

	// Set CORS headers for the allowed origin
	setCORSHeaders(c, origin)

	// Parse the request body
	var req PingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get client info
	userAgent := c.Request().Header.Get("User-Agent")
	clientIP := c.RealIP()

	// Hash the IP for privacy
	ipHash := hashIP(clientIP)

	// Create pageview record
	collection, err := h.app.Dao().FindCollectionByNameOrId("pageviews")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal error",
		})
	}

	record := models.NewRecord(collection)
	record.Set("site", site.Id)
	record.Set("path", req.Path)
	record.Set("referrer", req.Referrer)
	record.Set("user_agent", userAgent)
	record.Set("ip_hash", ipHash)
	record.Set("screen_width", req.ScreenWidth)
	record.Set("screen_height", req.ScreenHeight)

	if err := h.app.Dao().SaveRecord(record); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to record pageview",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// hashIP creates a privacy-preserving hash of the IP address
func hashIP(ip string) string {
	// Use a static salt (in production, could use daily rotating salt)
	hash := sha256.Sum256([]byte(ip + "-dingdong"))
	return hex.EncodeToString(hash[:8]) // Only use first 8 bytes
}
