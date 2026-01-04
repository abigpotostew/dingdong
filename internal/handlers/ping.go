package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

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
		log.Println("[ping] Missing Origin header")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing Origin header",
		})
	}

	// Parse the origin to get the domain
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		log.Printf("[ping] Invalid Origin header: %s, error: %v\n", origin, err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid Origin header",
		})
	}
	domain := ExtractDomain(parsedOrigin.Host)

	// Look up the site by domain (checks primary domain and additional_domains)
	site, err := FindSiteByDomain(h.app, domain)
	if err != nil {
		log.Printf("[ping] Domain not registered: %s\n", domain)
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Domain not registered",
		})
	}

	// Set CORS headers for the allowed origin
	setCORSHeaders(c, origin)

	// Read and parse the request body manually for better error handling
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Printf("[ping] Failed to read body: %v\n", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to read request body",
		})
	}

	// Handle empty body gracefully
	if len(body) == 0 {
		log.Println("[ping] Empty request body")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Empty request body",
		})
	}

	var req PingRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[ping] Failed to parse JSON body: %v, body: %s\n", err, string(body))
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid JSON in request body",
		})
	}

	// Get client info
	userAgent := c.Request().Header.Get("User-Agent")
	clientIP := getRealClientIP(c)

	// Hash the IP for privacy
	ipHash := hashIP(clientIP)

	// Create pageview record
	collection, err := h.app.Dao().FindCollectionByNameOrId("pageviews")
	if err != nil {
		log.Printf("[ping] Failed to find pageviews collection: %v\n", err)
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
		log.Printf("[ping] Failed to save pageview: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to record pageview",
		})
	}

	log.Printf("[ping] Recorded pageview for %s: %s\n", domain, req.Path)
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// getRealClientIP extracts the real client IP from request headers
// Checks proxy headers in priority order: Cloudflare, then standard proxy headers
func getRealClientIP(c echo.Context) string {
	// Cloudflare: CF-Connecting-IP is the most reliable for Cloudflare Tunnel
	if ip := c.Request().Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}

	// Standard proxy header: True-Client-IP (used by some CDNs)
	if ip := c.Request().Header.Get("True-Client-IP"); ip != "" {
		return ip
	}

	// X-Real-IP (common proxy header)
	if ip := c.Request().Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// X-Forwarded-For can contain multiple IPs: client, proxy1, proxy2, ...
	// The first one is typically the real client
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		// Split by comma and take the first IP
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Fall back to Echo's RealIP (which also checks some headers)
	return c.RealIP()
}

// hashIP creates a privacy-preserving hash of the IP address
func hashIP(ip string) string {
	// Use a static salt (in production, could use daily rotating salt)
	hash := sha256.Sum256([]byte(ip + "-dingdong"))
	return hex.EncodeToString(hash[:8]) // Only use first 8 bytes
}
