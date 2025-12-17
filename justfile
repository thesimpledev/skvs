set shell := ["bash", "-cu"]

PKGS := `go list -f '{{.Dir}}' ./cmd/... ./internal/... | grep -v /vendor/ | tr '\n' ' '`

test:
	rm -f test/*
	mkdir -p test
	go fmt ./...
	go vet ./...
	staticcheck ./...
	errcheck ./...
	revive -config ~/.revive.toml ./...
	gosec ./...
	govulncheck ./...
	go test ./... -race -vet=all -shuffle=on -count=1 -timeout=30s -coverprofile=test/coverage.out
	go tool cover -func=test/coverage.out
	go tool cover -html=test/coverage.out -o test/coverage.html

test-mem:
	rm -f test/*
	mkdir -p test
	go test {{PKGS}} -v -race -vet=all -shuffle=on -count=1 -timeout=30s -coverprofile=test/coverage.out -gcflags="-m"

test-open:
	open test/coverage.html

install-tools:
    go install honnef.co/go/tools/cmd/staticcheck@latest
    go install github.com/kisielk/errcheck@latest
    go install github.com/mgechev/revive@latest
    go install github.com/securego/gosec/v2/cmd/gosec@latest
    go install golang.org/x/vuln/cmd/govulncheck@latest
