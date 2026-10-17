setup:
    lefthook install -f

mod:
    go mod download
    go mod tidy    
    
lint-fix:
  golangci-lint run --fix ./...

format:
  nix fmt

test-all:
  go test -race -count=1 -parallel=4 ./...
  go tool otelc go test -tags=otelc -race -count=1 ./dbsql
