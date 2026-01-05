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
		return e.NoContent(http.StatusForbidden)
	}

	setCORSHeaders(e, origin)
	return e.NoContent(http.StatusNoContent)
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

	site, err := FindSiteByDomain(h.app, domain)
	if err != nil {
		log.Printf("[ping] Domain not registered: %s\n", domain)
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "Domain not registered",
		})
	}

	setCORSHeaders(e, origin)

	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		log.Printf("[ping] Failed to read body: %v\n", err)
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to read request body",
		})
	}

	if len(body) == 0 {
		log.Println("[ping] Empty request body")
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Empty request body",
		})
	}

	var req PingRequest
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
