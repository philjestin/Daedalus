# PrintFarm (Daedalus)

**Maker Project Management + Print Farm OS**

A project-centric platform for makers and small 3D print farms. Unlike traditional printer-centric tools (OctoPrint, SimplyPrint), PrintFarm treats **projects as the command center** — you control printers from within your project context.

Runs as a **desktop app** (macOS/Windows via [Wails](https://wails.io)) or as a **self-hosted web app** (Docker / Fly.io).

## Features

- **Project-Centric Workflow**: Organize work by projects, not printers. Track full lifecycle from draft through shipping.
- **Multi-Printer Control**: Manage OctoPrint, Bambu Lab, and Klipper/Moonraker printers from one interface with automatic network discovery.
- **Design Versioning**: Immutable design versions with 3MF file parsing (print time, weight, filament usage).
- **Material & Cost Tracking**: Material catalog, spool inventory, per-project cost rollups, and profit margin calculations.
- **Templates & Recipes**: Create reusable project templates with material requirements and cost estimates.
- **Expense & Sales Tracking**: Log expenses with receipt uploads, track sales by channel, and view profitability analytics.
- **Etsy Integration**: OAuth-based Etsy sync for orders, listings, and inventory.
- **Real-Time Status**: Live printer status and job progress via WebSocket.
- **Analytics Dashboard**: Financial summaries, time-series charts, expense breakdowns, and profit-per-hour metrics.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.24, Chi router |
| Frontend | React 19, TypeScript, Tailwind CSS 4, TanStack Query v5 |
| Database | SQLite (embedded, WAL mode) |
| Real-time | WebSocket |
| Desktop | Wails v2 |
| Build | Vite 7 |
| Deployment | Docker, Fly.io |

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 20+
- (Optional) [Wails CLI](https://wails.io/docs/gettingstarted/installation) for desktop builds

### 1. Install Dependencies

```bash
make deps
```

### 2. Run Development Servers

```bash
# Run backend and frontend together
make dev

# Or run separately:
make backend   # Go server on :8080
make frontend  # Vite dev server on :5173
```

### 3. Open the App

Navigate to `http://localhost:5173`

The SQLite database is created automatically at `~/.daedalus/daedalus.db` with migrations applied on startup.

### Desktop App (Wails)

```bash
# Development with hot reload
wails dev

# Production build
wails build -platform darwin   # macOS
wails build -platform windows  # Windows
```

The built app is output to `build/bin/`.

## Project Structure

```
├── cmd/server/           # Go entry point (standalone server)
├── main.go               # Wails desktop app entry point
├── app.go                # Wails app lifecycle
├── internal/
│   ├── api/              # HTTP handlers & router
│   ├── service/          # Business logic
│   ├── repository/       # Data access layer
│   ├── model/            # Domain types
│   ├── database/         # SQLite init & migrations
│   ├── printer/          # Printer integrations (OctoPrint, Bambu, Moonraker)
│   ├── realtime/         # WebSocket hub
│   ├── storage/          # File storage
│   ├── etsy/             # Etsy API client (OAuth + PKCE)
│   ├── bambu/            # Bambu Lab cloud integration
│   ├── receipt/          # Receipt parsing
│   └── threemf/          # 3MF file format parsing
├── migrations/           # SQL migration files
├── web/                  # React frontend
│   └── src/
│       ├── api/          # API client
│       ├── components/   # Shared React components
│       ├── hooks/        # TanStack Query hooks
│       ├── pages/        # Route pages
│       ├── contexts/     # React contexts
│       ├── types/        # TypeScript types
│       └── lib/          # Utilities
├── Makefile              # Build automation
├── Dockerfile            # Multi-stage production build
├── fly.toml              # Fly.io deployment config
└── wails.json            # Wails desktop app config
```

## API Endpoints

### Projects
- `GET /api/projects` - List projects (filterable by status)
- `POST /api/projects` - Create project
- `GET /api/projects/{id}` - Get project
- `PATCH /api/projects/{id}` - Update project
- `DELETE /api/projects/{id}` - Delete project
- `GET /api/projects/{id}/summary` - Project analytics
- `GET /api/projects/{id}/jobs` - List jobs for project
- `GET /api/projects/{id}/job-stats` - Job statistics
- `POST /api/projects/{id}/start-production` - Start production
- `POST /api/projects/{id}/ready-to-ship` - Mark ready to ship
- `POST /api/projects/{id}/ship` - Mark shipped with tracking

### Parts & Designs
- `GET /api/projects/{id}/parts` - List parts
- `POST /api/projects/{id}/parts` - Create part
- `GET /api/parts/{id}/designs` - List design versions
- `POST /api/parts/{id}/designs` - Upload new design (multipart)
- `GET /api/designs/{id}/download` - Download design file

### Printers
- `GET /api/printers` - List printers
- `POST /api/printers` - Register printer
- `GET /api/printers/states` - Get all printer states
- `GET /api/printers/{id}/state` - Get single printer state
- `GET /api/printers/{id}/jobs` - Printer job history
- `POST /api/printers/discover` - Network discovery

### Print Jobs
- `POST /api/print-jobs` - Create job
- `POST /api/print-jobs/{id}/start` - Send to printer
- `POST /api/print-jobs/{id}/pause` - Pause print
- `POST /api/print-jobs/{id}/resume` - Resume print
- `POST /api/print-jobs/{id}/cancel` - Cancel print
- `POST /api/print-jobs/{id}/outcome` - Record result
- `POST /api/print-jobs/{id}/retry` - Retry failed job
- `GET /api/print-jobs/{id}/events` - Immutable event log
- `GET /api/print-jobs/{id}/retry-chain` - Related retries

### Materials & Spools
- `GET /api/materials` - List materials
- `POST /api/materials` - Create material
- `GET /api/spools` - List spools
- `POST /api/spools` - Create spool

### Templates
- `GET /api/templates` - List templates
- `POST /api/templates` - Create template
- `POST /api/templates/{id}/instantiate` - Create project from template
- `GET /api/templates/{id}/cost-estimate` - Price calculation

### Expenses & Sales
- `GET /api/expenses` - List expenses
- `POST /api/expenses/receipt` - Upload receipt
- `GET /api/sales` - List sales
- `POST /api/sales` - Record sale

### Analytics
- `GET /api/stats/financial` - Revenue, costs, margins
- `GET /api/stats/time-series` - Dashboard chart data
- `GET /api/stats/expenses-by-category` - Cost breakdown
- `GET /api/stats/sales-by-channel` - Revenue by channel

### Etsy Integration
- `GET /api/integrations/etsy/auth` - Start OAuth flow
- `GET /api/integrations/etsy/status` - Connection status
- `POST /api/integrations/etsy/receipts/sync` - Sync orders
- `GET /api/integrations/etsy/listings` - List/link products

### WebSocket
- `GET /ws` - Real-time events (printer status, job updates, print progress)

## Printer Integration

### Supported Platforms

| Platform | Protocol | Features |
|----------|----------|----------|
| OctoPrint | REST API | Full control, file upload, status polling |
| Bambu Lab | MQTT (LAN + Cloud) | Status streaming, device pairing |
| Moonraker/Klipper | REST API | Full control, timelapse, temperature |
| Manual | None | Log jobs manually |

### Adding a Printer

1. Go to **Printers** page
2. Click **Add Printer** or use **Discover** to scan your network
3. Enter connection details:
   - **OctoPrint**: `http://<ip>` + API key
   - **Bambu Lab**: Printer IP + access code (or cloud pairing)
   - **Moonraker**: `http://<ip>`
   - **Manual**: No connection required

## Development

### Build for Production

```bash
make build
```

Produces:
- `bin/server` - Standalone Go binary
- `web/dist/` - Static frontend assets

### Run Tests

```bash
make test
```

### Docker

```bash
docker build -t printfarm .
docker run -p 8080:8080 -v uploads:/app/uploads printfarm
```

### Deploy to Fly.io

```bash
fly deploy
```

Configuration is in `fly.toml`. Uploads are persisted via a mounted volume.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_PATH` | `~/.daedalus/daedalus.db` | SQLite database file path |
| `PORT` | `8080` | Backend server port |
| `UPLOAD_DIR` | `./uploads` | File storage directory |
| `STATIC_DIR` | `./web/dist` | Frontend static files |
| `REQUIRE_AUTH` | `false` | Enable authentication |
| `JWT_SECRET` | - | Secret for JWT signing (required if auth enabled) |
| `ETSY_CLIENT_ID` | - | Etsy OAuth app ID |
| `ETSY_REDIRECT_URI` | `http://localhost:8080/api/integrations/etsy/callback` | Etsy OAuth callback |
| `FRONTEND_URL` | `http://localhost:5173` | Frontend URL (for CORS in dev) |
| `VITE_API_URL` | `http://localhost:8080` | API URL for frontend |
| `VITE_REQUIRE_AUTH` | `false` | Show login UI in frontend |

## License

MIT
