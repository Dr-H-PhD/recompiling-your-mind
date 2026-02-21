# Chapter 23: Building and Deploying

PHP deployment means copying files and configuring PHP-FPM. Go deployment means shipping a single binary. This simplicity transforms how you think about deployment.

## Single Binary Deployment (vs PHP's File Deployment)

PHP deployment:
```
/var/www/myapp/
├── composer.json
├── composer.lock
├── vendor/             # Dependencies
├── public/
│   └── index.php
├── src/
├── config/
├── var/                # Cache, logs
└── .env
```

Requires:
- PHP runtime
- Required extensions
- Composer dependencies
- Web server (nginx + PHP-FPM)
- Writeable directories

Go deployment:
```
/opt/myapp/
└── myapp               # Single binary
```

Requires:
- Nothing

### Building the Binary

```bash
# Simple build
go build -o myapp .

# Optimised build
go build -ldflags="-s -w" -o myapp .
# -s: Strip symbol table
# -w: Strip debug info
# Reduces binary size ~30%
```

### Embedding Version Info

```go
// main.go
var (
    version = "dev"
    commit  = "unknown"
    date    = "unknown"
)

func main() {
    if len(os.Args) > 1 && os.Args[1] == "version" {
        fmt.Printf("version: %s\ncommit: %s\nbuilt: %s\n", version, commit, date)
        return
    }
    // ...
}
```

```bash
go build -ldflags="-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%d)" -o myapp .
```

## Cross-Compilation

PHP can't cross-compile. Go can:

```bash
# Linux from macOS
GOOS=linux GOARCH=amd64 go build -o myapp-linux .

# Windows from macOS
GOOS=windows GOARCH=amd64 go build -o myapp.exe .

# ARM (Raspberry Pi)
GOOS=linux GOARCH=arm GOARM=7 go build -o myapp-arm .

# Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o myapp-darwin-arm64 .
```

### Supported Platforms

```bash
go tool dist list
# Shows all GOOS/GOARCH combinations
```

Common targets:
- `linux/amd64` — Linux servers
- `linux/arm64` — AWS Graviton, Apple Silicon Linux
- `darwin/amd64` — Intel Mac
- `darwin/arm64` — Apple Silicon Mac
- `windows/amd64` — Windows

### Build Matrix

```makefile
# Makefile
BINARY=myapp
VERSION=$(shell git describe --tags --always)

.PHONY: build-all
build-all:
    GOOS=linux GOARCH=amd64 go build -o dist/$(BINARY)-linux-amd64 .
    GOOS=linux GOARCH=arm64 go build -o dist/$(BINARY)-linux-arm64 .
    GOOS=darwin GOARCH=amd64 go build -o dist/$(BINARY)-darwin-amd64 .
    GOOS=darwin GOARCH=arm64 go build -o dist/$(BINARY)-darwin-arm64 .
    GOOS=windows GOARCH=amd64 go build -o dist/$(BINARY)-windows-amd64.exe .
```

## Docker Images: Multi-Stage Builds

PHP Dockerfile:
```dockerfile
FROM php:8.2-fpm
RUN apt-get update && apt-get install -y libpq-dev
RUN docker-php-ext-install pdo pdo_pgsql
COPY composer.json composer.lock ./
RUN composer install --no-dev
COPY . .
# Image size: 500MB+
```

Go multi-stage Dockerfile:
```dockerfile
# Build stage
FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o myapp .

# Runtime stage
FROM scratch
COPY --from=builder /app/myapp /myapp
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
ENTRYPOINT ["/myapp"]
# Image size: ~10MB
```

### Using `scratch` vs `alpine`

**scratch** (empty image):
```dockerfile
FROM scratch
COPY --from=builder /app/myapp /myapp
# Size: Just your binary (~10-20MB)
# No shell, no debugging tools
```

**alpine** (minimal Linux):
```dockerfile
FROM alpine:3.18
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/myapp /myapp
# Size: ~15-25MB
# Has shell for debugging
```

**distroless** (Google's minimal images):
```dockerfile
FROM gcr.io/distroless/static-debian11
COPY --from=builder /app/myapp /myapp
# Size: ~20MB
# Minimal but debuggable
```

## No Runtime Dependencies

PHP requires:
- PHP runtime
- Extensions (pdo, json, mbstring, etc.)
- Composer autoloader
- Configuration files

Go binary is self-contained:
- Statically linked (with CGO_ENABLED=0)
- All dependencies compiled in
- No runtime needed

### Verifying Static Build

```bash
# Check if truly static
file myapp
# myapp: ELF 64-bit LSB executable, x86-64, ... statically linked

ldd myapp
# not a dynamic executable
```

### Embedding Files

Go 1.16+ can embed files in the binary:

```go
import "embed"

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*.html
var templates embed.FS

func main() {
    data, _ := staticFiles.ReadFile("static/index.html")
    // ...
}
```

No need to deploy separate static files—they're in the binary.

## Systemd Services vs PHP-FPM

PHP-FPM + nginx:
```ini
# /etc/php/8.2/fpm/pool.d/www.conf
[www]
user = www-data
pm = dynamic
pm.max_children = 50
pm.start_servers = 5
```

```nginx
# /etc/nginx/sites-available/myapp
server {
    listen 80;
    root /var/www/myapp/public;
    location ~ \.php$ {
        fastcgi_pass unix:/var/run/php/php8.2-fpm.sock;
    }
}
```

Go systemd service:
```ini
# /etc/systemd/system/myapp.service
[Unit]
Description=My Go Application
After=network.target

[Service]
Type=simple
User=myapp
ExecStart=/opt/myapp/myapp
Restart=always
RestartSec=5
Environment=PORT=8080
Environment=DATABASE_URL=postgres://...

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable myapp
sudo systemctl start myapp
sudo journalctl -u myapp -f  # View logs
```

### Advantages

- No nginx needed (Go handles HTTP directly)
- No PHP-FPM process management
- Automatic restart on crash
- Simple log management via journald
- Single process to monitor

## Summary

- **Single binary** simplifies deployment dramatically
- **Cross-compilation** builds for any platform from any platform
- **Multi-stage Docker** creates tiny production images
- **No runtime** means no dependency management in production
- **Systemd** manages Go services simply and reliably

---

## Exercises

1. **Build Optimisation**: Build the same application with and without `-ldflags="-s -w"`. Compare sizes.

2. **Version Embedding**: Add version, commit, and build date to a binary using ldflags.

3. **Cross-Compile**: Build a binary for 3 different OS/arch combinations from your machine.

4. **Docker Multi-Stage**: Write a multi-stage Dockerfile. Compare image sizes with single-stage.

5. **Scratch Image**: Create a Docker image from scratch. Verify it runs and what's missing.

6. **File Embedding**: Embed static files and templates. Deploy as a single binary.

7. **Systemd Service**: Write a systemd unit file for a Go application. Test restart behaviour.

8. **CI/CD Pipeline**: Create a GitHub Actions workflow that builds for multiple platforms and creates releases.
