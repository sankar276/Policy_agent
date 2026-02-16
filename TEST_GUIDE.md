# Testing Guide - Policy AI Agent

## Prerequisites

Before testing, ensure you have:
- **Go 1.22+** installed ([download here](https://go.dev/dl/))
- Terminal access

## Quick Verification

Check your Go installation:
```bash
go version
# Expected: go version go1.22.0 or higher
```

## Test Scenarios

### Test 1: Invalid Topic (Should Fail) ❌

This test validates a Kafka topic with multiple policy violations.

```bash
cd /Users/ramasankarmolleti/Desktop/MyWebsite/policy-agent

go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/invalid-topic.yaml \
  --config ../config/policy-agent.yaml
```

**Expected Output:**
```
✓ Loaded policies from: ../policies
✓ Registered validators: [kafka]

Validating: ../examples/kafka/invalid-topic.yaml

======================================================================
Domain: kafka (KafkaTopic)
Resource: test-topic
Status: failed
Duration: ~15ms
======================================================================

❌ VIOLATIONS:

1. [high] kafka.topics.replication
   → Topic 'test-topic' has insufficient replication factor 1 (minimum: 3)
   Field: spec.replicas
   Current: 1
   Expected: 3

   💡 Suggestion: Ensures data durability and high availability across multiple brokers

2. [high] kafka.topics.replication
   → Topic 'test-topic' missing min.insync.replicas configuration
   Field: spec.config.min.insync.replicas
   Current: <nil>
   Expected: 2

   💡 Suggestion: Guarantees that writes are acknowledged by at least this many replicas

3. [medium] kafka.topics.compression
   → Topic 'test-topic' missing compression configuration
   Field: spec.config.compression.type
   Current: <nil>
   Expected: lz4

   💡 Suggestion: Reduces network bandwidth and storage costs with minimal CPU overhead

4. [medium] kafka.topics.retention
   → Topic 'test-topic' retention 90.0 days exceeds maximum 90 days

Error: validation failed with 4 violation(s)
exit status 1
```

**What This Tests:**
- ✓ Replication factor < 3
- ✓ Missing min.insync.replicas
- ✓ Missing compression
- ✓ Retention at maximum limit

---

### Test 2: Valid Topic (Should Pass) ✅

This test validates a fully compliant Kafka topic.

```bash
go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/valid-topic.yaml \
  --config ../config/policy-agent.yaml
```

**Expected Output:**
```
✓ Loaded policies from: ../policies
✓ Registered validators: [kafka]

Validating: ../examples/kafka/valid-topic.yaml

======================================================================
Domain: kafka (KafkaTopic)
Resource: user-events-topic
Status: passed
Duration: ~12ms
======================================================================

✅ PASSED:
   • kafka.topics.replication
   • kafka.topics.compression
   • kafka.topics.retention
```

**What This Tests:**
- ✓ Replication factor = 3 (meets minimum)
- ✓ min.insync.replicas = 2 (configured)
- ✓ Compression = lz4 (optimal)
- ✓ Retention = 30 days (within limits)

---

### Test 3: Topic with Warnings (Should Pass with Warnings) ⚠️

This test validates a topic that passes but has suboptimal configurations.

```bash
go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/warning-topic.yaml \
  --config ../config/policy-agent.yaml
```

**Expected Output:**
```
✓ Loaded policies from: ../policies
✓ Registered validators: [kafka]

Validating: ../examples/kafka/warning-topic.yaml

======================================================================
Domain: kafka (KafkaTopic)
Resource: legacy-topic
Status: warning
Duration: ~13ms
======================================================================

⚠️  WARNINGS:

1. [medium] kafka.topics.compression
   → Topic 'legacy-topic' uses 'gzip' compression which has higher CPU overhead (consider 'lz4' or 'zstd')

2. [medium] kafka.topics.retention
   → Topic 'legacy-topic' retention 65.0 days is high (consider reviewing storage costs)

✅ PASSED:
   • kafka.topics.replication
```

**What This Tests:**
- ✓ Suboptimal compression (gzip vs lz4)
- ✓ High retention warning (65 days > 60 day threshold)
- ✓ Core policies still pass

---

## Building the Binary

Instead of `go run`, you can build a binary for faster execution:

```bash
cd policy-agent

# Build
make build

# Run with binary
./bin/policy-agent validate --file ../examples/kafka/valid-topic.yaml --config ../config/policy-agent.yaml
```

## Testing Different Configurations

### Test with Custom Policy Settings

Edit `config/policy-agent.yaml` to adjust policy parameters:

```yaml
domains:
  kafka:
    min_replication_factor: 5        # Change from 3 to 5
    max_retention_days: 30           # Change from 90 to 30
    require_compression: false       # Disable compression requirement
```

Then re-run validation to see how the results change.

### Test Policy Loading

Verify policies are loading correctly:

```bash
# Check what policies exist
find ../policies -name "*.rego"

# Expected output:
# ../policies/kafka/topics/replication.rego
# ../policies/kafka/topics/compression.rego
# ../policies/kafka/topics/retention.rego
```

### Test with Invalid YAML

Create a test file with invalid YAML to verify error handling:

```bash
echo "invalid: yaml: content::" > test-invalid.yaml

go run ./cmd/policy-agent/main.go validate \
  --file test-invalid.yaml \
  --config ../config/policy-agent.yaml

# Expected: Error message about YAML parsing failure
```

## Debugging

### Enable Verbose Logging

If you encounter issues, you can add debug output. Edit `config/policy-agent.yaml`:

```yaml
agent:
  log_level: "debug"  # Change from "info" to "debug"
```

### Common Issues

**Issue: "Policy engine failed to load policies"**
```bash
# Check policy path is correct
ls -la ../policies/kafka/topics/

# Verify Rego syntax (if you have OPA installed)
opa check ../policies/kafka/topics/*.rego
```

**Issue: "No validator found for domain: kafka"**
- Verify `kafka` is in `enabled_domains` in config
- Check that the Kafka validator is registered in the code

**Issue: "Failed to read file"**
```bash
# Verify file exists and is readable
ls -l ../examples/kafka/invalid-topic.yaml
cat ../examples/kafka/invalid-topic.yaml
```

## Performance Testing

Test validation speed with multiple runs:

```bash
# Time a single validation
time go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/valid-topic.yaml \
  --config ../config/policy-agent.yaml

# Expected: < 2 seconds (includes compilation)
# With binary: < 100ms
```

## Next Steps After Testing

Once validation is working:

1. **Try Your Own Kafka Topics**
   ```bash
   go run ./cmd/policy-agent/main.go validate \
     --file /path/to/your/kafka-topic.yaml \
     --config ../config/policy-agent.yaml
   ```

2. **Add Custom Policies**
   - Create new `.rego` files in `policies/kafka/topics/`
   - Policies are automatically loaded

3. **Customize Policy Parameters**
   - Edit `config/policy-agent.yaml`
   - Adjust thresholds for your environment

4. **Enable AI Features** (Next phase)
   - Implement Claude API client
   - Test config generation
   - Test auto-remediation

## Success Criteria

✅ All three test scenarios run successfully
✅ Policy violations are detected accurately
✅ Warnings are displayed for suboptimal configs
✅ Valid configurations pass all checks
✅ Performance is < 100ms for validation

## Troubleshooting Commands

```bash
# Check Go installation
go version

# Check dependencies
cd policy-agent
go mod download
go mod tidy

# Build to verify no compilation errors
make build

# Run with verbose output (when implemented)
go run ./cmd/policy-agent/main.go validate --file test.yaml --verbose
```

---

**Need Help?**
- Check [QUICKSTART.md](./QUICKSTART.md) for setup instructions
- See [PROJECT_STATUS.md](./PROJECT_STATUS.md) for current implementation status
- Review [config/policy-agent.yaml](./config/policy-agent.yaml) for configuration options
