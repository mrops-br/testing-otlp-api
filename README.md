# Products API - OpenTelemetry Example

A Go REST API built with clean architecture principles and comprehensive OpenTelemetry instrumentation for testing with the LGTM stack (Loki, Grafana, Tempo, Mimir/Prometheus).

## Features

- **Clean Architecture**: Separated domain, application, and infrastructure layers
- **Dependency Injection**: Manual constructor-based DI for loose coupling
- **OpenTelemetry Instrumentation**: Full observability with traces, metrics, and logs
- **OTLP Export**: Exports telemetry data via OTLP gRPC to your collector
- **RESTful API**: Three example endpoints for product management

## Architecture

```
├── internal/
│   ├── domain/              # Business entities and interfaces
│   │   ├── product.go
│   │   └── repository.go
│   ├── app/                 # Application services and DTOs
│   │   ├── dto/
│   │   └── service/
│   └── infrastructure/      # External concerns
│       ├── config/          # Configuration management
│       ├── telemetry/       # OpenTelemetry setup
│       ├── repository/      # Data storage implementations
│       └── http/            # HTTP handlers and server
└── main.go                  # Application entry point
```

## Prerequisites

### Local Development
- Go 1.21 or higher
- LGTM stack (or OpenTelemetry Collector) running and accepting OTLP data

### Docker Deployment
- Docker and Docker Compose

### Kubernetes Deployment
- Kubernetes cluster (1.21+)
- kubectl configured
- OpenTelemetry Collector deployed in cluster

## Configuration

The API is configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `0.0.0.0` | Server host address |
| `SERVER_PORT` | `8080` | Server port |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint |
| `OTEL_SERVICE_NAME` | `products-api` | Service name for telemetry |
| `OTEL_ENVIRONMENT` | `development` | Environment name |

## Running the API

### 1. Start your LGTM stack

Ensure your LGTM stack is running with an OTLP receiver on port 4317 (gRPC).

Example Docker Compose snippet for Tempo:
```yaml
services:
  tempo:
    image: grafana/tempo:latest
    ports:
      - "4317:4317"  # OTLP gRPC
      - "4318:4318"  # OTLP HTTP
```

### 2. Run the API

```bash
# With default configuration
go run main.go

# With custom configuration
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_SERVICE_NAME=my-products-api \
SERVER_PORT=8080 \
go run main.go
```

The API will start on `http://localhost:8080`

## Running with Docker

### Build Docker Image

```bash
# Build the Docker image
docker build -t products-api:latest .
```

The multi-stage Dockerfile:
- Uses `golang:1.21-alpine` for building
- Creates a minimal `alpine:latest` runtime image (~20MB)
- Runs as non-root user for security
- Includes health check endpoint

### Run with Docker

```bash
# Run the API container
docker run -p 8080:8080 \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=host.docker.internal:4317 \
  -e OTEL_SERVICE_NAME=products-api \
  -e OTEL_ENVIRONMENT=production \
  products-api:latest
```

### Run Complete Stack with Docker Compose

The project includes a complete `docker-compose.yml` with the full LGTM stack:

```bash
# Start everything (API + LGTM stack)
docker-compose up -d

# View logs
docker-compose logs -f products-api

# Stop everything
docker-compose down
```

This starts:
- **products-api** on port 8080
- **Tempo** (traces) on port 3200
- **Loki** (logs) on port 3100
- **Prometheus** (metrics) on port 9090
- **Grafana** (dashboards) on port 3000
- **OpenTelemetry Collector** (optional) on ports 4317/4318

Access Grafana at `http://localhost:3000` (no login required in dev mode).

## Running on Kubernetes

The application is **fully ConfigMap-ready** for Kubernetes deployments.

### Quick Deploy

```bash
# 1. Update ConfigMap with your OTLP collector endpoint
# Edit k8s/configmap.yaml and change OTEL_EXPORTER_OTLP_ENDPOINT

# 2. Update image in k8s/deployment.yaml

# 3. Deploy
kubectl create namespace production
kubectl apply -f k8s/configmap.yaml -n production
kubectl apply -f k8s/deployment.yaml -n production
kubectl apply -f k8s/service.yaml -n production

# 4. Test
kubectl port-forward -n production svc/products-api 8080:80
curl http://localhost:8080/health
```

### What's Included

