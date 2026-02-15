# iDRAC6 Manager

Web-based management for Dell iDRAC6 controllers. Replaces the fragile docker-idrac6 container.

## Quick Start

```bash
go run ./cmd/server --host 192.168.1.172 --user root --pass calvin
# Open http://localhost:8080
```

## Build & Run

```bash
go build -o idrac6-manager ./cmd/server
./idrac6-manager --host 192.168.1.172

# Docker
docker-compose up
```

## Architecture

- **Go backend** with chi router, embedded web UI via `embed.FS`
- **iDRAC6 API**: XML-based REST at `/data?get=...` and `/data?set=...`
- **Auth**: Cookie-based (`_appwebSessionId_`) with ST1/ST2 tokens for firmware >=2.92
- **Virtual Media**: via SSH/RACADM (`racadm remoteimage`)
- **IPMI**: Pure Go via `bougou/go-ipmi` (SOL, chassis control)

## Project Structure

```
cmd/server/         Entry point
internal/idrac/     iDRAC6 REST client (auth, power, sensors, sysinfo, SEL, virtual media)
internal/ipmi/      IPMI 2.0 client
internal/ssh/       SSH/RACADM executor
internal/api/       HTTP API (chi router, handlers, middleware)
web/static/         Web UI (vanilla JS SPA)
```

## Testing

```bash
go test ./...                     # Unit tests
IDRAC_LIVE_TEST=true go test ./... # Integration tests against live iDRAC
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `IDRAC_HOST` | iDRAC IP address | (required) |
| `IDRAC_USER` | Username | `root` |
| `IDRAC_PASS` | Password | `calvin` |
| `IDRAC_API_KEY` | API key for auth | (none) |
