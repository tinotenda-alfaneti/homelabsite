.PHONY: help build run test clean docker-build docker-run

help:
	@echo "Available targets:"
	@echo "  build        - Build the Go binary"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container locally"

build:
	go build -o bin/homelabsite .

run:
	go run .

test:
	go test -v ./...

clean:
	rm -rf bin/

docker-build:
	docker build -t homelabsite:latest .

docker-run:
	docker run -p 8080:8080 -v $(PWD)/config:/srv/config homelabsite:latest
