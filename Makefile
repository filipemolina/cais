.PHONY: dev build

dev:
	go run main.go

# Build and install to $(go env GOPATH)/bin (~/go/bin by default).
# ~/go/bin is on PATH, so `stack-stitcher` is runnable immediately
# after `make build` — no sudo, no extra setup.
build:
	go install .
