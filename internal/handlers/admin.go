package handlers

import (
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
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
func (h *Handlers) HandleDashboard(e *core.RequestEvent) error {
	sites, err := h.app.FindRecordsByFilter("sites", "1=1", "-created", 100, 0)
	if err != nil {
		sites = []*core.Record{}
	}

	data := DashboardData{
		Sites:      make([]SiteSummary, 0, len(sites)),
		TotalSites: len(sites),
		TrackerURL: GetPublicURL(e),
	}

	today := time.Now().UTC().Truncate(24 * time.Hour).Format("2006-01-02 15:04:05")

	for _, site := range sites {
		summary := SiteSummary{
			ID:     site.Id,
			Name:   site.GetString("name"),
			Domain: site.GetString("domain"),
		}

		var totalCount struct {
			Count int `db:"count"`
		}
		err := h.app.DB().
			NewQuery("SELECT COUNT(*) as count FROM pageviews WHERE site = {:siteId}").
			Bind(map[string]any{"siteId": site.Id}).
			One(&totalCount)
		if err == nil {
			summary.Pageviews = totalCount.Count
			data.TotalPageviews += totalCount.Count
		}

		var todayCount struct {
			Count int `db:"count"`
		}
		err = h.app.DB().
			NewQuery("SELECT COUNT(*) as count FROM pageviews WHERE site = {:siteId} AND created >= {:today}").
			Bind(map[string]any{"siteId": site.Id, "today": today}).
			One(&todayCount)
		if err == nil {
			summary.TodayViews = todayCount.Count
		}

		data.Sites = append(data.Sites, summary)
	}

	return h.renderTemplate(e, "dashboard.html", data)
}

// HandleSites renders the sites list page
func (h *Handlers) HandleSites(e *core.RequestEvent) error {
	return h.HandleDashboard(e)
}

// HandleSiteStats renders detailed stats for a specific site
func (h *Handlers) HandleSiteStats(e *core.RequestEvent) error {
	siteId := e.Request.PathValue("siteId")

	site, err := h.app.FindRecordById("sites", siteId)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Site not found",
		})
	}

	data := SiteStatsData{
		Site: SiteSummary{
			ID:     site.Id,
			Name:   site.GetString("name"),
			Domain: site.GetString("domain"),
		},
		TrackerURL: GetPublicURL(e),
	}

	today := time.Now().UTC().Truncate(24 * time.Hour).Format("2006-01-02 15:04:05")

	var totalCount struct {
		Count int `db:"count"`
	}
	err = h.app.DB().
		NewQuery("SELECT COUNT(*) as count FROM pageviews WHERE site = {:siteId}").
		Bind(map[string]any{"siteId": siteId}).
		One(&totalCount)
	if err == nil {
		data.TotalViews = totalCount.Count
	}

	var todayCount struct {
		Count int `db:"count"`
	}
	err = h.app.DB().
		NewQuery("SELECT COUNT(*) as count FROM pageviews WHERE site = {:siteId} AND created >= {:today}").
		Bind(map[string]any{"siteId": siteId, "today": today}).
		One(&todayCount)
	if err == nil {
		data.TodayViews = todayCount.Count
	}

	var uniqueCount struct {
		Count int `db:"count"`
	}
	err = h.app.DB().
		NewQuery("SELECT COUNT(DISTINCT ip_hash) as count FROM pageviews WHERE site = {:siteId} AND ip_hash != ''").
		Bind(map[string]any{"siteId": siteId}).
		One(&uniqueCount)
	if err == nil {
		data.UniqueVisitors = uniqueCount.Count
	}

	var topPages []struct {
		Path  string `db:"path"`
		Views int    `db:"views"`
	}
	err = h.app.DB().
		NewQuery("SELECT path, COUNT(*) as views FROM pageviews WHERE site = {:siteId} GROUP BY path ORDER BY views DESC LIMIT 10").
		Bind(map[string]any{"siteId": siteId}).
		All(&topPages)
	if err == nil {
		data.TopPages = make([]PageStats, len(topPages))
		for i, p := range topPages {
			data.TopPages[i] = PageStats{Path: p.Path, Views: p.Views}
		}
	}

	var topReferrers []struct {
		Referrer string `db:"referrer"`
		Views    int    `db:"views"`
	}
	err = h.app.DB().
		NewQuery("SELECT referrer, COUNT(*) as views FROM pageviews WHERE site = {:siteId} AND referrer != '' GROUP BY referrer ORDER BY views DESC LIMIT 10").
		Bind(map[string]any{"siteId": siteId}).
		All(&topReferrers)
	if err == nil {
		data.TopReferrers = make([]ReferrerStats, len(topReferrers))
		for i, r := range topReferrers {
			data.TopReferrers[i] = ReferrerStats{Referrer: r.Referrer, Views: r.Views}
		}
	}

	var dailyStats []struct {
		Date  string `db:"date"`
		Views int    `db:"views"`
	}
	err = h.app.DB().
		NewQuery("SELECT DATE(created) as date, COUNT(*) as views FROM pageviews WHERE site = {:siteId} GROUP BY DATE(created) ORDER BY date DESC LIMIT 30").
		Bind(map[string]any{"siteId": siteId}).
		All(&dailyStats)
	if err == nil {
		data.DailyStats = make([]DailyStats, len(dailyStats))
		for i, d := range dailyStats {
			data.DailyStats[i] = DailyStats{Date: d.Date, Views: d.Views}
		}
	}

	recentPageviews, err := h.app.FindRecordsByFilter(
		"pageviews",
		"site = {:siteId}",
		"-created",
		20,
		0,
		map[string]any{"siteId": siteId},
	)
	if err == nil {
		data.RecentViews = make([]PageviewRecord, len(recentPageviews))
		for i, pv := range recentPageviews {
			data.RecentViews[i] = PageviewRecord{
				Path:      pv.GetString("path"),
				Referrer:  pv.GetString("referrer"),
				CreatedAt: pv.GetDateTime("created").Time(),
				UserAgent: pv.GetString("user_agent"),
			}
		}
	}

	return h.renderTemplate(e, "site_stats.html", data)
}

// HandleAdmin renders the admin management page
func (h *Handlers) HandleAdmin(e *core.RequestEvent) error {
	return h.renderTemplate(e, "admin.html", nil)
}

// renderTemplate renders an HTML template
func (h *Handlers) renderTemplate(e *core.RequestEvent, name string, data any) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(e.Response, name, data)
}
