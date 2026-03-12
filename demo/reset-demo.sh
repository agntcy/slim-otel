#!/bin/bash

# Reset SLIM OTel Demo - Clear all metrics data
# Run this before starting your demo to get a clean slate

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "🔄 Resetting demo environment..."
echo ""

cd "$SCRIPT_DIR"

# Stop services
echo "Stopping services..."
docker-compose stop prometheus grafana

# Remove data volumes to clear all metrics
echo "Clearing metrics data..."
docker volume rm demo_prometheus-data 2>/dev/null || true
docker volume rm demo_grafana-data 2>/dev/null || true

# Restart services fresh
echo "Starting services with fresh data..."
docker-compose up -d prometheus grafana

echo ""
echo "✅ Demo environment reset complete!"
echo ""
echo "⏳ Waiting for services to be ready..."
sleep 5

echo ""
echo "📊 Grafana: http://localhost:3000 (admin/admin)"
echo "📈 Prometheus: http://localhost:9090"
echo ""
echo "Ready to start your demo! 🚀"
echo ""
