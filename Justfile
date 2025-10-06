setup:
    lefthook install -f

mod:
    go mod download
    go mod tidy

lint-fix:
  golangci-lint run --fix ./...

test-all:
  go test -race -count=1 -parallel=4 ./...
