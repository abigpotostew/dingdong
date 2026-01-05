package handlers

import (
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// FindSiteByDomain finds an active site that matches the given domain.
// It checks the primary domain field first, then checks the additional_domains field.
// The additional_domains field is a comma-separated list of additional domains/subdomains.
func FindSiteByDomain(app *pocketbase.PocketBase, domain string) (*core.Record, error) {
	// Normalize the domain (lowercase, trim whitespace)
	domain = strings.ToLower(strings.TrimSpace(domain))

	// First, try to find by primary domain (exact match)
	site, err := app.FindFirstRecordByFilter("sites", "domain = {:domain} && active = true", map[string]any{
		"domain": domain,
	})
	if err == nil {
		return site, nil
	}

	// If not found by primary domain, search all active sites and check additional_domains
	sites, err := app.FindRecordsByFilter("sites", "active = true", "-created", 1000, 0)
	if err != nil {
		return nil, err
	}

	for _, site := range sites {
		additionalDomains := site.GetString("additional_domains")
		if additionalDomains == "" {
			continue
		}

		// Parse comma-separated list
		domains := strings.Split(additionalDomains, ",")
		for _, d := range domains {
			d = strings.ToLower(strings.TrimSpace(d))
			if d == domain {
				return site, nil
			}
		}
	}

	return nil, err // Return the original error (not found)
}

// ExtractDomain removes port from host if present
func ExtractDomain(host string) string {
	// Remove port if present for domain matching
	if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
		// Check if this is not an IPv6 address
		if !strings.Contains(host, "]") || strings.LastIndex(host, "]") < colonIdx {
			return host[:colonIdx]
		}
	}
	return host
}
