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

## Kubernetes Deployment

PHP applications rarely run on Kubernetes due to complexity. Go's single-binary model makes Kubernetes deployment straightforward.

### Basic Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  labels:
    app: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myregistry/myapp:1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: myapp-secrets
              key: database-url
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

### Service and Ingress

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: myapp
spec:
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
---
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - api.example.com
    secretName: myapp-tls
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: myapp
            port:
              number: 80
```

### ConfigMaps and Secrets

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: myapp-config
data:
  LOG_LEVEL: "info"
  CACHE_TTL: "300"
---
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: myapp-secrets
type: Opaque
stringData:
  database-url: "postgres://user:pass@host/db"
  jwt-secret: "your-secret-key"
```

Use in deployment:

```yaml
envFrom:
- configMapRef:
    name: myapp-config
- secretRef:
    name: myapp-secrets
```

### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Helm Charts

Helm packages Kubernetes manifests for reusable deployment.

### Chart Structure

```
myapp/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   └── _helpers.tpl
└── charts/           # Dependencies
```

### Chart.yaml

```yaml
apiVersion: v2
name: myapp
description: My Go Application
version: 1.0.0
appVersion: "1.0.0"
dependencies:
- name: postgresql
  version: "12.x.x"
  repository: https://charts.bitnami.com/bitnami
  condition: postgresql.enabled
```

### values.yaml

```yaml
replicaCount: 3

image:
  repository: myregistry/myapp
  tag: "1.0.0"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 80

ingress:
  enabled: true
  host: api.example.com
  tls: true

resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "128Mi"
    cpu: "500m"

config:
  logLevel: info
  cacheTTL: 300

secrets:
  databaseUrl: ""
  jwtSecret: ""

postgresql:
  enabled: true
  auth:
    database: myapp
```

### Templated Deployment

```yaml
# templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "myapp.fullname" . }}
  labels:
    {{- include "myapp.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "myapp.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "myapp.selectorLabels" . | nindent 8 }}
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        imagePullPolicy: {{ .Values.image.pullPolicy }}
        ports:
        - containerPort: 8080
        env:
        - name: LOG_LEVEL
          value: {{ .Values.config.logLevel | quote }}
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: {{ include "myapp.fullname" . }}-secrets
              key: database-url
        resources:
          {{- toYaml .Values.resources | nindent 10 }}
```

### Helm Commands

```bash
# Install
helm install myapp ./myapp -f production-values.yaml

# Upgrade
helm upgrade myapp ./myapp -f production-values.yaml

# Rollback
helm rollback myapp 1

# Uninstall
helm uninstall myapp

# Template locally (debug)
helm template myapp ./myapp -f values.yaml
```

## Service Mesh with Istio

Service mesh provides traffic management, security, and observability for microservices.

### Istio Installation

```bash
# Install Istio
istioctl install --set profile=demo

# Enable sidecar injection for namespace
kubectl label namespace default istio-injection=enabled
```

### Virtual Service for Traffic Management

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: myapp
spec:
  hosts:
  - myapp
  http:
  - match:
    - headers:
        x-version:
          exact: "v2"
    route:
    - destination:
        host: myapp
        subset: v2
  - route:
    - destination:
        host: myapp
        subset: v1
      weight: 90
    - destination:
        host: myapp
        subset: v2
      weight: 10
```

### Destination Rules

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: myapp
spec:
  host: myapp
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        h2UpgradePolicy: UPGRADE
        http1MaxPendingRequests: 100
        http2MaxRequests: 1000
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 30s
      baseEjectionTime: 30s
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
```

### Mutual TLS

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: default
spec:
  mtls:
    mode: STRICT
```

### Circuit Breaker via Istio

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: myapp-circuit-breaker
spec:
  host: myapp
  trafficPolicy:
    outlierDetection:
      consecutive5xxErrors: 3
      interval: 10s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
```

### Observability

Istio automatically provides:
- **Tracing**: Jaeger/Zipkin integration
- **Metrics**: Prometheus metrics for all traffic
- **Visualisation**: Kiali for service graph

```bash
# Access Kiali dashboard
istioctl dashboard kiali

# Access Grafana
istioctl dashboard grafana

# Access Jaeger
istioctl dashboard jaeger
```

## GitOps with Argo CD

Declarative continuous deployment using Git as the source of truth.

### Application Definition

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: myapp
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/myorg/myapp-config
    targetRevision: HEAD
    path: overlays/production
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

### Kustomize Overlays

```
myapp-config/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   └── service.yaml
└── overlays/
    ├── staging/
    │   ├── kustomization.yaml
    │   └── patches/
    └── production/
        ├── kustomization.yaml
        └── patches/
```

```yaml
# overlays/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: production

resources:
- ../../base

replicas:
- name: myapp
  count: 5

images:
- name: myregistry/myapp
  newTag: 1.2.0

patches:
- path: patches/resources.yaml
```

## Summary

- **Single binary** simplifies deployment dramatically
- **Cross-compilation** builds for any platform from any platform
- **Multi-stage Docker** creates tiny production images
- **No runtime** means no dependency management in production
- **Systemd** manages Go services simply and reliably
- **Kubernetes** orchestrates containerised Go services at scale
- **Helm** packages and versions Kubernetes deployments
- **Service mesh** provides traffic management, security, and observability
- **GitOps** enables declarative, version-controlled deployments

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

9. **Kubernetes Deployment**: Deploy a Go application to Kubernetes with ConfigMaps, Secrets, and health checks.

10. **Helm Chart**: Create a Helm chart for your application with configurable replicas, resources, and ingress.

11. **Service Mesh**: Enable Istio sidecar injection and configure traffic splitting between two versions.

12. **GitOps Setup**: Configure Argo CD to automatically deploy from a Git repository with Kustomize overlays.
