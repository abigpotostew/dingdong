package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/models"
)

// DashboardData contains data for the main dashboard view
type DashboardData struct {
	Sites          []SiteSummary
	TotalPageviews int
	TotalSites     int
	TrackerURL     string
}

// SiteSummary contains summary stats for a site
type SiteSummary struct {
	ID         string
	Name       string
	Domain     string
	Pageviews  int
	TodayViews int
}

// SiteStatsData contains detailed stats for a single site
type SiteStatsData struct {
	Site           SiteSummary
	TopPages       []PageStats
	TopReferrers   []ReferrerStats
	DailyStats     []DailyStats
	RecentViews    []PageviewRecord
	TotalViews     int
	TodayViews     int
	UniqueVisitors int
	TrackerURL     string
}

// PageStats represents stats for a single page
type PageStats struct {
	Path  string
	Views int
}

// ReferrerStats represents stats for a referrer
type ReferrerStats struct {
	Referrer string
	Views    int
}

// DailyStats represents daily pageview counts
type DailyStats struct {
	Date  string
	Views int
}

// PageviewRecord represents a single pageview for display
type PageviewRecord struct {
	Path      string
	Referrer  string
	CreatedAt time.Time
	UserAgent string
}

// HandleDashboard renders the main dashboard showing all sites
func (h *Handlers) HandleDashboard(c echo.Context) error {
	// Get all sites
	sites, err := h.app.Dao().FindRecordsByFilter("sites", "1=1", "-created", 100, 0)
	if err != nil {
		sites = []*models.Record{}
	}

	data := DashboardData{
		Sites:      make([]SiteSummary, 0, len(sites)),
		TotalSites: len(sites),
		TrackerURL: GetPublicURL(c),
	}

	// Get pageview counts for each site
	for _, site := range sites {
		summary := SiteSummary{
			ID:     site.Id,
			Name:   site.GetString("name"),
			Domain: site.GetString("domain"),
		}

		// Count total pageviews for this site
		pageviews, err := h.app.Dao().FindRecordsByExpr("pageviews", dbx.HashExp{"site": site.Id})
		if err == nil {
			summary.Pageviews = len(pageviews)
			data.TotalPageviews += len(pageviews)

			// Count today's views
			today := time.Now().Truncate(24 * time.Hour)
			for _, pv := range pageviews {
				if pv.Created.Time().After(today) {
					summary.TodayViews++
				}
			}
		}

		data.Sites = append(data.Sites, summary)
	}

	return h.renderTemplate(c, "dashboard.html", data)
}

// HandleSites renders the sites list page
func (h *Handlers) HandleSites(c echo.Context) error {
	return h.HandleDashboard(c)
}

// HandleSiteStats renders detailed stats for a specific site
func (h *Handlers) HandleSiteStats(c echo.Context) error {
	siteId := c.PathParam("siteId")

	// Get the site
	site, err := h.app.Dao().FindRecordById("sites", siteId)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Site not found",
		})
	}

	// Get all pageviews for this site
	pageviews, err := h.app.Dao().FindRecordsByExpr("pageviews", dbx.HashExp{"site": siteId})
	if err != nil {
		pageviews = []*models.Record{}
	}

	data := SiteStatsData{
		Site: SiteSummary{
			ID:     site.Id,
			Name:   site.GetString("name"),
			Domain: site.GetString("domain"),
		},
		TotalViews: len(pageviews),
		TrackerURL: GetPublicURL(c),
	}

	// Calculate stats
	pageCounts := make(map[string]int)
	referrerCounts := make(map[string]int)
	dailyCounts := make(map[string]int)
	uniqueIPs := make(map[string]bool)
	today := time.Now().Truncate(24 * time.Hour)

	for _, pv := range pageviews {
		path := pv.GetString("path")
		referrer := pv.GetString("referrer")
		created := pv.Created.Time()
		ipHash := pv.GetString("ip_hash")

		pageCounts[path]++

		if referrer != "" {
			referrerCounts[referrer]++
		}

		dateStr := created.Format("2006-01-02")
		dailyCounts[dateStr]++

		if ipHash != "" {
			uniqueIPs[ipHash] = true
		}

		if created.After(today) {
			data.TodayViews++
		}
	}

	data.UniqueVisitors = len(uniqueIPs)

	// Convert to sorted slices (top 10)
	data.TopPages = topN(pageCounts, 10)
	data.TopReferrers = topNReferrers(referrerCounts, 10)
	data.DailyStats = sortedDailyStats(dailyCounts, 30)

	// Get recent pageviews (last 20)
	data.RecentViews = make([]PageviewRecord, 0, 20)
	count := 0
	for i := len(pageviews) - 1; i >= 0 && count < 20; i-- {
		pv := pageviews[i]
		data.RecentViews = append(data.RecentViews, PageviewRecord{
			Path:      pv.GetString("path"),
			Referrer:  pv.GetString("referrer"),
			CreatedAt: pv.Created.Time(),
			UserAgent: pv.GetString("user_agent"),
		})
		count++
	}

	return h.renderTemplate(c, "site_stats.html", data)
}

// renderTemplate renders an HTML template
func (h *Handlers) renderTemplate(c echo.Context, name string, data any) error {
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(c.Response(), name, data)
}

// topN returns the top N pages by view count
func topN(counts map[string]int, n int) []PageStats {
	result := make([]PageStats, 0, len(counts))
	for path, views := range counts {
		result = append(result, PageStats{Path: path, Views: views})
	}

	// Simple bubble sort for small datasets
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Views > result[i].Views {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if len(result) > n {
		result = result[:n]
	}
	return result
}

// topNReferrers returns the top N referrers by view count
func topNReferrers(counts map[string]int, n int) []ReferrerStats {
	result := make([]ReferrerStats, 0, len(counts))
	for referrer, views := range counts {
		result = append(result, ReferrerStats{Referrer: referrer, Views: views})
	}

	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Views > result[i].Views {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if len(result) > n {
		result = result[:n]
	}
	return result
}

// sortedDailyStats returns daily stats sorted by date (last n days)
func sortedDailyStats(counts map[string]int, n int) []DailyStats {
	result := make([]DailyStats, 0, len(counts))
	for date, views := range counts {
		result = append(result, DailyStats{Date: date, Views: views})
	}

	// Sort by date descending
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Date > result[i].Date {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if len(result) > n {
		result = result[:n]
	}
	return result
}
