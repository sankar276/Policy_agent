# Quick Start Guide - Policy AI Agent

This guide will help you get started with the unified policy AI agent for validating Kafka configurations.

## Prerequisites

- Go 1.22 or higher
- Access to Kafka/Confluent for Kubernetes (CFK) configurations

## Supported Domains

- **Kafka** (Confluent for Kubernetes) - Topics, Connectors, Schema Registry
- **Kubernetes** - Manifests, Deployments, Services, ConfigMaps
- **Infrastructure as Code (IaC)** - Terraform, CloudFormation, Pulumi
- **CI/CD** - GitHub Actions, GitLab CI, Jenkins
- **Application Config** - YAML/JSON application configurations

## Installation

### Option 1: Build from Source

```bash
cd policy-agent
make build
```

The binary will be available at `./bin/policy-agent`.

### Option 2: Run Directly with Go

```bash
cd policy-agent
go run ./cmd/policy-agent/main.go [command]
```

## Project Structure

```
MyWebsite/
├── policy-agent/          # Go implementation
│   ├── cmd/
│   │   └── policy-agent/  # CLI entry point
│   ├── internal/
│   │   ├── agent/         # Orchestrator
│   │   ├── validator/     # Domain validators
│   │   │   └── kafka/     # Kafka validator
│   │   ├── policy/        # OPA engine
│   │   └── config/        # Configuration
│   └── pkg/types/         # Shared types
├── policies/              # Shared OPA/Rego policies
│   └── kafka/topics/
│       ├── replication.rego
│       ├── compression.rego
│       └── retention.rego
├── config/
│   └── policy-agent.yaml  # Configuration file
└── examples/
    └── kafka/             # Example Kafka topics
```

## Quick Test

### 1. Validate an Invalid Topic (Will Fail)

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
Duration: 15ms
======================================================================

❌ VIOLATIONS:

1. [high] kafka.topics.replication
   → Topic 'test-topic' has insufficient replication factor 1 (minimum: 3)
   Field: spec.replicas
   Current: 1
   Expected: 3

   💡 Suggestion: Ensures data durability and high availability across multiple brokers

2. [medium] kafka.topics.compression
   → Topic 'test-topic' missing compression configuration
   Field: spec.config.compression.type
   Current: <nil>
   Expected: lz4

   💡 Suggestion: Reduces network bandwidth and storage costs with minimal CPU overhead

3. [medium] kafka.topics.retention
   → Topic 'test-topic' retention 90.0 days exceeds maximum 90 days

4. [high] kafka.topics.replication
   → Topic 'test-topic' missing min.insync.replicas configuration
```

### 2. Validate a Compliant Topic (Will Pass)

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
Duration: 12ms
======================================================================

✅ PASSED:
   • kafka.topics.replication
   • kafka.topics.compression
   • kafka.topics.retention
```

### 3. Validate a Topic with Warnings

```bash
go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/warning-topic.yaml \
  --config ../config/policy-agent.yaml
```

**Expected Output:**
```
======================================================================
Domain: kafka (KafkaTopic)
Resource: legacy-topic
Status: warning
Duration: 13ms
======================================================================

⚠️  WARNINGS:

1. [medium] kafka.topics.compression
   → Topic 'legacy-topic' uses 'gzip' compression which has higher CPU overhead (consider 'lz4' or 'zstd')

2. [medium] kafka.topics.retention
   → Topic 'legacy-topic' retention 65.0 days is high (consider reviewing storage costs)

✅ PASSED:
   • kafka.topics.replication
```

## Understanding the Policies

### 1. Replication Policy (`policies/kafka/topics/replication.rego`)

Ensures data durability and high availability:
- **Minimum replication factor**: 3 (configurable)
- **Requires min.insync.replicas**: At least 2 replicas must acknowledge writes
- **Prevents invalid configurations**: min.insync.replicas must be < replication factor

### 2. Compression Policy (`policies/kafka/topics/compression.rego`)

Reduces network and storage costs:
- **Requires compression**: Enabled by default
- **Allowed types**: lz4, snappy, zstd, gzip
- **Recommended**: lz4 (best balance of speed and compression)

### 3. Retention Policy (`policies/kafka/topics/retention.rego`)

Manages data lifecycle and storage:
- **Maximum retention**: 90 days (configurable)
- **Warning threshold**: 60 days
- **Minimum retention**: 1 day (prevents accidental data loss)

## Configuration

Edit `config/policy-agent.yaml` to customize policy parameters:

```yaml
domains:
  kafka:
    min_replication_factor: 3
    require_min_insync_replicas: true
    min_insync_replicas_value: 2
    require_compression: true
    allowed_compression_types:
      - "lz4"
      - "snappy"
      - "zstd"
    max_retention_days: 90
    warn_retention_days: 60
```

## Next Steps

1. **Add More Domains**: Implement validators for Kubernetes, IaC, CI/CD, etc.
2. **Enable AI Features**: Configure Claude API for auto-generation and remediation
3. **Git Hooks**: Install pre-commit hooks for automatic validation
4. **CI/CD Integration**: Add to GitHub Actions or GitLab CI pipelines
5. **Python Implementation**: Use the Python CLI for scripting workflows

## Troubleshooting

### "Policy engine failed to load policies"
- Ensure the `policy_path` in config points to the correct `policies/` directory
- Check that .rego files are valid using `opa check`

### "No validator found for domain: kafka"
- Verify that the Kafka validator is registered in the orchestrator
- Check that domain is in `enabled_domains` in the config file

### "Failed to parse YAML"
- Ensure your Kafka topic YAML is valid
- Check for proper indentation and structure

## Contributing

See the implementation plan in `.claude/plans/` for details on:
- Architecture and design
- Adding new domain validators
- Implementing AI integration
- Testing strategy

---

**Built with:**
- Go 1.22+
- OPA (Open Policy Agent)
- Claude AI (Anthropic)
- Cobra (CLI framework)
