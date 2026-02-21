# Recompiling Your Mind — Companion Code

This repository contains working code examples for the book "Recompiling Your Mind: A PHP Developer's Journey to Go".

## Projects

| Project | Chapter | Description |
|---------|---------|-------------|
| [01-basic-http](./01-basic-http/) | Chapter 10 | Basic HTTP server with middleware patterns |
| [02-database](./02-database/) | Chapter 11 | Database access patterns with `database/sql` |
| [03-rest-api](./03-rest-api/) | Chapter 12 | Complete REST API with authentication |
| [04-channels](./04-channels/) | Chapter 16 | Channel communication patterns |
| [05-worker-pool](./05-worker-pool/) | Chapter 18 | Worker pool and concurrency patterns |
| [06-migration-example](./06-migration-example/) | Chapter 25 | PHP to Go migration example |

## Requirements

- Go 1.21+
- Docker & Docker Compose (for database examples)
- Make

## Quick Start

Each project can be run independently:

```bash
cd 01-basic-http
make run
```

## Project Structure

Each project follows a consistent structure:

```
project/
├── main.go           # Entry point
├── Makefile          # Common operations
├── Dockerfile        # Container build
├── docker-compose.yml # Local development
├── go.mod            # Dependencies
└── *_test.go         # Tests
```

## Running Tests

```bash
# Run all tests in a project
cd 01-basic-http
make test

# Run with coverage
make coverage
```

## For PHP Developers

Each project includes comments comparing Go patterns to their PHP/Symfony equivalents. Look for comments like:

```go
// PHP equivalent: $container->get('service.name')
// Symfony: services.yaml dependency injection
```
