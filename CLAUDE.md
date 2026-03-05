# DingDong — Claude Code Guide

## Project Overview

DingDong is a **privacy-friendly web analytics tracker** built on [Pocketbase](https://pocketbase.io/) (SQLite). It deploys as a single executable and is designed for self-hosted use on small sites.

- **Backend:** Go + Pocketbase framework
- **Database:** SQLite (managed by Pocketbase)
- **Frontend:** Server-side rendered HTML templates + lightweight async JS tracker (no cookies)
- **Deployment:** Single binary or Docker

## Key Commands

```bash
make build          # Build tracker.min.js + Go binary
make build-tracker  # Minify tracker JS only (requires esbuild)
make run            # Build and run server on 0.0.0.0:8090
make build-prod     # Optimized production build (-s -w linker flags)
make clean          # Remove build artifacts

docker compose up --build   # Build and start via Docker
```

## Architecture

```
main.go                              # Entry point; starts Pocketbase app
internal/
  app/
    app.go                           # Pocketbase setup, route registration
    templates/
      dashboard.html                 # Main analytics dashboard
      site_stats.html                # Per-site stats view
      admin.html                     # Admin setup page
    static/
      robots.txt
  handlers/
    handlers.go                      # Handler struct definition
    ping.go                          # POST /api/ping — receive pageviews
    admin.go                         # Dashboard handlers
    sites.go                         # Site management handlers
    tracker.go                       # GET /tracker.js — serve JS tracker
    static/
      tracker.src.js                 # Source JS tracker — EDIT THIS
      tracker.min.js                 # Auto-generated — DO NOT EDIT DIRECTLY
  migrations/
    migrations.go                    # DB schema setup (collections)
scripts/
  build-tracker.sh                   # Shell script; minifies tracker.src.js via esbuild
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Main dashboard |
| GET | `/sites/{siteId}` | Site-specific stats |
| POST | `/api/ping` | Receive pageview data from tracker |
| GET | `/tracker.js` | Serve the JavaScript tracker |
| GET | `/_/` | Pocketbase admin UI |
| GET | `/admin` | Setup page |

## Database Schema

Managed by Pocketbase. Collections defined in `internal/migrations/migrations.go`.

**Sites** — registered domains allowed to track
- `domain`, `name`, `active`, `additional_domains`

**Pageviews** — recorded page visits
- `site`, `path`, `referrer`, `user_agent`, `ip_hash`, `screen` (dimensions), `created`

**Denied Pageviews** — rejected requests with reasons
- Reasons: `cors_preflight_denied`, `cors_post_denied`, `domain_not_registered`, `site_not_found`

## Configuration

| Setting | How |
|---------|-----|
| Public URL | `PUBLIC_URL` environment variable |
| HTTP port | CLI flag (default `8090`) |
| Data directory | CLI flag (default `pb_data/`) |

## Development Notes

- **JS Tracker workflow:** Edit `internal/handlers/static/tracker.src.js`, then run `make build-tracker` to regenerate `tracker.min.js`. Never edit `.min.js` directly.
- **CORS protection:** Only domains registered in the Sites collection are accepted by the ping endpoint.
- **Pocketbase admin UI** is at `/_/` — useful for inspecting/editing DB records directly.
- **First run:** Create an admin account via the Pocketbase UI, then register your domains in the Sites collection before the tracker will accept pageviews.
- **IP privacy:** IPs are hashed before storage — no raw IP addresses are persisted.
- CGO is enabled in the Docker build (required by SQLite driver).
