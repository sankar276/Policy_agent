#!/bin/bash
# Quick test script for Policy AI Agent

set -e

echo "=========================================="
echo "Policy AI Agent - Validation Tests"
echo "=========================================="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.22+ from https://go.dev/dl/"
    exit 1
fi

echo "✓ Go version: $(go version)"
echo ""

# Navigate to policy-agent directory
cd "$(dirname "$0")/policy-agent"

echo "Running validation tests..."
echo ""

# Test 1: Invalid Topic (should fail)
echo "----------------------------------------"
echo "Test 1: Invalid Topic (Expected: FAIL)"
echo "----------------------------------------"
go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/invalid-topic.yaml \
  --config ../config/policy-agent.yaml \
  2>&1 || echo "✓ Failed as expected"
echo ""

# Test 2: Valid Topic (should pass)
echo "----------------------------------------"
echo "Test 2: Valid Topic (Expected: PASS)"
echo "----------------------------------------"
go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/valid-topic.yaml \
  --config ../config/policy-agent.yaml
echo ""

# Test 3: Warning Topic (should pass with warnings)
echo "----------------------------------------"
echo "Test 3: Warning Topic (Expected: WARN)"
echo "----------------------------------------"
go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/warning-topic.yaml \
  --config ../config/policy-agent.yaml
echo ""

echo "=========================================="
echo "All tests completed!"
echo "=========================================="
