(function() {
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
})();
