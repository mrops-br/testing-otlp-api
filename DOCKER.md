# Docker Deployment Guide

This guide explains how to build and deploy the Products API using Docker.

## Multi-Stage Dockerfile

The project uses a multi-stage Dockerfile for optimal image size and security:

### Stage 1: Builder
- Base image: `golang:1.21-alpine`
- Installs build dependencies (git, ca-certificates, tzdata)
- Compiles the Go binary with optimizations:
  - `CGO_ENABLED=0` for static linking
  - `-ldflags="-w -s"` to strip debug symbols (~30% size reduction)

### Stage 2: Runtime
- Base image: `alpine:latest` (~5MB)
- Adds only runtime essentials (ca-certificates, tzdata)
- Creates non-root user (UID 1001) for security
- Final image size: ~20-25MB

## Building the Image

### Basic Build

```bash
docker build -t products-api:latest .
```

### Build with Custom Tag

```bash
docker build -t products-api:v1.0.0 .
```

### Build for Different Architecture

```bash
# For ARM64 (Apple Silicon, AWS Graviton)
docker build --platform linux/arm64 -t products-api:latest-arm64 .

# For AMD64 (most cloud providers)
docker build --platform linux/amd64 -t products-api:latest-amd64 .
```

### Multi-Architecture Build (for push to registry)

```bash
docker buildx build --platform linux/amd64,linux/arm64 \
  -t your-registry/products-api:latest \
  --push .
```

## Running the Container

### Standalone Container

```bash
docker run -d \
  --name products-api \
  -p 8080:8080 \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=your-collector:4317 \
  -e OTEL_SERVICE_NAME=products-api \
  -e OTEL_ENVIRONMENT=production \
  products-api:latest
```

### With Local OTLP Collector

If running the OTLP collector on your host machine:

```bash
docker run -d \
  --name products-api \
  -p 8080:8080 \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=host.docker.internal:4317 \
  products-api:latest
```

### View Logs

```bash
# Follow logs
docker logs -f products-api

# Last 100 lines
docker logs --tail 100 products-api
```

### Execute Commands in Container

```bash
# Get shell access
docker exec -it products-api /bin/sh

# Check health
docker exec products-api wget -qO- http://localhost:8080/health
```

## Docker Compose Deployment

### Full Stack (Recommended)

Start the complete LGTM stack with the API:

```bash
# Start all services
docker-compose up -d

# View logs for specific service
docker-compose logs -f products-api
docker-compose logs -f tempo
docker-compose logs -f grafana

# Check service status
docker-compose ps

# Stop all services
docker-compose down

# Stop and remove volumes (cleans all data)
docker-compose down -v
```

### API Only

If you have your own LGTM stack running:

```yaml
# docker-compose.api-only.yml
version: '3.8'

services:
  products-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - OTEL_EXPORTER_OTLP_ENDPOINT=your-collector-host:4317
      - OTEL_SERVICE_NAME=products-api
      - OTEL_ENVIRONMENT=production
```

Run with:
```bash
docker-compose -f docker-compose.api-only.yml up -d
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `0.0.0.0` | Server bind address |
| `SERVER_PORT` | `8080` | Server port |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint |
| `OTEL_SERVICE_NAME` | `products-api` | Service name in telemetry |
| `OTEL_ENVIRONMENT` | `development` | Environment tag |

## Health Check

The Docker image includes a built-in health check:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health
```

Check health status:
```bash
docker inspect --format='{{.State.Health.Status}}' products-api
```

## Ports

| Port | Service | Description |
|------|---------|-------------|
| 8080 | Products API | HTTP API endpoints |
| 3000 | Grafana | Dashboards and visualization |
| 3100 | Loki | Log aggregation |
| 3200 | Tempo | Trace storage |
| 4317 | OTLP Collector | OTLP gRPC receiver |
| 4318 | OTLP Collector | OTLP HTTP receiver |
| 9090 | Prometheus | Metrics storage |

## Accessing Services

After running `docker-compose up -d`:

- **API**: http://localhost:8080
  - Health: http://localhost:8080/health
  - Products: http://localhost:8080/products

- **Grafana**: http://localhost:3000
  - No login required (anonymous mode)
  - Pre-configured data sources for Tempo, Loki, Prometheus

- **Prometheus**: http://localhost:9090
  - Query metrics directly

## Production Deployment

### Security Best Practices

1. **Run as non-root**: Already configured (UID 1001)

2. **Use read-only filesystem**:
   ```bash
   docker run --read-only -d products-api:latest
   ```

3. **Limit resources**:
   ```bash
   docker run -d \
     --memory=256m \
     --cpus=0.5 \
     products-api:latest
   ```

4. **Use secrets for sensitive config**:
   ```yaml
   services:
     products-api:
       environment:
         - OTEL_EXPORTER_OTLP_ENDPOINT_FILE=/run/secrets/otlp_endpoint
       secrets:
         - otlp_endpoint

   secrets:
     otlp_endpoint:
       external: true
   ```

### Kubernetes Deployment

Example Kubernetes manifest:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: products-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: products-api
  template:
    metadata:
      labels:
        app: products-api
    spec:
      containers:
      - name: products-api
        image: products-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "otel-collector:4317"
        - name: OTEL_SERVICE_NAME
          value: "products-api"
        - name: OTEL_ENVIRONMENT
          value: "production"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: products-api
spec:
  selector:
    app: products-api
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs products-api

# Check if port is already in use
lsof -i :8080

# Inspect container
docker inspect products-api
```

### OTLP Connection Issues

```bash
# Test connectivity from container
docker exec products-api wget -O- http://tempo:4317

# Check if collector is running
docker-compose ps tempo

# Verify network connectivity
docker network inspect optl-testing-api_lgtm-network
```

### High Memory Usage

```bash
# Check container stats
docker stats products-api

# Limit memory
docker update --memory 256m products-api
```

### Logs Not Appearing in Loki

1. Check if logs are JSON formatted
2. Verify OTLP collector is forwarding to Loki
3. Check Loki configuration in `otel-collector-config.yaml`

## Building for Production Registry

### Tag and Push

```bash
# Tag for registry
docker tag products-api:latest registry.example.com/products-api:v1.0.0
docker tag products-api:latest registry.example.com/products-api:latest

# Push to registry
docker push registry.example.com/products-api:v1.0.0
docker push registry.example.com/products-api:latest
```

### Using Docker Hub

```bash
docker login
docker tag products-api:latest yourusername/products-api:latest
docker push yourusername/products-api:latest
```

### Using GitHub Container Registry

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
docker tag products-api:latest ghcr.io/username/products-api:latest
docker push ghcr.io/username/products-api:latest
```

## Clean Up

```bash
# Remove all containers and volumes
docker-compose down -v

# Remove images
docker rmi products-api:latest

# Clean up unused images and containers
docker system prune -a

# Remove specific volumes
docker volume rm optl-testing-api_tempo-data
docker volume rm optl-testing-api_loki-data
docker volume rm optl-testing-api_prometheus-data
docker volume rm optl-testing-api_grafana-data
```
