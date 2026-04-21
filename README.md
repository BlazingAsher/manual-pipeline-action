# manual-pipeline-action

A lightweight HTTP server that lets CI pipelines pause and wait for a human decision. The pipeline posts a question with labelled options, then polls for an answer while a human clicks a button in their browser.

## How it works

1. The `ask` client posts a question to the server and prints a URL to stderr
2. A human opens the URL and clicks a button
3. The client receives the answer and exits with a code based on which option was chosen

## Running the server

```bash
# Copy and fill in the required values
DATABASE_URL=postgres://... BASE_URL=http://your-host AUTH_TOKEN=secret docker compose up --build
```

**Environment variables:**

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | yes | — | Postgres connection string |
| `BASE_URL` | yes | — | Public base URL (used to build links) |
| `AUTH_TOKEN` | yes | — | Token required to create questions |
| `PORT` | no | `8080` | Port to listen on |
| `CLEANUP_INTERVAL` | no | `1h` | How often the cleanup job runs |
| `CLEANUP_MAX_AGE` | no | `24h` | Delete questions older than this |

## Using the client

```bash
# Build
go build -o ask ./cmd/ask

# Example
./ask \
  --server http://your-host \
  --token secret \
  --question "Deploy to production?" \
  --success Deploy \
  --failure Abort \
  --timeout 30m \
  --timeout-exit-code 1
```

**Flags:**

| Flag | Description |
|---|---|
| `--server` | Server base URL |
| `--token` | Auth token |
| `--question` | Question to ask |
| `--success` | Answer that exits 0 (repeatable) |
| `--failure` | Answer that exits 1 (repeatable) |
| `--timeout` | How long to wait (e.g. `10m`, `1h`) |
| `--timeout-exit-code` | Exit code on timeout: `0` or `1` |

The chosen answer is printed to stdout. The question, options, and URL are printed to stderr so they don't interfere with shell capture (`ANSWER=$(./ask ...)`).

**Exit codes:**

| Condition | Exit code |
|---|---|
| `--success` answer received | 0 |
| `--failure` answer received | 1 |
| Timeout | `--timeout-exit-code` |
| Ctrl+C / SIGTERM | 1 |

## Admin

Visit `/admin` in your browser to see all outstanding and answered questions. Log in with the `AUTH_TOKEN`.

## API

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/questions` | Bearer token | Create a question |
| `GET` | `/i/{id}` | — | Browser interaction page |
| `POST` | `/i/{id}/respond` | — | Submit an answer (one-time) |
| `GET` | `/poll/{id}` | — | Poll for an answer |

`POST /questions` request body:
```json
{ "question": "Deploy?", "options": ["Yes", "No"] }
```

Response:
```json
{ "interaction_url": "http://your-host/i/...", "poll_url": "http://your-host/poll/..." }
```

`GET /poll/{id}` returns `{"answer": null}` while waiting, `{"answer": "<answered option>"}` once answered.
