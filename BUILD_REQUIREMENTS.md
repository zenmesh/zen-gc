# Build Requirements

## Dependencies

zen-gc has no external private dependencies. All SDK code is inlined into `internal/` packages.

## Build

```bash
# Build the controller
go build -o bin/gc-controller ./cmd/gc-controller

# Run tests
go test ./...

# Build Docker image
docker build -t zenmesh/gc-controller:latest .
```

## Local Development

No special requirements - standard Go tooling works out of the box.