- **ConfigMap**: Externalized configuration (OTLP endpoint, environment, etc.)
- **Deployment**: Production-ready with health checks, resource limits, security contexts
- **Service**: ClusterIP and headless services
- **HPA**: Horizontal Pod Autoscaler (2-10 replicas based on CPU/memory)
- **Ingress**: External access configuration
- **ServiceMonitor**: Prometheus metrics scraping

### ConfigMap-Based Configuration

All configuration is managed via ConfigMaps:

```yaml
# k8s/configmap.yaml
data:
  OTEL_EXPORTER_OTLP_ENDPOINT: "otel-collector.observability.svc.cluster.local:4317"
  OTEL_SERVICE_NAME: "products-api"
  OTEL_ENVIRONMENT: "production"
```

Update configuration without rebuilding:

```bash
# Edit ConfigMap
kubectl edit configmap products-api-config -n production

# Restart to apply changes
kubectl rollout restart deployment/products-api -n production
```

### Environment-Specific Deployments

```bash
# Development
kubectl apply -f k8s/configmap-dev.yaml -n development

# Staging
kubectl apply -f k8s/configmap-staging.yaml -n staging

# Production
kubectl apply -f k8s/configmap-prod.yaml -n production
```

**For complete Kubernetes documentation, see [KUBERNETES.md](KUBERNETES.md)**

## API Endpoints

### 1. Create Product

```bash
POST /products
Content-Type: application/json

{
  "name": "Laptop",
  "description": "High-performance laptop",
  "price": 1299.99
}
```

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Laptop",
  "description": "High-performance laptop",
  "price": 1299.99,
  "created_at": "2025-12-20T10:00:00Z",
  "updated_at": "2025-12-20T10:00:00Z"
}
```

### 2. Get Product by ID

```bash
GET /products/{id}
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Laptop",
  "description": "High-performance laptop",
  "price": 1299.99,
  "created_at": "2025-12-20T10:00:00Z",
  "updated_at": "2025-12-20T10:00:00Z"
}
```

### 3. List All Products

```bash
GET /products
```

**Response (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Laptop",
    "description": "High-performance laptop",
    "price": 1299.99,
    "created_at": "2025-12-20T10:00:00Z",
    "updated_at": "2025-12-20T10:00:00Z"
  }
]
```

### 4. Health Check

```bash
GET /health
```

**Response (200 OK):**
```
OK
```

## Example Usage

```bash
# Create a product
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Wireless Mouse",
    "description": "Ergonomic wireless mouse",
    "price": 29.99
  }'

# Get product by ID (replace with actual ID from create response)
curl http://localhost:8080/products/550e8400-e29b-41d4-a716-446655440000

# List all products
curl http://localhost:8080/products
```

## OpenTelemetry Instrumentation

### Traces

The API creates distributed traces for all operations:

- **HTTP Layer**: Automatic tracing of all incoming requests
- **Service Layer**: Manual spans for business logic operations
- **Repository Layer**: Spans for data storage operations

**Example trace hierarchy:**
```
HTTP POST /products
├── ProductService.CreateProduct
│   └── ProductRepository.Create
```

### Metrics

The API exports the following metrics:

**HTTP Metrics:**
- `http.server.request.count` - Total number of HTTP requests
- `http.server.request.duration` - HTTP request duration (histogram)

**Business Metrics:**
- `products.created.total` - Total products created
- `products.operations` - Product operations by type

### Logs

Structured JSON logs with trace correlation:

```json
{
  "time": "2025-12-20T10:00:00Z",
  "level": "INFO",
  "msg": "Creating product",
  "service.name": "products-api",
  "environment": "development",
  "name": "Laptop",
  "price": 1299.99
}
```

## Viewing Telemetry Data

### Traces (Tempo)

Access Grafana and query traces:
- Service: `products-api`
- Operation: `HTTP POST /products`, `ProductService.CreateProduct`, etc.

### Metrics (Prometheus)

Query metrics in Prometheus or Grafana:
```promql
rate(http_server_request_count[5m])
histogram_quantile(0.95, http_server_request_duration)
products_created_total
```

### Logs (Loki)

Query logs in Grafana:
```logql
{service_name="products-api"} | json
```

## Development

### Build

```bash
go build -o products-api main.go
```

### Run tests

```bash
go test ./...
```

### Project Structure

The project follows clean architecture principles:

- **Domain Layer** (`internal/domain/`): Core business logic, entities, and interfaces. No external dependencies.
- **Application Layer** (`internal/app/`): Use cases and DTOs. Orchestrates domain entities.
- **Infrastructure Layer** (`internal/infrastructure/`): External concerns like HTTP, database, telemetry.

## License

MIT
