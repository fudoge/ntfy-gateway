# ntfy-gateway

`ntfy-gateway` receives authenticated webhook events, converts them into a
common event format, and publishes notifications to an [ntfy](https://ntfy.sh/)
server.

The gateway currently supports:

- Flux generic webhook events
- A simple generic JSON event format
- Per-source webhook secrets and ntfy topic routing
- ntfy access-token authentication
- Structured JSON logs
- Container images and release binaries published from Git tags

## How it works

Each configured source has a URL, payload type, shared secret, and destination
topic:

```text
POST /api/webhooks/{source}
        │
        ├── authenticate the source
        ├── decode the configured payload type
        ├── convert it to a common event
        └── publish it to the configured ntfy topic
```

The source name in the URL selects the route. A client cannot choose an ntfy
topic through its request body.

## Configuration

The gateway reads a YAML configuration file. It uses `config.yaml` by default,
or the path specified by the `CONFIG_PATH` environment variable.

```yaml
server:
  address: ":8080"

ntfy:
  endpoint: "https://ntfy.sh"
  token_env: "NTFY_TOKEN"

sources:
  flux-production:
    type: "flux"
    topic: "production-alerts"
    secret_env: "FLUX_PRODUCTION_SECRET"

  monitoring:
    type: "generic"
    topic: "monitoring-alerts"
    secret_env: "MONITORING_SECRET"
```

See [`config.example.yaml`](config.example.yaml) for a complete example.

### Configuration schema

| Field | Required | Default | Description |
| --- | --- | --- | --- |
| `server.address` | No | `:8080` | HTTP listen address. |
| `ntfy.endpoint` | Yes | — | Base HTTP or HTTPS URL of the ntfy server. |
| `ntfy.token_env` | No | — | Environment variable holding the ntfy token. |
| `sources` | Yes | — | Map containing at least one source definition. |
| `sources.<name>.type` | Yes | — | Payload decoder: `flux` or `generic`. |
| `sources.<name>.topic` | Yes | — | Destination ntfy topic. |
| `sources.<name>.secret_env` | Yes | — | Env var holding the source secret. |

Source names may contain letters, numbers, hyphens, and underscores. Topic
names follow ntfy's rules: 1–64 letters, numbers, hyphens, or underscores.

Configuration parsing is strict. Unknown fields, multiple YAML documents,
invalid URLs, unsupported source types, and missing referenced environment
variables cause startup to fail.

## Environment variables

| Variable | When needed | Description |
| --- | --- | --- |
| `CONFIG_PATH` | Optional | Configuration file path. |
| `NTFY_TOKEN` | Referenced | Example ntfy access token. |
| `FLUX_PRODUCTION_SECRET` | Referenced | Example source secret. |
| `MONITORING_SECRET` | Referenced | Example source secret. |

`CONFIG_PATH` defaults to `config.yaml`. The container changes the default to
`/etc/ntfy-gateway/config.yaml`.

`token_env` and `secret_env` contain environment variable names, not secret
values. The names are arbitrary; they only need to match variables present in
the process environment.

If the ntfy server allows anonymous publishing, omit `ntfy.token_env` entirely:

```yaml
ntfy:
  endpoint: "https://ntfy.example.com"
```

Generate a sufficiently long random secret for every webhook source and do not
reuse the ntfy access token as a source secret.

## Running from source

Go 1.26.6 or later is required.

```bash
cp config.example.yaml config.yaml

export NTFY_TOKEN="your-ntfy-token"
export FLUX_PRODUCTION_SECRET="your-flux-production-secret"
export FLUX_STAGING_SECRET="your-flux-staging-secret"
export MONITORING_SECRET="your-monitoring-secret"

go run ./cmd/ntfy-gateway
```

To use a different configuration file:

```bash
CONFIG_PATH=/etc/ntfy-gateway/config.yaml go run ./cmd/ntfy-gateway
```

## Running with Docker

Build the image locally:

```bash
docker build -t ntfy-gateway .
```

Run it with a read-only configuration mount:

```bash
docker run --rm \
  -p 8080:8080 \
  -v "$PWD/config.yaml:/etc/ntfy-gateway/config.yaml:ro" \
  -e NTFY_TOKEN \
  -e FLUX_PRODUCTION_SECRET \
  -e FLUX_STAGING_SECRET \
  -e MONITORING_SECRET \
  ntfy-gateway
```

Published images are available from GitHub Container Registry:

```bash
docker pull ghcr.io/fudoge/ntfy-gateway:latest
```

The runtime image runs as a non-root user and does not contain a shell.

## Running a release binary

Download the archive for your operating system and architecture from the
project's GitHub Releases page, verify it against `checksums.txt`, and run:

```bash
CONFIG_PATH=/path/to/config.yaml ./ntfy-gateway
```

Release archives are produced for:

- Linux amd64 and arm64
- macOS amd64 and arm64
- Windows amd64

## Sending a generic event

Send JSON to the configured source URL using the shared secret as a Bearer
token:

```bash
curl \
  -X POST \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer your-monitoring-secret' \
  -d '{
    "type": "deployment",
    "title": "Production deployment",
    "message": "Version 1.2.3 deployed",
    "severity": "info",
    "timestamp": "2026-09-17T12:00:00Z",
    "metadata": {
      "version": "1.2.3"
    }
  }' \
  http://localhost:8080/api/webhooks/monitoring
```

The generic payload supports the following fields:

| Field | JSON type | Description |
| --- | --- | --- |
| `type` | string | Event type or reason. |
| `title` | string | ntfy notification title. |
| `message` | string | ntfy notification body. |
| `severity` | string | Source-defined severity. |
| `timestamp` | RFC 3339 string | Time at which the event occurred. |
| `metadata` | object of strings | Additional event metadata. |

## Configuring Flux

Configure a Flux generic Provider to send events to the matching gateway source.
The Authorization token must equal the value referenced by the source's
`secret_env` setting.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ntfy-gateway
  namespace: flux-system
stringData:
  headers: |
    Authorization: Bearer your-flux-production-secret
---
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Provider
metadata:
  name: ntfy-gateway
  namespace: flux-system
spec:
  type: generic
  address: https://gateway.example.com/api/webhooks/flux-production
  secretRef:
    name: ntfy-gateway
```

Reference this Provider from a Flux Alert. The Flux decoder uses the involved
object's kind and name as the ntfy title and the Flux event message as the ntfy
message.

## HTTP API

### `GET /health`

Returns `200 OK` with the body `ok`.

### `POST /api/webhooks/{source}`

Required headers:

```http
Content-Type: application/json
Authorization: Bearer <source-secret>
```

Webhook bodies are limited to 1 MiB. Possible responses include:

| Status | Meaning |
| --- | --- |
| `202 Accepted` | ntfy accepted the notification. |
| `400 Bad Request` | The payload could not be decoded. |
| `401 Unauthorized` | Unknown source or invalid secret. |
| `413 Content Too Large` | The request exceeded the 1 MiB limit. |
| `415 Unsupported Media Type` | The request was not JSON. |
| `502 Bad Gateway` | Publishing to ntfy failed. |

## Security notes

- Put the gateway behind HTTPS. The application does not terminate TLS itself.
- Keep secret values in environment variables or a secrets manager, not in the
  YAML file.
- Use a different shared secret for every source.
- Do not expose an ntfy access token to webhook senders.
- The gateway performs synchronous delivery and currently has no persistent
  queue or retry mechanism.

## Publishing

Pushes to `main` build and scan the container image before replacing the
`latest` tag in GHCR. A Git tag push publishes both the matching image tag and
`latest`, then creates a GitHub Release containing binaries and SHA-256
checksums. HIGH or CRITICAL Trivy findings stop publication.

## License

Licensed under the Apache License 2.0. See [`LICENSE`](LICENSE).
