# Claude AI Integration Examples

This guide demonstrates how to use the Claude AI-powered features of the policy agent for intelligent configuration generation and automatic remediation.

## Prerequisites

1. **Anthropic API Key**: Get your API key from [console.anthropic.com](https://console.anthropic.com)
2. **Set Environment Variable**:
   ```bash
   export ANTHROPIC_API_KEY="your-api-key-here"
   ```
3. **Go 1.22+**: Installed and working

## Feature 1: AI-Powered Configuration Generation

Generate policy-compliant configurations from natural language requirements.

### Example 1: Generate a Kafka Topic for User Events

```bash
cd policy-agent

go run ./cmd/policy-agent/main.go generate \
  --domain kafka \
  --requirements "Create a high-throughput topic for user login events.
                   Should retain data for 7 days and be highly available.
                   Expected throughput: 10,000 events/second" \
  --output ../examples/kafka/generated-user-login.yaml
```

**Expected Output:**
```
🤖 Generating kafka configuration using Claude AI...

Requirements: Create a high-throughput topic for user login events...

⏳ Calling Claude API...
✅ Configuration generated!

======================================================================
GENERATED CONFIGURATION:
======================================================================
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: user-login-events
  labels:
    app: authentication-service
    environment: production
    owner: security-team
spec:
  # High availability: 3 replicas across brokers
  replicas: 3

  # High throughput: 24 partitions for 10k events/sec
  partitions: 24

  config:
    # LZ4 compression: best balance for high throughput
    compression.type: "lz4"

    # 7 days retention as requested
    retention.ms: "604800000"

    # min.insync.replicas: ensures durability
    min.insync.replicas: "2"

    # Segment size optimized for high throughput
    segment.ms: "3600000"  # 1 hour segments

    # Cleanup policy: delete old data
    cleanup.policy: "delete"

----------------------------------------------------------------------
EXPLANATION:
----------------------------------------------------------------------
This configuration is optimized for high-throughput user login events:

1. **24 Partitions**: Distributes load for 10k events/sec across consumers
2. **RF=3**: Ensures high availability and fault tolerance
3. **LZ4 Compression**: Minimal CPU overhead with good compression ratio
4. **7-day Retention**: Meets requirement exactly (604800000 ms = 7 days)
5. **1-hour Segments**: Balances storage efficiency with compaction overhead

All policies satisfied:
✓ kafka.topics.replication (RF=3, min.insync=2)
✓ kafka.topics.compression (lz4)
✓ kafka.topics.retention (7 days within 90-day limit)

✅ Saved to: ../examples/kafka/generated-user-login.yaml
```

### Example 2: Generate a Compact Topic for Configuration

```bash
go run ./cmd/policy-agent/main.go generate \
  --domain kafka \
  --requirements "Create a compacted topic for storing application configuration.
                   Each key represents a config parameter, latest value wins.
                   Should be highly available and compressed." \
  --output ../examples/kafka/generated-app-config.yaml
```

**Claude will generate:**
- Replication factor: 3
- Compaction policy instead of delete
- Appropriate retention and compression
- Comments explaining compaction strategy

### Example 3: Generate Multiple Related Topics

```bash
go run ./cmd/policy-agent/main.go generate \
  --domain kafka \
  --requirements "Create 3 topics for an e-commerce system:
                   1. orders-created: Order creation events (retain 30 days)
                   2. orders-completed: Completion events (retain 90 days)
                   3. orders-failed: Failed orders (retain 7 days, low throughput)
                   All should be production-ready and compliant." \
  --output ../examples/kafka/generated-ecommerce-topics.yaml
```

---

## Feature 2: Automatic Violation Fixing

Automatically fix policy violations in existing configurations.

### Example 1: Fix Invalid Topic (Interactive Mode)

Start with a non-compliant topic:

```bash
# First, let's see the violations
go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/invalid-topic.yaml
```

Output shows 4 violations. Now fix them:

```bash
go run ./cmd/policy-agent/main.go fix \
  --file ../examples/kafka/invalid-topic.yaml \
  --interactive \
  --output ../examples/kafka/fixed-topic.yaml
```

**Expected Output:**
```
🔧 Fixing policy violations in: ../examples/kafka/invalid-topic.yaml

📋 Validating current configuration...
Found 4 violation(s)

======================================================================
Domain: kafka (KafkaTopic)
Resource: test-topic
Status: failed
======================================================================

❌ VIOLATIONS:

1. [high] kafka.topics.replication
   → Topic 'test-topic' has insufficient replication factor 1 (minimum: 3)

2. [high] kafka.topics.replication
   → Topic 'test-topic' missing min.insync.replicas configuration

3. [medium] kafka.topics.compression
   → Topic 'test-topic' missing compression configuration

4. [medium] kafka.topics.retention
   → Topic 'test-topic' retention 90.0 days exceeds maximum 90 days

⏳ Generating fixes using Claude AI...

✅ Fixes generated!

======================================================================
FIXED CONFIGURATION:
======================================================================
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: test-topic
  labels:
    app: test-app
spec:
  # FIXED: Increased from 1 to 3 for high availability
  replicas: 3

  partitions: 3

  config:
    # ADDED: LZ4 compression for bandwidth/storage optimization
    compression.type: "lz4"

    # FIXED: Reduced from 90 to 30 days to comply with limits
    retention.ms: "2592000000"  # 30 days

    # ADDED: Ensures write durability
    min.insync.replicas: "2"

    segment.ms: "604800000"  # 7 days

----------------------------------------------------------------------
CHANGES MADE:
----------------------------------------------------------------------
Fixed 4 policy violations:

1. Replication Factor: 1 → 3
   - Ensures high availability and fault tolerance
   - Meets minimum requirement for production

2. Added min.insync.replicas: 2
   - Guarantees writes are acknowledged by 2+ replicas
   - Prevents data loss during broker failures

3. Added compression.type: "lz4"
   - Reduces network bandwidth by ~60-70%
   - Reduces storage costs
   - LZ4 offers best speed/compression balance

4. Retention: 90 days → 30 days
   - Complies with maximum retention policy
   - Reduces storage costs
   - Still provides adequate retention for most use cases

All configurations preserve existing settings (partitions, segment.ms, etc.)
that were already compliant.

Apply these fixes? (y/n): y
✅ Fixed configuration saved to: ../examples/kafka/fixed-topic.yaml
```

### Example 2: Auto-Fix Without Interaction

```bash
# Automatically fix and overwrite original file
go run ./cmd/policy-agent/main.go fix \
  --file ../examples/kafka/invalid-topic.yaml
```

### Example 3: Fix Warnings

```bash
go run ./cmd/policy-agent/main.go fix \
  --file ../examples/kafka/warning-topic.yaml \
  --output ../examples/kafka/optimized-topic.yaml
```

**Claude will:**
- Change gzip compression to lz4 (better performance)
- Adjust retention if needed
- Explain optimization reasoning

---

## Feature 3: Policy Explanation (Coming Soon)

```bash
go run ./cmd/policy-agent/main.go explain \
  --policy kafka.topics.replication

# Will explain:
# - What replication factor means
# - Why it's important for availability
# - Common mistakes and best practices
# - Examples of good/bad configurations
```

---

## Advanced Usage

### 1. Generate with Custom Context

```bash
go run ./cmd/policy-agent/main.go generate \
  --domain kafka \
  --requirements "Create a topic for CDC (Change Data Capture) from PostgreSQL.
                   Source: users table (updates frequently)
                   Consumers: Data warehouse, analytics platform
                   Must support exactly-once semantics
                   Expected size: 100MB/day" \
  --output cdc-users.yaml
```

### 2. Batch Fix Multiple Files

```bash
# Fix all invalid topics
for file in invalid-topic-*.yaml; do
  go run ./cmd/policy-agent/main.go fix --file "$file"
done
```

### 3. Generate and Validate in One Go

```bash
# Generate
go run ./cmd/policy-agent/main.go generate \
  --domain kafka \
  --requirements "High-throughput orders topic" \
  --output orders.yaml

# Validate immediately
go run ./cmd/policy-agent/main.go validate --file orders.yaml

# Should show: ✅ All policies passed
```

---

## Configuration Options

### AI Settings in `config/policy-agent.yaml`

```yaml
ai:
  provider: "anthropic"
  api_key_env: "ANTHROPIC_API_KEY"
  model: "claude-sonnet-4-5-20250929"

  features:
    auto_remediation: true
    generation: true
    explanation: true

  rate_limiting:
    requests_per_minute: 50
    burst: 10

  cache:
    enabled: true
    ttl: "1h"
```

**Rate Limiting:**
- Prevents exceeding API quotas
- Default: 50 requests/minute
- Configurable per your API tier

**Caching:**
- Caches AI responses for 1 hour
- Reduces API costs for repeated queries
- Can be disabled for always-fresh responses

---

## Cost Optimization Tips

1. **Use Cache**: Enable caching to avoid repeated API calls for same requirements
2. **Batch Requests**: Generate multiple related configs in one prompt
3. **Validate First**: Only use AI for actual violations (not already-compliant configs)
4. **Pick the Right Model**:
   - `claude-sonnet-4-5`: Best quality (default)
   - `claude-haiku-4-5`: Faster, cheaper for simple fixes

---

## Troubleshooting

### "ANTHROPIC_API_KEY not set"
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

### "claude API error: rate limit exceeded"
Reduce `requests_per_minute` in config or wait before retrying.

### "Generation produced invalid YAML"
The AI occasionally generates malformed YAML. Re-run or manually adjust.

### "Fixed config still has violations"
Validate the fixed config:
```bash
go run ./cmd/policy-agent/main.go validate --file fixed-topic.yaml
```

If violations persist, the AI may have misunderstood. Try providing more specific requirements.

---

## Examples Directory

After running these examples, you'll have:

```
examples/kafka/
├── invalid-topic.yaml              # Original (violations)
├── valid-topic.yaml                # Hand-crafted (passes)
├── warning-topic.yaml              # Suboptimal (warnings)
├── fixed-topic.yaml                # AI-fixed (violations → passes)
├── generated-user-login.yaml       # AI-generated (high throughput)
├── generated-app-config.yaml       # AI-generated (compaction)
└── generated-ecommerce-topics.yaml # AI-generated (multi-topic)
```

---

## Next Steps

1. **Try Generation**: Create your first AI-generated config
2. **Fix Existing Configs**: Use `fix` command on your actual topics
3. **Integrate in CI/CD**: Add validation and AI fixes to your pipelines
4. **Customize Prompts**: Edit `internal/ai/prompts.go` for domain-specific needs
5. **Add New Domains**: Extend to Kubernetes, Terraform, etc.

**Happy policy-compliant configuration management!** 🚀
