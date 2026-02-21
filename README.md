# DingDong

A privacy-friendly web analytics tracker using sqlite with an easily deployable single executable. Suitable for small sites.

## Features

- **Lightweight JavaScript Tracker**: Async script that tracks pageviews without cookies
- **Privacy-First**: IP addresses are hashed, no personal data stored
- **SPA Support**: Automatically tracks navigation in single-page applications
- **CORS Protection**: Only registered domains can send analytics data
- **Beautiful Dashboard**: Server-side rendered analytics dashboard
- **Self-Hosted**: Run on your own infrastructure with Docker

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/abigpotostew/dingdong.git
cd dingdong

# Start the server
docker-compose up -d

# Access the dashboard at http://localhost:8090
# Access Pocketbase admin at http://localhost:8090/_/
```

### Using Docker

```bash
docker build -t dingdong .
docker run -d -p 8090:8090 -v dingdong_data:/app/pb_data dingdong
```

### From Source

```bash
# Install dependencies
go mod download

# Run the server
go run . serve --http=0.0.0.0:8090
```

## Usage

### 1. Create Admin Account

Visit `http://localhost:8090/_/` to create your Pocketbase admin account.

### 2. Register a Domain

1. Go to the Pocketbase admin at `http://localhost:8090/_/`
2. Navigate to the `sites` collection
3. Add a new record with:
   - **name**: A friendly name for your site (e.g., "My Blog")
   - **domain**: The domain to track (e.g., `example.com` or `localhost`)
   - **active**: Set to `true` to enable tracking

### 3. Add the Tracker to Your Website

Add this script to your website's `<head>` or before `</body>`:

```html
<script src="https://stats.example.com/tracker.js" async></script>
```

Replace `stats.example.com` with your actual DingDong server URL.

#### Custom Endpoint Override

You can override the tracking endpoint using a `data-endpoint` attribute:

```html
<script src="https://stats.example.com/tracker.js" data-endpoint="https://my-custom-stats.example.com" async></script>
```

This is useful when serving the tracker script from a CDN or different domain than the API.

## Architecture

```
dingdong/
├── main.go                     # Entry point
├── Makefile                    # Build commands
├── scripts/
│   └── build-tracker.sh        # Minifies tracker JavaScript
├── internal/
│   ├── app/
│   │   ├── app.go              # Pocketbase setup and routing
│   │   ├── templates/          # HTML templates for dashboard
│   │   └── static/             # Static files (robots.txt)
│   ├── handlers/
│   │   ├── handlers.go         # Handler struct
│   │   ├── ping.go             # Ping API endpoint
│   │   ├── admin.go            # Dashboard handlers
│   │   ├── tracker.go          # JavaScript tracker endpoint
│   │   └── static/
│   │       ├── tracker.src.js  # Tracker source (edit this)
│   │       └── tracker.min.js  # Minified tracker (generated)
│   └── migrations/
│       └── migrations.go       # Database schema setup
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Main dashboard |
| `/sites/{siteId}` | GET | Site-specific stats |
| `/api/ping` | POST | Receive pageview data |
| `/tracker.js` | GET | JavaScript tracker script |
| `/_/` | GET | Pocketbase admin UI |
| `/admin` | GET | Setup your analytics |

## Database Schema

### Sites Collection

| Field | Type | Description |
|-------|------|-------------|
| domain | text | The primary domain for CORS validation |
| name | text | Friendly name for the site |
| active | bool | Whether tracking is enabled |
| additional_domains | text | Comma-separated list of additional domains/subdomains (e.g., `www.example.com, blog.example.com`) |

### Pageviews Collection

| Field | Type | Description |
|-------|------|-------------|
| site | relation | Reference to the site |
| path | text | The page path visited |
| referrer | text | Referring URL |
| user_agent | text | Browser user agent |
| ip_hash | text | Privacy-preserving hash of IP |
| screen_width | number | Screen width in pixels |
| screen_height | number | Screen height in pixels |
| created | datetime | Timestamp of the pageview |

### Denied Pageviews Collection

Tracks requests from unregistered domains for monitoring and debugging.

| Field | Type | Description |
|-------|------|-------------|
| domain | text | The domain that was denied |
| origin | text | Full origin header from request |
| reason | text | Denial reason (`cors_preflight_denied`, `cors_post_denied`, `domain_not_registered`, `site_not_found`) |
| path | text | Page path (if available) |
| referrer | text | Referring URL (if available) |
| user_agent | text | Browser user agent |
| ip_hash | text | Privacy-preserving hash of IP |
| screen_width | number | Screen width (if available) |
| screen_height | number | Screen height (if available) |
| created | datetime | Timestamp of the denied request |

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PUBLIC_URL` | Public URL where DingDong is accessible (e.g., `https://stats.example.com`) | Auto-detected from request |

### Command-line Flags

DingDong uses Pocketbase's default configuration. You can customize it with command-line flags:

```bash
./dingdong serve \
  --http=0.0.0.0:8090 \
  --dir=/path/to/pb_data
```

### Docker Example

```yaml
environment:
  - PUBLIC_URL=https://stats.example.com
```

## Development

```bash
# Run in development mode
export PUBLIC_URL=0.0.0.0:8090
go run . serve --http=0.0.0.0:8090

# Build for production (includes tracker minification)
make build

# Or build without tracker minification (uses existing tracker.min.js)
go build -o dingdong -ldflags="-s -w" .
```

### Building the Tracker Script

The tracker JavaScript (`internal/handlers/static/tracker.src.js`) is minified using esbuild before being embedded in the Go binary. The minified file (`tracker.min.js`) is committed to the repository.

To rebuild after modifying `tracker.src.js`:

```bash
# Requires esbuild: npm install -g esbuild
make build-tracker

# Or run the script directly
./scripts/build-tracker.sh
```

The minified tracker is ~57% smaller than the source.

## Privacy

DingDong is designed with privacy in mind:

- **No Cookies**: Tracking works without cookies
- **IP Hashing**: IP addresses are hashed before storage
- **No Personal Data**: No personally identifiable information is collected
- **Self-Hosted**: Your data stays on your server

## Features to work on
- Periodic data archiving. Right now all page view data is stored forever. To save on storage space over time, the backend should periodically generate a report and then delete or compress the data.


## License

MIT License - see LICENSE file for details.

