# PrintFarm

**Maker Project Management + Print Farm OS**

A project-centric platform for makers and small 3D print farms. Unlike traditional printer-centric tools (OctoPrint, SimplyPrint), PrintFarm treats **projects as the command center** — you control printers from within your project context.

## Features

- **Project-Centric Workflow**: Organize work by projects, not printers
- **Multi-Printer Control**: Manage OctoPrint, Bambu Lab, and Klipper printers from one interface
- **Design Versioning**: Track every iteration of your designs
- **Material & Cost Tracking**: Know the true cost of every project
- **Real-Time Status**: Live printer status with WebSocket updates

## Tech Stack

- **Backend**: Go 1.22+ (chi router, pgx/PostgreSQL)
- **Frontend**: React 18, TypeScript, Tailwind CSS, TanStack Query
- **Database**: PostgreSQL 16+
- **Real-time**: WebSocket for live printer status

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- PostgreSQL 16+ (or Docker)

### 1. Start PostgreSQL

```bash
# Using Docker
make docker-db

# Or use your existing PostgreSQL
```

### 2. Run Migrations

```bash
# Set your database URL
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/printfarm?sslmode=disable"

# Run migrations
make migrate-docker  # If using Docker
# or
make migrate         # If using local PostgreSQL
```

### 3. Install Dependencies

```bash
# Backend
go mod download

# Frontend
cd web && npm install
```

### 4. Run Development Servers

```bash
# Run both backend and frontend
make dev

# Or run separately:
make backend   # Go server on :8080
make frontend  # Vite dev server on :5173
```

### 5. Open the App

Navigate to `http://localhost:5173`

## Project Structure

```
├── cmd/server/           # Go entry point
├── internal/
│   ├── api/              # HTTP handlers
│   ├── service/          # Business logic
│   ├── repository/       # Database queries
│   ├── model/            # Domain types
│   ├── printer/          # Printer integrations
│   ├── realtime/         # WebSocket hub
│   └── storage/          # File storage
├── migrations/           # SQL migrations
└── web/                  # React frontend
    ├── src/
    │   ├── api/          # API client
    │   ├── components/   # React components
    │   ├── hooks/        # TanStack Query hooks
    │   ├── pages/        # Route pages
    │   └── types/        # TypeScript types
```

## API Endpoints

### Projects
- `GET /api/projects` - List projects
- `POST /api/projects` - Create project
- `GET /api/projects/:id` - Get project
- `PATCH /api/projects/:id` - Update project

### Parts
- `GET /api/projects/:id/parts` - List parts
- `POST /api/projects/:id/parts` - Create part

### Designs
- `GET /api/parts/:id/designs` - List design versions
- `POST /api/parts/:id/designs` - Upload new design (multipart)
- `GET /api/designs/:id/download` - Download design file

### Printers
- `GET /api/printers` - List printers
- `POST /api/printers` - Register printer
- `GET /api/printers/states` - Get all printer states (real-time)
- `GET /api/printers/:id/state` - Get single printer state

### Print Jobs
- `POST /api/print-jobs` - Create job
- `POST /api/print-jobs/:id/start` - Send to printer
- `POST /api/print-jobs/:id/pause` - Pause print
- `POST /api/print-jobs/:id/resume` - Resume print
- `POST /api/print-jobs/:id/cancel` - Cancel print

### WebSocket
- `GET /ws` - Real-time events (printer status, job updates)

## Printer Integration

### Supported Platforms

| Platform | Status | Protocol |
|----------|--------|----------|
| OctoPrint | ✅ Full | REST + WebSocket |
| Bambu Lab | 🚧 Partial | MQTT/LAN (WIP) |
| Moonraker/Klipper | ✅ Full | REST + WebSocket |
| Manual | ✅ Full | No integration |

### Adding a Printer

1. Go to **Printers** page
2. Click **Add Printer**
3. Enter connection details:
   - **OctoPrint**: `http://192.168.1.x` + API key
   - **Bambu Lab**: Printer IP + access code
   - **Moonraker**: `http://192.168.1.x`
   - **Manual**: No connection (log jobs manually)

## Development

### Build for Production

```bash
make build
```

Produces:
- `bin/server` - Go binary
- `web/dist/` - Static frontend assets

### Run Tests

```bash
make test
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://...` | PostgreSQL connection string |
| `PORT` | `8080` | Backend server port |
| `UPLOAD_DIR` | `./uploads` | File storage directory |
| `VITE_API_URL` | `http://localhost:8080` | API URL for frontend |

## License

MIT
