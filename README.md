# Products API - OpenTelemetry Example

A Go REST API built with clean architecture principles and comprehensive OpenTelemetry instrumentation for testing with the LGTM stack (Loki, Grafana, Tempo, Prometheus).

## Features

- **Clean Architecture**: Separated domain, application, and infrastructure layers
- **Dependency Injection**: Manual constructor-based DI for loose coupling
- **OpenTelemetry Instrumentation**: Full observability with traces, metrics, and logs
- **OTLP Traces**: Exports traces via OTLP gRPC to your collector
- **Prometheus Metrics**: Exposes `/metrics` endpoint for Prometheus scraping
- **Structured Logs**: JSON logs with trace context correlation
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

## Environment Configuration

The API is configured via environment variables. These can be set directly or via ConfigMaps in Kubernetes deployments.

### Required Environment Variables

| Variable | Default | Description | Example |
|----------|---------|-------------|---------|
| `SERVER_HOST` | `0.0.0.0` | Server host address | `0.0.0.0` |
| `SERVER_PORT` | `8080` | Server port | `8080` |
| `OTEL_ENABLED` | `true` | Enable/disable OpenTelemetry export | `true`, `false`, `1`, `0` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint for traces | `alloy.observability.svc.cluster.local:4317` |
| `OTEL_SERVICE_NAME` | `products-api` | Service name for telemetry | `products-api`, `otlp-api` |
| `OTEL_ENVIRONMENT` | `development` | Environment name | `development`, `staging`, `production` |

### OTEL_ENABLED Behavior

- **`true` (default)**: Full telemetry with OTLP export for traces, Prometheus metrics, and trace-correlated logs
- **`false`**: No-op mode - traces and metrics are recorded but NOT exported. Useful for:
  - Local development without telemetry infrastructure
  - Testing environments where observability isn't needed
  - Debugging without trace overhead
  - Cost optimization in non-production environments

**Note**: Prometheus `/metrics` endpoint remains available even when `OTEL_ENABLED=false`

### What Gets Exported Where

- **Traces**: Sent via OTLP gRPC to `OTEL_EXPORTER_OTLP_ENDPOINT` (e.g., Alloy)
- **Metrics**: All sent via OTLP gRPC to `OTEL_EXPORTER_OTLP_ENDPOINT`
  - **HTTP metrics** (automatic): `http.server.active_requests`, `http.server.duration`, etc.
  - **Business metrics**: `products_created_total`, `products_operations_total`
  - **Prometheus endpoint**: Also available at `/metrics` for direct scraping
- **Logs**: Written to stdout as JSON (collected by log aggregators like Alloy/Promtail)

### Example Configurations

**Development (with telemetry):**
```bash
export SERVER_HOST="0.0.0.0"
export SERVER_PORT="8080"
export OTEL_ENABLED="true"
export OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
export OTEL_SERVICE_NAME="products-api"
export OTEL_ENVIRONMENT="development"
```

**Development (without telemetry):**
```bash
export SERVER_HOST="0.0.0.0"
export SERVER_PORT="8080"
export OTEL_ENABLED="false"
# OTLP settings ignored when disabled
```

**Kubernetes Production (via ConfigMap):**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otlp-api-config
data:
  SERVER_HOST: "0.0.0.0"
  SERVER_PORT: "8080"
  OTEL_ENABLED: "true"
  OTEL_EXPORTER_OTLP_ENDPOINT: "alloy.observability.svc.cluster.local:4317"
  OTEL_SERVICE_NAME: "otlp-api"
  OTEL_ENVIRONMENT: "production"
```

**Kubernetes Development (telemetry disabled):**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otlp-api-config
data:
  SERVER_HOST: "0.0.0.0"
  SERVER_PORT: "8080"
  OTEL_ENABLED: "false"
  OTEL_SERVICE_NAME: "otlp-api"
  OTEL_ENVIRONMENT: "development"
```

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

### 5. Metrics (Prometheus)

```bash
GET /metrics
```

**Response (200 OK):**
```
# Prometheus metrics in exposition format
http_requests_total{method="GET",route="/products",status_code="200",service="otlp-api"} 42
...
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

The API creates distributed traces for all operations via OTLP:

- **HTTP Layer**: Automatic tracing of all incoming requests
- **Service Layer**: Manual spans for business logic operations
- **Repository Layer**: Spans for data storage operations
- **Export**: Sent to `OTEL_EXPORTER_OTLP_ENDPOINT` via gRPC

**Example trace hierarchy:**
```
HTTP POST /products
├── ProductService.CreateProduct
│   └── ProductRepository.Create
```

### Metrics

The API uses **OpenTelemetry automatic HTTP instrumentation** via `otelhttp.NewHandler`:

#### HTTP Metrics (Automatic via otelhttp)

Automatically exported via OTLP to `OTEL_EXPORTER_OTLP_ENDPOINT`:

- `http.server.active_requests` - Number of active/in-flight HTTP requests (gauge)
- `http.server.duration` - HTTP request duration with histogram buckets
- `http.server.request.size` - Size of HTTP requests
- `http.server.response.size` - Size of HTTP responses

**With trace correlation:**
- All metrics include **exemplars** linking to traces via `trace_id` and `span_id`
- Follows OpenTelemetry semantic conventions
- Automatic span creation for each HTTP request

#### Business Metrics (OTLP)

Application-specific metrics sent via OTLP:

- `products_created_total` - Total products created
- `products_operations_total` - Product operations by type and result

#### Prometheus /metrics Endpoint

Still available at `http://<host>:<port>/metrics` for compatibility:
- Exposes OpenTelemetry metrics in Prometheus format
- Useful for direct Prometheus scraping if needed
- No manual metric tracking required

### Logs

Structured JSON logs with **automatic trace correlation**:

```json
{
  "time": "2025-12-21T10:00:00Z",
  "level": "INFO",
  "msg": "Creating product",
  "service.name": "otlp-api",
  "environment": "production",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "name": "Laptop",
  "price": 1299.99
}
```

**Key fields for correlation:**
- `trace_id`: Links log entry to distributed trace
- `span_id`: Links to specific span in trace
- Grafana automatically correlates logs ↔ traces using these fields

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
