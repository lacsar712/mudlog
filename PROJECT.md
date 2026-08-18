# MudLog

Wellsite **mud-logging frame relay**: receive signed WITS-like lithology and lag-depth frames from the rig, persist a journal, and fan them out to one or more downstream geology HTTP stores with retry, circuit breaking, token-bucket rate limits, dead-letter, and operator replay.

This repository is a **runnable healthy product**. It does not ship pre-buried defects on `main`.

## 1. Why this product

This is **outbound geology-store delivery infrastructure**, not:

- IM / chat
- e-commerce, RBAC, inventory, OA
- hospital appointments / 预约
- a dashboard-only stats product
- games, CLI file tools, CRM, tickets, parking, auctions

Product boundary: a frame arrives from the wellsite, is signature-checked and routed, and this process is responsible for **POSTing it to someone else's geology HTTP API**. The operator console only operates relay itself (stores, attempts, replay). It is not a lithology analytics product.

## 2. Roles and happy path

| Role | What they do |
|------|----------------|
| Rig / mud logger | POST `/api/v1/frames` with HMAC headers, lithology, lag depth |
| Geology store | An HTTP URL registered by the operator; receives JSON WITS-like callbacks |
| Operator | Opens `/`, registers stores, watches the journal, replays failures |

Happy path:

1. Operator registers a geology store (URL + outbound secret + frame-type prefixes).
2. The rig posts a frame (type, wellbore, lithology, lag depth, extra WITS payload).
3. The engine verifies the signature window, nonce, and idempotency key, matches stores, enqueues per store.
4. Workers POST to each store with backoff; successes are journaled; retryable failures re-queue; terminal failures go to DLQ.
5. The operator sees attempts on the page and can replay a failed relay.

## 3. Business rules

### 3.1 Inbound signature

- Algorithm: `HMAC-SHA256(secret, canonical)`, lowercase hex.
- Canonical string: `v1.{timestamp}.{nonce}.{sha256_hex(raw_body)}`.
- Headers:
  - `X-Mud-Timestamp`: Unix seconds
  - `X-Mud-Nonce`: 16–64 printable bytes
  - `X-Mud-Signature`: `v1=<hex>`
- Window: default ±300 seconds; outside the window is rejected (anti-replay).
- The same nonce may succeed only once inside the window.
- Rig ingest secrets and geology-store outbound secrets are separate.

### 3.2 Idempotency

- Caller must send `Idempotency-Key` (8–128 characters).
- Same key + same body hash: return the first accept result, do not re-enqueue.
- Same key + different body hash: 409 Conflict.
- Records have TTL (default 24h); after expiry the key may be reused.

### 3.3 Fan-out to geology stores

- A store has: id, name, URL, outbound secret, enable flag, type-prefix list, per-store concurrency, token bucket.
- Frame `type` must hit a prefix (`lithology.` matches `lithology.cuttings`) before relay.
- One inbound frame may fan out to several stores; each store has its own queue and journal rows.
- `ordered=true` serializes relays on that store.

### 3.4 Outbound WITS POST

- Method POST, `Content-Type: application/json`.
- Outbound signature uses the **store** secret, same algorithm as inbound.
- Extra headers: `X-Mud-Frame-Id`, `X-Mud-Relay-Id`, `X-Mud-Attempt`, `X-Mud-Store`.
- Timeout: default 10s; do not follow cross-host 3xx (at most one same-host 307/308).
- 2xx is success.
- Retryable: 408, 429, 500–599, network errors, timeouts.
- Terminal: 400–407, 409–428, 430–499 (including 422) → DLQ.

### 3.5 Retry

- Full-jitter exponential backoff: `sleep = random(0, min(cap, base * 2^attempt))`.
- Default `base=200ms`, `cap=30s`, `maxAttempts=8` (including the first try).
- Attempt numbers written to the journal start at 1.

### 3.6 Circuit breaker

Per store:

- Closed: consecutive failures ≥ `failThreshold` (default 5) → Open.
- Open: reject new attempts for `openFor` (default 30s), then HalfOpen.
- HalfOpen: allow `probe` (default 1) probes; success → Closed and clear count; failure → Open again.

While Open the job stays in queue; it is not a business failure attempt, but the journal records `skipped_open`.

### 3.7 Rate limit

Per-store token bucket: capacity `burst`, refill `rate` tokens/second. No token → delay and retry without consuming an attempt.

### 3.8 Dead letter and replay

- Terminal outcome or attempts exhausted → DLQ.
- Replay: take the original body from the journal (or DLQ), mint a new `relay_id`, reset attempt to 1. Replay is an internal path and is not blocked by the old idempotency key.

