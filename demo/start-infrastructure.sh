#!/bin/bash

# SLIM OpenTelemetry Demo - Quick Start Script
# This script helps run the complete demo with Grafana visualization

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🚀 SLIM OpenTelemetry Demo Setup"
echo "================================"
echo ""

# Check if collector is built
if [ ! -f "$REPO_ROOT/slim-otelcol/slim-otelcol" ]; then
    echo "❌ Collector not found. Building..."
    cd "$REPO_ROOT"
    task collector:build || {
        echo "❌ Failed to build collector"
        echo "Please install Task and try again: https://taskfile.dev/installation/"
        exit 1
    }
    echo "✅ Collector built"
else
    echo "✅ Collector found"
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi
echo "✅ Docker is running"

# Start Prometheus and Grafana
echo ""
echo "📊 Starting Prometheus and Grafana..."
cd "$SCRIPT_DIR"
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 5

# Check if services are running
if docker-compose ps | grep -q "Up"; then
    echo "✅ Prometheus running on http://localhost:9090"
    echo "✅ Grafana running on http://localhost:3000 (admin/admin)"
else
    echo "❌ Services failed to start"
    docker-compose logs
    exit 1
fi

echo ""
echo "================================"
echo "✅ Infrastructure Ready!"
echo "================================"
echo ""
echo "Now start the demo components in separate terminals:"
echo ""
echo "Terminal 1 - OTel Collector:"
echo "  cd $REPO_ROOT/slim-otelcol"
echo "  ./slim-otelcol --config ../demo/collector-config.yaml"
echo ""
echo "Terminal 2 - Monitor Agent:"
echo "  cd $SCRIPT_DIR/monitor_agent"
echo "  go run main.go"
echo ""
echo "Terminal 3 - Special Agent:"
echo "  cd $SCRIPT_DIR/special_agent"
echo "  go run main.go"
echo ""
echo "Terminal 4 - Monitored App:"
echo "  cd $SCRIPT_DIR/monitored_app"
echo "  go run main.go"
echo ""
echo "📊 View real-time metrics:"
echo "  Grafana: http://localhost:3000"
echo "  Prometheus: http://localhost:9090"
echo "  Collector metrics: http://localhost:8889/metrics"
echo ""
echo "To stop infrastructure:"
echo "  cd $SCRIPT_DIR && docker-compose down"
echo ""
