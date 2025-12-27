#!/bin/bash

# Test script to verify HTTP metrics are being exported

echo "Testing HTTP metrics..."
echo ""

# Make some requests to generate metrics
echo "1. Making test requests..."
for i in {1..5}; do
    curl -s http://localhost:8080/products > /dev/null &
done

# Wait for all requests to complete
wait

echo "2. Checking /metrics endpoint for OpenTelemetry metrics..."
echo ""
curl -s http://localhost:8080/metrics | grep -E "http_server_|http_client_" | head -20

echo ""
echo "3. Looking for active requests metrics..."
curl -s http://localhost:8080/metrics | grep -i "active"

echo ""
echo "Done! Check your OTLP collector (Tempo/Alloy) for:"
echo "  - http.server.active_requests"
echo "  - http.server.duration"
echo "  - http.server.request.size"
echo "  - http.server.response.size"