### 3.9 Payload and snapshot

- Body max 256 KiB.
- Console lists mask `authorization` / `password` / `secret` / `token` fields; storage keeps the original for replay.
- On shutdown (and on a ticker) the in-memory store is snapshotted to the data directory.

## 4. Packages

| Package | Role |
|---------|------|
| `cmd/mudlog` | Process entry, signals, wiring |
| `internal/config` | Env + defaults |
| `internal/clock` | Injectable clock |
| `internal/hashutil` | Body hash, canonical string |
| `internal/sign` | HMAC issue and verify |
| `internal/idempotency` | Idempotency records and conflict |
| `internal/nonce` | Nonce book |
| `internal/frame` | Frame / Lithology / LagDepth parse |
| `internal/lithcode` | Cuttings lithology code table |
| `internal/annulus` | Lag-depth from annular volume / strokes |
| `internal/channel` | WITS channel identifiers |
| `internal/wellbore` | Wellbore id rules |
| `internal/wellkey` | Rig source-key registry |
| `internal/geology` | Geology-store CRUD and prefix match |
| `internal/fanout` | Fan-out plan |
| `internal/queue` | Per-store queues |
| `internal/backoff` | Backoff + jitter |
| `internal/circuit` | Breaker |
| `internal/ratelimit` | Token bucket |
| `internal/classify` | HTTP/network retry vs terminal |
| `internal/wits` | Outbound WITS HTTP client |
| `internal/worker` | Lease, breaker, rate limit, POST, journal |
| `internal/journal` | Attempt log |
| `internal/dlq` | Dead letter |
| `internal/replay` | Replay job construction |
| `internal/store` | Snapshot file |
| `internal/ingest` | Signed ingest pipeline |
| `internal/httpapi` | JSON API |
| `internal/web` | Static operator UI |
| `internal/redact` | Console masking |
| `internal/idgen` | Crockford ids |
| `internal/cuttings` | Local cuttings sink for demos |

Tests live next to packages and **do not count** toward production line totals.

## 5. HTTP API

Base URL: `http://127.0.0.1:8080`

### Control plane (unsigned, local demo)

- `GET /api/v1/meta`
- `GET /api/v1/stores`
- `POST /api/v1/stores` body: `{name,url,secret,type_prefixes[],ordered,rate,burst,enabled}`
- `POST /api/v1/stores/{id}/enable` `{enabled:bool}`
- `GET /api/v1/journal?store_id=&limit=`
- `GET /api/v1/dlq`
- `POST /api/v1/replay/{relay_id}`
- `GET /api/v1/healthz`

### Data plane

- `POST /api/v1/frames`
  - Headers: signature triple + `Idempotency-Key` + `X-Mud-Source-Key` (demo default `rig`)
  - Body: `{"type":"lithology.cuttings","wellbore":"W-12H","lithology":{...},"lag_depth":{...},"payload":{...}}`

### Built-in cuttings sink

`POST /api/v1/cuttings` records the last 50 callbacks so the console can show a successful relay. A default store points at this URL.

## 6. Operator UI

Static `web/index.html` + `web/app.js` + `web/style.css` (no npm).

Sections: health, geology stores, send a test frame, journal, DLQ + replay, cuttings sink. All data comes from the API.

## 7. Data flow

```
Rig --sign--> /frames --> parse/validate
                       --> nonce + idempotency
                       --> fanout
                       --> queue.enqueue (per store)
worker --> ratelimit.Take --> circuit.Allow --> wits.POST
      --> journal.append
      --> success: done
      --> retryable: backoff + requeue
      --> terminal: dlq
UI replay --> replay.FromJournal --> same worker path
store.snapshot <-- ticker / shutdown
```

## 8. Run

```text
set GOTOOLCHAIN=local
go test ./...
go run ./cmd/mudlog
```

Browser: `http://127.0.0.1:8080/`.

| Variable | Default |
|----------|---------|
| `MUDLOG_ADDR` | `:8080` |
| `MUDLOG_DATA_DIR` | `./data` |
| `MUDLOG_INGEST_SECRET` | `dev-rig-secret` |
| `MUDLOG_WINDOW_SEC` | `300` |

## 9. Constraints

- Module: `github.com/lacsar712/mudlog`
- `go.mod` language `go 1.22`; run with `GOTOOLCHAIN=local`
- No CGO; snapshot is a pure-Go file
- Standard library only
- Each package has real branches; no empty padding functions
- No Docker files in this tree
