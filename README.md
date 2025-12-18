# DingDong

A privacy-friendly web analytics tracker built with Go and Pocketbase.

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

## Setup

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
<script src="https://stats.example.com/tracker.js" data-endpoint="https://custom.example.com" async></script>
```

This is useful when serving the tracker script from a CDN or different domain than the API.

## Architecture

```
dingdong/
├── main.go                     # Entry point
├── internal/
│   ├── app/
│   │   ├── app.go              # Pocketbase setup and routing
│   │   ├── templates/          # HTML templates for dashboard
│   │   └── static/             # Static files
│   ├── handlers/
│   │   ├── handlers.go         # Handler struct
│   │   ├── ping.go             # Ping API endpoint
│   │   ├── admin.go            # Dashboard handlers
│   │   └── tracker.go          # JavaScript tracker endpoint
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

## Database Schema

### Sites Collection

| Field | Type | Description |
|-------|------|-------------|
| domain | text | The registered domain for CORS validation |
| name | text | Friendly name for the site |
| active | bool | Whether tracking is enabled |

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
go run . serve --http=0.0.0.0:8090

# Build for production
go build -o dingdong -ldflags="-s -w" .
```

## Privacy

DingDong is designed with privacy in mind:

- **No Cookies**: Tracking works without cookies
- **IP Hashing**: IP addresses are hashed before storage
- **No Personal Data**: No personally identifiable information is collected
- **Self-Hosted**: Your data stays on your server

## License

MIT License - see LICENSE file for details.

