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

	"github.com/pocketbase/pocketbase/core"
)

// PingRequest represents the incoming ping data from the tracker
type PingRequest struct {
	Path         string `json:"path"`
	Referrer     string `json:"referrer"`
	ScreenWidth  int    `json:"screen_width"`
	ScreenHeight int    `json:"screen_height"`
}

// setCORSHeaders sets the CORS headers for the ping endpoint
func setCORSHeaders(e *core.RequestEvent, origin string) {
	e.Response.Header().Set("Access-Control-Allow-Origin", origin)
	e.Response.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	e.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	e.Response.Header().Set("Access-Control-Max-Age", "86400")
}

// HandlePingPreflight handles CORS preflight requests for the ping endpoint
func (h *Handlers) HandlePingPreflight(e *core.RequestEvent) error {
	origin := e.Request.Header.Get("Origin")
	if origin == "" {
		return e.NoContent(http.StatusNoContent)
	}

	// Parse the origin to get the domain
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return e.NoContent(http.StatusNoContent)
	}
	domain := ExtractDomain(parsedOrigin.Host)

	// Look up the site by domain (checks primary domain and additional_domains)
	_, err = FindSiteByDomain(h.app, domain)
	if err != nil {
		h.RecordDeniedPageview(e, domain, origin, "cors_preflight_denied", nil)
		return e.NoContent(http.StatusForbidden)
	}

	setCORSHeaders(e, origin)
	return e.NoContent(http.StatusNoContent)
}

// DeniedPageviewData holds optional data for denied pageview logging
type DeniedPageviewData struct {
	Path         string
	Referrer     string
	ScreenWidth  int
	ScreenHeight int
}

// RecordDeniedPageview logs a denied pageview request to the database
func (h *Handlers) RecordDeniedPageview(e *core.RequestEvent, domain, origin, reason string, data *DeniedPageviewData) {
	collection, err := h.app.FindCollectionByNameOrId("denied_pageviews")
	if err != nil {
		log.Printf("[denied] Failed to find denied_pageviews collection: %v\n", err)
		return
	}

	userAgent := e.Request.Header.Get("User-Agent")
	clientIP := getRealClientIP(e)
	ipHash := hashIP(clientIP)

	record := core.NewRecord(collection)
	record.Set("domain", domain)
	record.Set("origin", origin)
	record.Set("reason", reason)
	record.Set("user_agent", userAgent)
	record.Set("ip_hash", ipHash)

	if data != nil {
		record.Set("path", data.Path)
		record.Set("referrer", data.Referrer)
		record.Set("screen_width", data.ScreenWidth)
		record.Set("screen_height", data.ScreenHeight)
	}

	if err := h.app.Save(record); err != nil {
		log.Printf("[denied] Failed to save denied pageview: %v\n", err)
		return
	}

	log.Printf("[denied] Recorded denied pageview from %s (reason: %s)\n", domain, reason)
}

// HandlePing processes incoming pageview pings from the JavaScript tracker
func (h *Handlers) HandlePing(e *core.RequestEvent) error {
	origin := e.Request.Header.Get("Origin")
	if origin == "" {
		log.Println("[ping] Missing Origin header")
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing Origin header",
		})
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		log.Printf("[ping] Invalid Origin header: %s, error: %v\n", origin, err)
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid Origin header",
		})
	}
	domain := ExtractDomain(parsedOrigin.Host)

	// Read body first so we can log it for denied requests
	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		log.Printf("[ping] Failed to read body: %v\n", err)
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to read request body",
		})
	}

	// Try to parse request body for logging purposes
	var req PingRequest
	if len(body) > 0 {
		json.Unmarshal(body, &req) // Ignore errors, just best-effort parse
	}

	site, err := FindSiteByDomain(h.app, domain)
	if err != nil {
		log.Printf("[ping] Domain not registered: %s\n", domain)
		h.RecordDeniedPageview(e, domain, origin, "domain_not_registered", &DeniedPageviewData{
			Path:         req.Path,
			Referrer:     req.Referrer,
			ScreenWidth:  req.ScreenWidth,
			ScreenHeight: req.ScreenHeight,
		})
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "Domain not registered",
		})
	}

	if site == nil {
		log.Printf("[ping] Site not found: %s\n", domain)
		h.RecordDeniedPageview(e, domain, origin, "site_not_found", &DeniedPageviewData{
			Path:         req.Path,
			Referrer:     req.Referrer,
			ScreenWidth:  req.ScreenWidth,
			ScreenHeight: req.ScreenHeight,
		})
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "not found",
		})
	}

	if len(body) == 0 {
		log.Println("[ping] Empty request body")
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Empty request body",
		})
	}

	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[ping] Failed to parse JSON body: %v, body: %s\n", err, string(body))
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid JSON in request body",
		})
	}

	userAgent := e.Request.Header.Get("User-Agent")
	clientIP := getRealClientIP(e)
	ipHash := hashIP(clientIP)

	collection, err := h.app.FindCollectionByNameOrId("pageviews")
	if err != nil {
		log.Printf("[ping] Failed to find pageviews collection: %v\n", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal error",
		})
	}

	record := core.NewRecord(collection)
	record.Set("site", site.Id)
	record.Set("path", req.Path)
	record.Set("referrer", req.Referrer)
	record.Set("user_agent", userAgent)
	record.Set("ip_hash", ipHash)
	record.Set("screen_width", req.ScreenWidth)
	record.Set("screen_height", req.ScreenHeight)

	if err := h.app.Save(record); err != nil {
		log.Printf("[ping] Failed to save pageview: %v\n", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to record pageview",
		})
	}

	log.Printf("[ping] Recorded pageview for %s: %s\n", domain, req.Path)
	return e.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// getRealClientIP extracts the real client IP from request headers
func getRealClientIP(e *core.RequestEvent) string {
	if ip := e.Request.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := e.Request.Header.Get("True-Client-IP"); ip != "" {
		return ip
	}
	if ip := e.Request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if xff := e.Request.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	return e.RealIP()
}

// hashIP creates a privacy-preserving hash of the IP address
func hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip + "-dingdong"))
	return hex.EncodeToString(hash[:8])
}
