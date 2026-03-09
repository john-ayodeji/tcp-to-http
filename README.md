# HTTP From TCP

A from-scratch HTTP/1.1 server built on raw TCP sockets in Go — no `net/http` server, no frameworks. 

## Overview

This project implements an HTTP/1.1 server by manually parsing requests and constructing responses at the byte level over TCP connections. It supports:

- **Request parsing** — Request line, headers, and body parsed incrementally from a chunked byte stream
- **Response writing** — Status lines, headers, body, chunked transfer encoding, and trailers
- **Reverse proxy** — Proxies requests to [httpbin.org](https://httpbin.org) with chunked streaming and SHA-256 trailer verification
- **Binary responses** — Serves video files with correct `Content-Type`
- **Graceful shutdown** — Signal handling for clean server termination

## Project Structure

```
http-from-tcp/
├── cmd/
│   ├── httpserver/        # Main HTTP server application
│   ├── tcplistener/       # Raw TCP listener (early prototype)
│   └── udpsender/         # UDP sender utility
├── internal/
│   ├── headers/           # HTTP header parsing & manipulation
│   ├── request/           # HTTP request parser (request line, headers, body)
│   ├── response/          # HTTP response writer (status, headers, body, chunked, trailers)
│   └── server/            # TCP server with handler routing
└── assets/                # Static assets (e.g. video files, gitignored)
```

## Getting Started

### Prerequisites

- Go 1.25+

### Build & Run

```bash
# Download video asset (optional, for /video endpoint)
mkdir -p assets
curl -o assets/vim.mp4 https://storage.googleapis.com/qvault-webapp-dynamic-assets/lesson_videos/vim-vs-neovim-prime.mp4

# Run the HTTP server
go run ./cmd/httpserver/
```

The server starts on port **42069**.

### Run Tests

```bash
go test ./... -v
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Returns an HTML success page (200) |
| `GET` | `/yourproblem` | Returns an HTML error page (400) |
| `GET` | `/myproblem` | Returns an HTML error page (500) |
| `GET` | `/video` | Serves `assets/vim.mp4` as `video/mp4` |
| `GET` | `/httpbin/*` | Proxies to `https://httpbin.org/*` with chunked transfer encoding and trailers |

### Examples

```bash
# HTML response
curl http://localhost:42069/

# Video file
curl http://localhost:42069/video --output vim.mp4

# Proxy to httpbin (chunked + trailers)
curl -v http://localhost:42069/httpbin/html

# See raw chunked response with netcat
echo -e "GET /httpbin/stream/5 HTTP/1.1\r\nHost: localhost:42069\r\nConnection: close\r\n\r\n" | nc localhost 42069
```

## Internal Packages

### `internal/request`

Incremental HTTP/1.1 request parser using a state machine:

- `stateInitialized` → `stateParsingHeaders` → `stateParsingBody` → `stateDone`
- Handles chunked reads (tested with 1-byte-at-a-time reads)
- Validates method (uppercase alpha only) and HTTP version (1.1 only)
- Parses `Content-Length` for body reading

### `internal/headers`

HTTP header map (`map[string]string`) with:

- Case-insensitive key storage (lowercased)
- RFC 9110 token character validation
- Duplicate header merging (comma-separated)
- `Get`, `Set`, and `Parse` methods

### `internal/response`

HTTP response `Writer` with state-enforced ordering:

- `WriteStatusLine` → `WriteHeaders` → `WriteBody` (or `WriteChunkedBody`* → `WriteChunkedBodyDone` → `WriteTrailers`)
- Supports status codes: 200, 400, 500
- Chunked transfer encoding with hex-encoded chunk sizes
- HTTP trailers after final chunk

### `internal/server`

TCP server that:

- Accepts connections in a loop, handles each in a goroutine
- Parses requests with `request.RequestFromReader`
- Delegates to a `Handler` function: `func(w *response.Writer, req *request.Request)`
- Graceful shutdown via `atomic.Bool` flag

