# iDRAC6 Manager

[![CI](https://github.com/williamzujkowski/idrac6-manager/actions/workflows/ci.yml/badge.svg)](https://github.com/williamzujkowski/idrac6-manager/actions/workflows/ci.yml)

A modern web interface for managing Dell servers with iDRAC6 controllers. Built as a lightweight Go binary with an embedded web UI.

## Features

- **Power Control** - On, off, restart, graceful shutdown, hard reset, NMI
- **Sensor Monitoring** - Real-time temperatures, fan speeds, voltages with status indicators
- **System Information** - BIOS, firmware, hostname, service tag, OS info
- **Virtual Media** - Mount/unmount ISO/IMG from NFS, CIFS, or HTTP via RACADM
- **System Event Log** - View and clear the SEL
- **Multi-Host** - Manage multiple iDRAC6 controllers from one instance

## Quick Start

### Binary

```bash
go install github.com/williamzujkowski/idrac6-manager/cmd/server@latest

idrac6-manager --host <IDRAC_IP> --user root --pass <PASSWORD>
# Open http://localhost:8080
```

### Docker

```bash
cp .env.example .env   # Edit with your iDRAC credentials
docker-compose up
# or
docker run -p 8080:8080 -e IDRAC_HOST=<IP> -e IDRAC_PASS=<PASS> idrac6-manager
```

### From Source

```bash
git clone https://github.com/williamzujkowski/idrac6-manager.git
cd idrac6-manager
go build -o idrac6-manager ./cmd/server
./idrac6-manager --host <IDRAC_IP> --user root --pass <PASSWORD>
```

## Configuration

### Command Line

```
--host       iDRAC host IP (required, or IDRAC_HOST env)
--user       Username (default: root, or IDRAC_USER env)
--pass       Password (required, or IDRAC_PASS env)
--addr       Listen address (default: :8080)
--api-key    API key for authentication (or IDRAC_API_KEY env)
--host-id    Host identifier (default: "default")
--host-name  Display name for the host
```

### Environment Variables

```bash
export IDRAC_HOST=192.168.1.172
export IDRAC_USER=root
export IDRAC_PASS=changeme
export IDRAC_API_KEY=my-secret-key  # optional
```

## API

All endpoints are under `/api/`:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check |
| GET | `/api/hosts` | List configured hosts |
| POST | `/api/hosts` | Add a host at runtime |
| GET | `/api/hosts/:id/power` | Get power state |
| POST | `/api/hosts/:id/power` | Power action (`{"action":"on\|off\|restart\|reset\|nmi\|shutdown"}`) |
| GET | `/api/hosts/:id/sensors` | All sensor readings |
| GET | `/api/hosts/:id/info` | System information |
| GET | `/api/hosts/:id/sel` | System Event Log |
| DELETE | `/api/hosts/:id/sel` | Clear SEL |
| GET | `/api/hosts/:id/virtualmedia` | Virtual media status |
| POST | `/api/hosts/:id/virtualmedia` | Mount image |
| DELETE | `/api/hosts/:id/virtualmedia` | Unmount image |

## Architecture

```
Browser  -->  Go HTTP Server  -->  iDRAC6 REST API (XML over HTTPS)
                              -->  SSH/RACADM (virtual media)
                              -->  IPMI 2.0 (chassis, SOL)
```

The web UI is a vanilla JavaScript SPA embedded in the Go binary via `embed.FS`. No build step, no npm, no node_modules.

### iDRAC6 Auth Flow

1. `POST /data/login` with username/password
2. Extract `_appwebSessionId_` cookie
3. For firmware >=2.92: extract ST1/ST2 tokens from `forwardUrl`
4. Send `Cookie` + `ST2` header on all subsequent requests
5. Auto-retry on 401 (re-login and replay)

## Development

```bash
go test ./...                          # Unit tests
go test -race -v ./...                 # With race detector
IDRAC_LIVE_TEST=true go test ./...     # Integration tests (live iDRAC)
go run ./cmd/server --host <IDRAC_IP> --user root --pass <PASSWORD>
```

## License

MIT
