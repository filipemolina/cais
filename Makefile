.PHONY: dev build

dev:
	go run main.go

build:
	go build -o dist/stack-stitcher
