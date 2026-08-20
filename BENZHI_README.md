# Mudlog

褰曚簳甯?WITS 澶栧彂涓户

## Build

```bash
export GOTOOLCHAIN=local
go build ./...
```

## Test

```bash
export GOTOOLCHAIN=local
go test ./... -count=1
```

## Docker (benzhi)

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh mudlog linux/amd64
./build_benzhi_docker.sh mudlog linux/arm64
```