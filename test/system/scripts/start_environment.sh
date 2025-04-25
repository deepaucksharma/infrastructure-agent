#!/bin/bash
# Script to start the system test environment for infrastructure agent tests

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TEST_DIR="$SCRIPT_DIR/.."
CONFIG_DIR="$TEST_DIR/config"

# Create config directory if it doesn't exist
mkdir -p "$CONFIG_DIR"

echo "Creating OpenTelemetry Collector configuration..."
cat > "$CONFIG_DIR/otel-collector-config.yaml" << EOL
receivers:
  otlp:
    protocols:
      grpc:
      http:

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"
    namespace: "nria"
  logging:
    verbosity: detailed

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheus, logging]
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging]
EOL

echo "Creating Prometheus configuration..."
cat > "$CONFIG_DIR/prometheus.yml" << EOL
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'otel-collector'
    scrape_interval: 5s
    static_configs:
      - targets: ['otel-collector:8889']
EOL

echo "Starting test environment with Docker Compose..."
docker-compose -f "$TEST_DIR/docker-compose.yml" up -d

echo "Waiting for services to be ready..."
timeout=60
elapsed=0
while ! docker exec otel-collector-test wget --spider -q http://localhost:8888/metrics; do
  if [ "$elapsed" -ge "$timeout" ]; then
    echo "Timed out waiting for OpenTelemetry Collector to be ready."
    exit 1
  fi
  
  sleep 1
  elapsed=$((elapsed + 1))
  echo -n "."
done

echo ""
echo "Test environment is ready!"
echo "- OpenTelemetry Collector: localhost:4317 (gRPC), localhost:4318 (HTTP)"
echo "- Prometheus: http://localhost:9090/"
echo ""
echo "To run system tests:"
echo "  make test-system-short"
echo ""
echo "To stop the test environment when done:"
echo "  ./scripts/stop_environment.sh"