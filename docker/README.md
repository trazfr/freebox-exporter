# Freebox Exporter - Docker

Prometheus exporter for Freebox metrics.

## Quick Start

```bash
docker run -d \
  -p 9091:9091 \
  -v ./data:/app/data \
  ghcr.io/<OWNER>/freebox-exporter:latest
```

Or with docker-compose:

```yaml
services:
  freebox-exporter:
    image: ghcr.io/<OWNER>/freebox-exporter:latest
    ports:
      - "9091:9091"
    volumes:
      - ./data:/app/data
    environment:
      - DEBUG=false
```

> Replace `<OWNER>` with `trazfr` (upstream) or your GitHub username (fork).

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TOKEN_FILE` | `/app/data/token.json` | Token storage path |
| `DEBUG` | `false` | Enable debug logging |
| `HOST_DETAILS` | `false` | Scrape WiFi/Ethernet host details |
| `HTTP_DISCOVERY` | `false` | Force HTTP discovery (auto-enabled on first run) |
| `LISTEN_ADDRESS` | `:9091` | Listen address |

## First Run

On first run, the exporter uses HTTP discovery to find your Freebox and request authorization. Check the logs and authorize on your Freebox.

## Metrics

Available at `http://localhost:9091/metrics`

