# MudLog

Wellsite mud-logging frame relay: rigs post signed WITS-like lithology and lag-depth frames; the service accepts, journals, and fans out to a downstream geology HTTP store with retry, circuit breaker, rate limit, DLQ, and replay.

Full design: [PROJECT.md](PROJECT.md).

## Run

```text
set GOTOOLCHAIN=local
go test ./...
go run ./cmd/mudlog
```

Open http://127.0.0.1:8080/

Default rig ingest secret: `dev-rig-secret`. A cuttings-sink store is seeded so a test frame can succeed without an external geology URL.
