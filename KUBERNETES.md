# Kubernetes Deployment Guide

This guide explains how to deploy the Products API to Kubernetes with ConfigMap-based configuration.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [ConfigMap Configuration](#configmap-configuration)
- [Deployment](#deployment)
- [Accessing the API](#accessing-the-api)
- [Scaling](#scaling)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Overview

The application is **fully ConfigMap-ready** and uses environment variables for all configuration. The Kubernetes manifests include:

- **ConfigMap**: Externalized configuration (OTLP endpoint, service name, etc.)
- **Deployment**: Application deployment with health checks and resource limits
- **Service**: ClusterIP and headless services
- **HPA**: Horizontal Pod Autoscaler for automatic scaling
- **Ingress**: External access configuration
- **ServiceMonitor**: Prometheus metrics scraping (optional)

## Prerequisites

- Kubernetes cluster (1.21+)
- kubectl configured
- Docker image built and pushed to registry
- OpenTelemetry Collector deployed in your cluster

## ConfigMap Configuration

### Understanding the ConfigMap

The ConfigMap stores all application configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: products-api-config
data:
  SERVER_HOST: "0.0.0.0"
  SERVER_PORT: "8080"
  OTEL_EXPORTER_OTLP_ENDPOINT: "otel-collector.observability.svc.cluster.local:4317"
  OTEL_SERVICE_NAME: "products-api"
  OTEL_ENVIRONMENT: "production"
```

### Key Configuration Values

| Variable | Description | Example |
|----------|-------------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector endpoint | `otel-collector.observability.svc.cluster.local:4317` |
| `OTEL_SERVICE_NAME` | Service name in traces | `products-api` |
| `OTEL_ENVIRONMENT` | Environment tag | `production`, `staging`, `development` |
| `SERVER_PORT` | API port | `8080` |

### Environment-Specific ConfigMaps

Use different ConfigMaps per environment:

```bash
# Development
kubectl apply -f k8s/configmap-dev.yaml -n development

# Staging
kubectl apply -f k8s/configmap-staging.yaml -n staging

# Production
kubectl apply -f k8s/configmap-prod.yaml -n production
```

### Updating Configuration

**Important**: ConfigMap changes don't automatically restart pods. You must:

```bash
# Update the ConfigMap
kubectl apply -f k8s/configmap.yaml

# Restart the deployment to pick up changes
kubectl rollout restart deployment/products-api

# Watch the rollout
kubectl rollout status deployment/products-api
```

**Alternative**: Use a tool like [Reloader](https://github.com/stakater/Reloader) to auto-restart on ConfigMap changes.

## Deployment

### 1. Build and Push Docker Image

```bash
# Build the image
docker build -t your-registry/products-api:v1.0.0 .

# Tag as latest
docker tag your-registry/products-api:v1.0.0 your-registry/products-api:latest

# Push to registry
docker push your-registry/products-api:v1.0.0
docker push your-registry/products-api:latest
```

### 2. Update Image in Deployment

Edit `k8s/deployment.yaml`:

```yaml
spec:
  template:
    spec:
      containers:
      - name: products-api
        image: your-registry/products-api:v1.0.0  # Update this
```

### 3. Deploy to Kubernetes

#### Option A: Direct Apply

```bash
# Create namespace (if needed)
kubectl create namespace production

# Apply all manifests
kubectl apply -f k8s/configmap-prod.yaml -n production
kubectl apply -f k8s/deployment.yaml -n production
kubectl apply -f k8s/service.yaml -n production
kubectl apply -f k8s/hpa.yaml -n production
kubectl apply -f k8s/ingress.yaml -n production

# Verify deployment
kubectl get pods -n production -l app=products-api
```

#### Option B: Using Kustomize

```bash
# Deploy using kustomize
kubectl apply -k k8s/

# Or with custom namespace
kubectl apply -k k8s/ -n production
```

#### Option C: Using Helm (if you create a chart)

```bash
helm install products-api ./helm/products-api \
  --namespace production \
  --create-namespace \
  --set image.tag=v1.0.0 \
  --set config.otlpEndpoint="otel-collector.observability.svc.cluster.local:4317"
```

### 4. Verify Deployment

```bash
# Check pods
kubectl get pods -n production -l app=products-api

# Check deployment status
kubectl rollout status deployment/products-api -n production

# View logs
kubectl logs -n production -l app=products-api --tail=100 -f

# Check ConfigMap
kubectl get configmap products-api-config -n production -o yaml
```

## Accessing the API

### Inside the Cluster

```bash
# Port-forward for testing
kubectl port-forward -n production svc/products-api 8080:80

# Test the API
curl http://localhost:8080/health
curl http://localhost:8080/products
```

### Via Ingress (External)

After deploying the Ingress:

```bash
# Get Ingress address
kubectl get ingress -n production

# Access via domain
curl https://products-api.yourdomain.com/health
```

### Via LoadBalancer (Alternative)

Modify `k8s/service.yaml`:

```yaml
spec:
  type: LoadBalancer  # Change from ClusterIP
```

Then:

```bash
# Get external IP
kubectl get svc products-api -n production

# Access via external IP
curl http://<EXTERNAL-IP>/products
```

## Scaling

### Manual Scaling

```bash
# Scale to 5 replicas
kubectl scale deployment products-api --replicas=5 -n production

# Verify
kubectl get pods -n production -l app=products-api
```

### Horizontal Pod Autoscaler (HPA)

The HPA is configured to scale based on CPU and memory:

```bash
# Check HPA status
kubectl get hpa products-api-hpa -n production

# View detailed metrics
kubectl describe hpa products-api-hpa -n production
```

HPA Configuration:
- **Min Replicas**: 2
- **Max Replicas**: 10
- **CPU Target**: 70%
- **Memory Target**: 80%

### Vertical Pod Autoscaler (Optional)

Install VPA:

```bash
kubectl apply -f https://github.com/kubernetes/autoscaler/releases/download/vertical-pod-autoscaler-0.13.0/vertical-pod-autoscaler.yaml
```

Create VPA manifest:

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: products-api-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: products-api
  updatePolicy:
    updateMode: "Auto"
```

## Monitoring

### Health Checks

The deployment includes three types of probes:

**Liveness Probe**: Restarts unhealthy pods
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
```

**Readiness Probe**: Removes unready pods from service
```yaml
readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

**Startup Probe**: Handles slow-starting containers
```yaml
startupProbe:
  httpGet:
    path: /health
    port: 8080
  failureThreshold: 12
  periodSeconds: 5
```

### Viewing Logs

```bash
# Tail logs from all pods
kubectl logs -n production -l app=products-api --tail=100 -f

# Logs from specific pod
kubectl logs -n production <pod-name>

# Previous container logs (after crash)
kubectl logs -n production <pod-name> --previous
```

### OpenTelemetry Data

All telemetry data is exported to your OTLP collector:

**Traces**: View in Tempo/Jaeger
**Metrics**: View in Prometheus/Grafana
**Logs**: View in Loki

### Prometheus Metrics (with ServiceMonitor)

If using Prometheus Operator:

```bash
# Apply ServiceMonitor
kubectl apply -f k8s/servicemonitor.yaml -n production

# Verify in Prometheus
# Navigate to Prometheus UI and query: up{app="products-api"}
```

## ConfigMap Management Best Practices

### 1. Version Your ConfigMaps

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: products-api-config-v2  # Versioned name
  labels:
    version: "2"
```

### 2. Use ConfigMap Generators (Kustomize)

```yaml
# kustomization.yaml
configMapGenerator:
  - name: products-api-config
    literals:
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector.observability.svc.cluster.local:4317
```

This auto-generates a hash suffix, triggering automatic pod restarts on changes.

### 3. Separate Secrets from ConfigMaps

For sensitive data, use Secrets:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: products-api-secrets
type: Opaque
stringData:
  otlp-auth-token: "your-secret-token"
```

Reference in deployment:

```yaml
env:
- name: OTEL_EXPORTER_OTLP_HEADERS
  valueFrom:
    secretKeyRef:
      name: products-api-secrets
      key: otlp-auth-token
```

## Updating OTLP Collector Endpoint

### Scenario: Change collector endpoint

```bash
# Edit ConfigMap
kubectl edit configmap products-api-config -n production

# Change:
# OTEL_EXPORTER_OTLP_ENDPOINT: "new-collector.observability.svc.cluster.local:4317"

# Restart deployment
kubectl rollout restart deployment/products-api -n production

# Verify
kubectl logs -n production -l app=products-api | grep "Initializing OpenTelemetry"
```

## Troubleshooting

### Pods Not Starting

```bash
# Describe pod to see events
kubectl describe pod <pod-name> -n production

# Common issues:
# - Image pull errors: Check image name and registry credentials
# - ConfigMap not found: Ensure ConfigMap exists in same namespace
# - Resource limits: Check if node has enough resources
```

### ConfigMap Not Loading

```bash
# Verify ConfigMap exists
kubectl get configmap products-api-config -n production

# Check if pod references correct ConfigMap
kubectl get deployment products-api -n production -o yaml | grep configMapRef

# Verify environment variables in pod
kubectl exec -n production <pod-name> -- env | grep OTEL
```

### OTLP Connection Issues

```bash
# Test connectivity from pod
kubectl exec -n production <pod-name> -- wget -qO- http://otel-collector.observability.svc.cluster.local:4318

# Check if collector is running
kubectl get pods -n observability -l app=otel-collector

# View application logs for connection errors
kubectl logs -n production <pod-name> | grep -i "otlp\|telemetry\|error"
```

### High Memory/CPU Usage

```bash
# Check resource usage
kubectl top pods -n production -l app=products-api

# Check if hitting limits
kubectl describe pod <pod-name> -n production | grep -A 5 "Limits"

# Increase limits in deployment.yaml
resources:
  limits:
    memory: "512Mi"  # Increase
    cpu: "1000m"
```

## Multi-Environment Deployment

### GitOps Approach with Kustomize Overlays

```
k8s/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   └── service.yaml
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   └── configmap.yaml
    ├── staging/
    │   ├── kustomization.yaml
    │   └── configmap.yaml
    └── production/
        ├── kustomization.yaml
        └── configmap.yaml
```

Deploy per environment:

```bash
# Development
kubectl apply -k k8s/overlays/dev

# Staging
kubectl apply -k k8s/overlays/staging

# Production
kubectl apply -k k8s/overlays/production
```

## Blue-Green Deployment

```bash
# Deploy green version
kubectl set image deployment/products-api \
  products-api=your-registry/products-api:v2.0.0 \
  -n production

# Monitor rollout
kubectl rollout status deployment/products-api -n production

# Rollback if needed
kubectl rollout undo deployment/products-api -n production
```

## Canary Deployment (with Istio/Linkerd)

Example with traffic split:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: products-api
spec:
  hosts:
  - products-api
  http:
  - match:
    - headers:
        canary:
          exact: "true"
    route:
    - destination:
        host: products-api
        subset: v2
  - route:
    - destination:
        host: products-api
        subset: v1
      weight: 90
    - destination:
        host: products-api
        subset: v2
      weight: 10
```

## Summary

Your Products API is **fully ready for Kubernetes deployment** with ConfigMap support:

✅ **ConfigMap-ready**: All configuration via environment variables
✅ **Production-ready**: Health checks, resource limits, security contexts
✅ **Observable**: Full OpenTelemetry integration
✅ **Scalable**: HPA configuration included
✅ **Secure**: Non-root user, read-only filesystem
✅ **Multi-environment**: Environment-specific ConfigMaps provided

Simply update the OTLP collector endpoint in the ConfigMap and deploy!
