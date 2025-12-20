# Kubernetes Quick Start

## Prerequisites Checklist

- [ ] Kubernetes cluster running
- [ ] kubectl configured
- [ ] Docker image built and pushed to registry
- [ ] OpenTelemetry Collector deployed (or know the endpoint)

## 5-Minute Deployment

### 1. Update ConfigMap with Your OTLP Collector Endpoint

Edit `k8s/configmap.yaml`:

```yaml
data:
  OTEL_EXPORTER_OTLP_ENDPOINT: "YOUR-COLLECTOR-HOST:4317"  # ← Change this
```

**Common endpoints:**
- Same namespace: `otel-collector:4317`
- Different namespace: `otel-collector.observability.svc.cluster.local:4317`
- External: `collector.example.com:4317`

### 2. Update Deployment Image

Edit `k8s/deployment.yaml`:

```yaml
containers:
- name: products-api
  image: your-registry/products-api:latest  # ← Change this
```

### 3. Deploy

```bash
# Create namespace
kubectl create namespace production

# Deploy everything
kubectl apply -f k8s/configmap.yaml -n production
kubectl apply -f k8s/deployment.yaml -n production
kubectl apply -f k8s/service.yaml -n production

# Watch deployment
kubectl get pods -n production -w
```

### 4. Test

```bash
# Port-forward
kubectl port-forward -n production svc/products-api 8080:80

# Test API
curl http://localhost:8080/health
curl http://localhost:8080/products

# Create a product
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","description":"K8s test","price":99.99}'
```

### 5. View Logs

```bash
kubectl logs -n production -l app=products-api --tail=50 -f
```

## Environment-Specific Deployment

### Development

```bash
kubectl apply -f k8s/configmap-dev.yaml -n development
kubectl apply -f k8s/deployment.yaml -n development
kubectl apply -f k8s/service.yaml -n development
```

### Staging

```bash
kubectl apply -f k8s/configmap-staging.yaml -n staging
kubectl apply -f k8s/deployment.yaml -n staging
kubectl apply -f k8s/service.yaml -n staging
```

### Production

```bash
kubectl apply -f k8s/configmap-prod.yaml -n production
kubectl apply -f k8s/deployment.yaml -n production
kubectl apply -f k8s/service.yaml -n production
kubectl apply -f k8s/hpa.yaml -n production
kubectl apply -f k8s/ingress.yaml -n production
```

## Update Configuration

When you need to change the OTLP endpoint or other settings:

```bash
# 1. Edit ConfigMap
kubectl edit configmap products-api-config -n production

# 2. Restart deployment to pick up changes
kubectl rollout restart deployment/products-api -n production

# 3. Wait for rollout to complete
kubectl rollout status deployment/products-api -n production
```

## Common Commands

```bash
# View pods
kubectl get pods -n production -l app=products-api

# View logs
kubectl logs -n production -l app=products-api --tail=100 -f

# Describe pod (for troubleshooting)
kubectl describe pod <pod-name> -n production

# Execute shell in pod
kubectl exec -it <pod-name> -n production -- /bin/sh

# Check ConfigMap
kubectl get configmap products-api-config -n production -o yaml

# Scale manually
kubectl scale deployment products-api --replicas=5 -n production

# Check HPA status
kubectl get hpa -n production

# Delete everything
kubectl delete -f k8s/ -n production
```

## Troubleshooting

### Pods not starting?

```bash
kubectl describe pod <pod-name> -n production
```

Look for:
- ImagePullBackOff → Check image name and registry credentials
- ConfigMapNotFound → Ensure ConfigMap is in same namespace
- CrashLoopBackOff → Check logs: `kubectl logs <pod-name> -n production`

### Can't connect to OTLP collector?

```bash
# Test from pod
kubectl exec -n production <pod-name> -- wget -qO- http://otel-collector:4318

# Check collector is running
kubectl get pods -n observability -l app=otel-collector
```

### Need to see environment variables?

```bash
kubectl exec -n production <pod-name> -- env | grep OTEL
```

## Next Steps

- [ ] Set up Ingress for external access
- [ ] Configure HPA for auto-scaling
- [ ] Set up monitoring with ServiceMonitor
- [ ] Configure CI/CD pipeline
- [ ] Review security settings

See [KUBERNETES.md](KUBERNETES.md) for detailed documentation.
