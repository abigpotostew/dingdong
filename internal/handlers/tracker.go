package handlers

import (
	"strings"

	"github.com/labstack/echo/v5"
)

// trackerScript is the JavaScript tracker that gets embedded on client sites
const trackerScript = `(function() {
  'use strict';
  
  // Configuration
  var endpoint = '{{ENDPOINT}}';
  
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

// HandleTrackerScript serves the JavaScript tracker with the correct endpoint
func (h *Handlers) HandleTrackerScript(c echo.Context) error {
	// Determine the endpoint from the request
	scheme := "https"
	if c.Request().TLS == nil {
		forwardedProto := c.Request().Header.Get("X-Forwarded-Proto")
		if forwardedProto != "" {
			scheme = forwardedProto
		} else {
			scheme = "http"
		}
	}
	host := c.Request().Host
	endpoint := scheme + "://" + host

	// Replace the placeholder with the actual endpoint
	script := strings.ReplaceAll(trackerScript, "{{ENDPOINT}}", endpoint)

	c.Response().Header().Set("Content-Type", "application/javascript; charset=utf-8")
	c.Response().Header().Set("Cache-Control", "public, max-age=86400")

	return c.String(200, script)
}
