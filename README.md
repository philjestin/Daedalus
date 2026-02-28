# Daedalus

**Print Farm Management + Order Fulfillment Platform**

A comprehensive platform for makers and small 3D print farms. Manage your product catalog, track orders from multiple sales channels, and coordinate print jobs across your printer fleet.

Runs as a **desktop app** (macOS/Windows via [Wails](https://wails.io)) or as a **self-hosted web app** (Docker).

## Features

- **Product Catalog (Projects)**: Define your products with SKUs, pricing, printer constraints, and default settings. Track profitability per product.
- **Order Management**: Unified order system with integrations for Etsy, Squarespace, and manual orders.
- **Task-Based Workflow**: Tasks are work instances created when fulfilling orders. Each task tracks print jobs, progress, and completion.
- **Multi-Printer Control**: Manage OctoPrint, Bambu Lab, and Klipper/Moonraker printers from one interface with automatic network discovery.
- **Design Versioning**: Immutable design versions with 3MF file parsing (print time, weight, filament usage).
- **Material & Cost Tracking**: Material catalog, spool inventory, per-product cost rollups, and profit margin calculations.
- **Expense Tracking**: Log expenses with receipt OCR, categorize by type (filament, tools, advertising, subscriptions), and track profitability.
- **Sales Analytics**: Track sales by channel, view revenue trends, and analyze profit-per-hour metrics.
- **Real-Time Status**: Live printer status and job progress via WebSocket.
- **Timeline View**: Gantt-style visualization of orders, tasks, and print jobs.

## Data Model

```
Sales Channels (Etsy, Squarespace, Direct)
    ↓
Orders → Order Items
    ↓
Tasks (work instances) → Print Jobs
    ↓
Projects (product catalog)
```

- **Projects** = Your product catalog. Each project defines a product you sell (name, SKU, pricing, designs, printer requirements).
- **Tasks** = Work instances created when processing orders. A task references a project and contains print jobs.
- **Print Jobs** = Individual prints sent to printers. Track status, progress, material usage, and outcomes.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.24, Chi router |
| Frontend | React 19, TypeScript, Tailwind CSS 4, TanStack Query v5 |
| Database | SQLite (embedded, WAL mode) |
| Real-time | WebSocket |
| Desktop | Wails v2 |
| Build | Vite 7 |
| Deployment | Docker |

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
│   ├── squarespace/      # Squarespace API client
│   ├── bambu/            # Bambu Lab cloud integration
│   ├── receipt/          # Receipt OCR parsing
│   └── threemf/          # 3MF file format parsing
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
└── wails.json            # Wails desktop app config
```

## API Endpoints

### Projects (Product Catalog)
- `GET /api/projects` - List all products
- `POST /api/projects` - Create product
- `GET /api/projects/{id}` - Get product details
- `PATCH /api/projects/{id}` - Update product (name, SKU, pricing, etc.)
- `DELETE /api/projects/{id}` - Delete product
- `GET /api/projects/{id}/summary` - Product analytics (revenue, costs, profit)
- `GET /api/projects/{id}/tasks` - List tasks for this product

### Tasks (Work Instances)
- `GET /api/tasks` - List tasks (filterable by status, project, order)
- `POST /api/tasks` - Create task manually
- `GET /api/tasks/{id}` - Get task with jobs
- `PATCH /api/tasks/{id}` - Update task
- `PATCH /api/tasks/{id}/status` - Update task status
- `DELETE /api/tasks/{id}` - Delete task
- `POST /api/tasks/{id}/start` - Start task
- `POST /api/tasks/{id}/complete` - Complete task
- `POST /api/tasks/{id}/cancel` - Cancel task
- `GET /api/tasks/{id}/progress` - Get task progress

### Orders
- `GET /api/orders` - List orders (filterable by status, channel)
- `POST /api/orders` - Create manual order
- `GET /api/orders/{id}` - Get order with items and tasks
- `PATCH /api/orders/{id}` - Update order
- `POST /api/orders/{id}/items` - Add item to order
- `POST /api/orders/{id}/items/{itemId}/process` - Create task from order item
- `POST /api/orders/{id}/ship` - Mark order shipped

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

### Materials & Spools
- `GET /api/materials` - List materials
- `POST /api/materials` - Create material
- `GET /api/spools` - List spools
- `POST /api/spools` - Create spool

### Expenses & Sales
- `GET /api/expenses` - List expenses
- `POST /api/expenses/receipt` - Upload receipt (OCR processing)
- `PATCH /api/expenses/{id}` - Update/confirm expense
- `GET /api/sales` - List sales
- `POST /api/sales` - Record sale

### Analytics
- `GET /api/stats/financial` - Revenue, costs, margins
- `GET /api/stats/time-series` - Dashboard chart data
- `GET /api/stats/expenses-by-category` - Cost breakdown
- `GET /api/stats/sales-by-channel` - Revenue by channel
- `GET /api/stats/sales-by-project` - Revenue by product

### Sales Channel Integrations
- `GET /api/channels` - List connected channels
- `GET /api/integrations/etsy/auth` - Start Etsy OAuth flow
- `GET /api/integrations/etsy/status` - Etsy connection status
- `POST /api/integrations/etsy/sync` - Sync Etsy orders
- `GET /api/integrations/squarespace/status` - Squarespace connection status
- `POST /api/integrations/squarespace/sync` - Sync Squarespace orders

### Timeline
- `GET /api/timeline` - Get timeline items (orders, tasks, jobs)
- `GET /api/timeline/orders/{id}` - Order timeline detail
- `GET /api/timeline/tasks/{id}` - Task timeline detail

### WebSocket
- `GET /ws` - Real-time events (printer status, job updates, order sync)

## Printer Integration

### Supported Platforms

| Platform | Protocol | Features |
|----------|----------|----------|
| OctoPrint | REST API | Full control, file upload, status polling |
| Bambu Lab | MQTT (LAN + Cloud) | Status streaming, device pairing, AMS support |
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

## Expense Categories

Track expenses by category for accurate cost analysis:

- **Filament** - Creates spools in inventory
- **Parts** - Replacement parts, hardware
- **Tools** - Equipment, maintenance
- **Shipping** - Packaging, postage
- **Marketplace Fees** - Etsy fees, payment processing
- **Subscriptions** - Software, services
- **Advertising** - Ads, marketing
- **Other** - Miscellaneous

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
docker build -t daedalus .
docker run -p 8080:8080 -v daedalus-data:/app/data daedalus
```

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
| `ETSY_REDIRECT_URI` | - | Etsy OAuth callback URL |
| `SQUARESPACE_API_KEY` | - | Squarespace API key |
| `FRONTEND_URL` | `http://localhost:5173` | Frontend URL (for CORS in dev) |

## License

MIT
