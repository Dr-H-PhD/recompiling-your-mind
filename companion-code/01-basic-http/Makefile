.PHONY: run test coverage build clean

# Run the server
run:
	go run main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Build binary
build:
	go build -o bin/server main.go

# Clean build artefacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Docker build
docker-build:
	docker build -t basic-http .

# Docker run
docker-run:
	docker run -p 8080:8080 basic-http
