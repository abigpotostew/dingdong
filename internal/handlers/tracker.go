package handlers

import (
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// trackerScript is the JavaScript tracker that gets embedded on client sites
const trackerScript = `(function() {
  'use strict';
  
  // Find the current script tag to read data attributes
  var scripts = document.getElementsByTagName('script');
  var currentScript = scripts[scripts.length - 1];
  
  // Configuration - check data-endpoint attribute first, then fall back to default
  var endpoint = currentScript.getAttribute('data-endpoint') || '{{ENDPOINT}}';
  // Remove trailing slash if present
  endpoint = endpoint.replace(/\/$/, '');
  
  // Collect page data
  function collectData() {
    return {
      path: window.location.pathname + window.location.search,
      referrer: document.referrer || '',
      screen_width: window.screen.width,
      screen_height: window.screen.height
    };
  }
  
  // Send ping to server
  function sendPing() {
    var data = collectData();
    
    // Use fetch with no credentials to avoid CORS issues
    fetch(endpoint + '/api/ping', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data),
      mode: 'cors',
      credentials: 'omit',
      keepalive: true
    }).catch(function() {
      // Silently fail - don't break the page
    });
  }
  
  // Send ping on page load
  if (document.readyState === 'complete') {
    sendPing();
  } else {
    window.addEventListener('load', sendPing);
  }
  
  // Also track on page visibility change (for SPAs)
  var lastPath = window.location.pathname;
  
  // Check for SPA navigation
  function checkNavigation() {
    if (window.location.pathname !== lastPath) {
      lastPath = window.location.pathname;
      sendPing();
    }
  }
  
  // Listen for history changes (pushState/popState)
  window.addEventListener('popstate', checkNavigation);
  
  // Intercept pushState and replaceState
  var originalPushState = history.pushState;
  var originalReplaceState = history.replaceState;
  
  history.pushState = function() {
    originalPushState.apply(this, arguments);
    checkNavigation();
  };
  
  history.replaceState = function() {
    originalReplaceState.apply(this, arguments);
    checkNavigation();
  };
})();`

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
	script := strings.ReplaceAll(trackerScript, "{{ENDPOINT}}", endpoint)

	e.Response.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	e.Response.Header().Set("Cache-Control", "public, max-age=86400")

	_, err := e.Response.Write([]byte(script))
	return err
}
