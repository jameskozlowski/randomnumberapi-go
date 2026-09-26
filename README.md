# Random Number API

A Go HTTP API for pseudorandom numbers, strings, and cryptographically random UUIDv4 values. The optional Reddit-seeded endpoint uses public comment text as a deterministic seed; it is **not** cryptographically secure and depends on Reddit being available.

## Run

Requires Go 1.27.1 or later. From the repository root:

```sh
go run ./cmd/api -addr 127.0.0.1:4000
```

The default address is `:4000`. The address flag is parsed at startup. Logs are JSON on stdout; each request has an `X-Request-ID` response header and a log entry with method, route, status, duration, and request ID. Errors include their underlying cause only in server logs, not in API responses.

```sh
docker build -f docker/Dockerfile -t randomnumberapi .
docker run --rm -p 4000:4000 randomnumberapi
```

The container uses a non-root, minimal runtime image and serves the static API page at `/`. Page examples use the browser's current origin: `https://www.randomnumberapi.com` when hosted there and the local host/port when running locally. The hosted address is present in the HTML as a fallback without JavaScript. `GET /healthz` returns 204 when the HTTP process is serving; it does not test Reddit availability. SIGINT/SIGTERM starts graceful shutdown. The server limits request/header, response, and idle time; Reddit fetches have a five-second timeout and respect request cancellation.

## API

All endpoints use GET (HEAD is also supported). Successful GET calls return a JSON array, including when `count=1`. `count` defaults to 1 and must be 1–100. Each endpoint also accepts a trailing slash. Other methods return 405. Invalid or repeated documented options, or out-of-range values, return HTTP 400 and `{"error":"..."}`; failures generating values return HTTP 500, and Reddit upstream failures return HTTP 502 with the same JSON error shape. No partially generated array is returned on failure.

| Endpoint | Options | Defaults |
| --- | --- | --- |
| `/api/v1.0/random` (`/randomnumber`) | `min` ≥ 0, `max` > `min`, `secure=true/false`, `count` | `min=0`, `max=min+100`, `secure=false` |
| `/api/v1.0/randomstring` | `min` ≥ 0, `max` > `min` and ≤ 1001, `all=true/false`, `secure=true/false`, `count` | `min=10`, `max=min+10`, `all=false`, `secure=false` |
| `/api/v1.0/uuid` (`/randomuuid`) | `count` | 1 |
| `/api/v1.0/randomredditnumber` | `min`, `max`, `count` as for `/random`; `secure=true` is rejected | `min=0`, `max=min+100`, `count=1` |

`max` is **exclusive**: numbers are in `[min,max)` and string lengths are in `[min,max)`. String length may be 1000 with `min=1000&max=1001`. Defaults that would overflow or exceed a permitted maximum are rejected; explicitly supply a valid `max` instead. `all=true` adds digits and `!@#$%^&*` to the ASCII letter alphabet.

The default number/string mode uses `math/rand/v2` and must not be used for secrets. `secure=true` uses `crypto/rand` for **both** number values and string lengths/characters with unbiased sampling. UUIDs are random UUIDv4 values. Neither `secure=true` nor UUID mode implies a guarantee that the Reddit endpoint has; comment text and its deterministic hash are predictable.

```sh
curl 'http://localhost:4000/api/v1.0/random?min=100&max=1000&count=5'
curl 'http://localhost:4000/api/v1.0/randomstring?min=24&max=25&count=2&all=true&secure=true'
curl 'http://localhost:4000/healthz'
```

**Compatibility note:** Previously invalid query parameters silently used defaults. They now return 400 on the existing v1.0 routes; update callers that relied on that behavior. This is an intentional behavior change without a new route prefix. `max` was already exclusive, and the endpoint aliases remain available.

## Development

```sh
go test -race ./...
go vet ./...
go build ./...
```

CI also runs `govulncheck`. Reddit tests use a local HTTP server and do not require public Reddit access. The only third-party runtime dependency is `github.com/google/uuid` v1.6.0.