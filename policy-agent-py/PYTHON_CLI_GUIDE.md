# Python CLI Guide - Policy AI Agent

Complete guide to using the Python implementation of the Policy AI Agent.

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [CLI Commands](#cli-commands)
  - [validate](#validate-command)
  - [generate](#generate-command)
  - [fix](#fix-command)
- [Python API](#python-api)
- [Examples](#examples)
- [Development](#development)

## Installation

### Prerequisites

- Python 3.9 or higher
- pip (Python package installer)

### Install from Source

```bash
cd policy-agent-py

# Basic installation
pip install -e .

# With development dependencies
pip install -e ".[dev]"
```

### Verify Installation

```bash
policy-agent --version
# Output: policy-agent, version 0.1.0
```

## Configuration

### Environment Variables

```bash
# Required for AI features
export ANTHROPIC_API_KEY="your-anthropic-api-key-here"

# Optional: Enable debug mode
export DEBUG=1
```

### Validator Configuration

Create custom validator configurations in Python:

```python
from policy_agent.validators.kafka import KafkaConfig

config = KafkaConfig(
    min_replication_factor=3,
    require_compression=True,
    allowed_compression_types=["lz4", "snappy", "zstd"],
    max_retention_days=90,
    warn_retention_days=60
)
```

## CLI Commands

### validate Command

Validate configurations against policies.

#### Syntax

```bash
policy-agent validate --file <path> [OPTIONS]
```

#### Options

| Option | Short | Description | Required |
|--------|-------|-------------|----------|
| `--file` | `-f` | Configuration file to validate | Yes |
| `--domain` | `-d` | Specific domain (auto-detected if not set) | No |
| `--format` | `-o` | Output format: text, json, yaml | No |
| `--policy-path` | `-p` | Path to policy files directory | No |

#### Examples

**Basic validation:**
```bash
policy-agent validate --file kafka-topic.yaml
```

**With specific domain:**
```bash
policy-agent validate --file topic.yaml --domain kafka
```

**JSON output:**
```bash
policy-agent validate --file topic.yaml --format json
```

**YAML output:**
```bash
policy-agent validate --file topic.yaml --format yaml
```

**Custom policy path:**
```bash
policy-agent validate \
  --file topic.yaml \
  --policy-path /path/to/policies
```

#### Output Formats

**Text (default):**
```
✅ kafka-topic.yaml
   Domain: kafka
   Resource: user-events (KafkaTopic)
   Status: PASSED

   Passed (3):
      ✓ kafka.topics.replication.factor
      ✓ kafka.topics.compression
      ✓ kafka.topics.retention

   Duration: 0.123s
```

**JSON:**
```json
{
  "status": "passed",
  "domain": "kafka",
  "resource": "user-events",
  "violations": [],
  "warnings": [],
  "passed": [
    {"policy": "kafka.topics.replication.factor"}
  ],
  "duration": 0.123
}
```

**Exit Codes:**
- `0`: Success (passed or warnings only)
- `1`: Failure (violations found) or error

### generate Command

Generate policy-compliant configurations using AI.

#### Syntax

```bash
policy-agent generate --domain <domain> --requirements <text> [OPTIONS]
```

#### Options

| Option | Short | Description | Required |
|--------|-------|-------------|----------|
| `--domain` | `-d` | Target domain (kafka, kubernetes, etc.) | Yes |
| `--requirements` | `-r` | Natural language requirements | Yes |
| `--output` | `-o` | Output file (default: stdout) | No |
| `--policies` | `-p` | Specific policies (can repeat) | No |
| `--api-key` | | Anthropic API key | No* |
| `--model` | | Claude model to use | No |

*Required if `ANTHROPIC_API_KEY` not set

#### Examples

**Generate Kafka topic:**
```bash
export ANTHROPIC_API_KEY="your-key"

policy-agent generate \
  --domain kafka \
  --requirements "High-throughput user events topic with 30 day retention"
```

**Save to file:**
```bash
policy-agent generate \
  --domain kafka \
  --requirements "Low-latency payments processing topic" \
  --output payments-topic.yaml
```

**With specific policies:**
```bash
policy-agent generate \
  --domain kafka \
  --requirements "Audit log topic" \
  --policies kafka.topics.replication \
  --policies kafka.topics.compression \
  --policies kafka.topics.retention \
  --output audit-logs.yaml
```

**Specify Claude model:**
```bash
policy-agent generate \
  --domain kafka \
  --requirements "User notifications topic" \
  --model claude-sonnet-4-5-20250929 \
  --output notifications.yaml
```

#### Output

```
Generating kafka configuration...
Requirements: High-throughput user events topic
Policies: kafka.topics.replication, kafka.topics.compression

Calling Claude AI API...

================================================================================
GENERATED CONFIGURATION:
================================================================================
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: user-events
  labels:
    app: user-service
spec:
  replicas: 3
  partitions: 12
  config:
    compression.type: "lz4"
    retention.ms: "2592000000"  # 30 days
    min.insync.replicas: "2"

================================================================================
EXPLANATION:
================================================================================
This configuration creates a high-throughput topic optimized for user events...

Next steps:
  1. Review the generated configuration
  2. Validate: policy-agent validate --file user-events.yaml
  3. Deploy if validation passes
```

### fix Command

Fix policy violations using AI.

#### Syntax

```bash
policy-agent fix --file <path> [OPTIONS]
```

#### Options

| Option | Short | Description | Required |
|--------|-------|-------------|----------|
| `--file` | `-f` | Configuration file to fix | Yes |
| `--output` | `-o` | Output file (default: overwrite) | No |
| `--dry-run` | | Show fixes without saving | No |
| `--interactive` | `-i` | Confirm each fix | No |
| `--api-key` | | Anthropic API key | No* |
| `--model` | | Claude model to use | No |

*Required if `ANTHROPIC_API_KEY` not set

#### Examples

**Fix violations (dry run):**
```bash
policy-agent fix --file invalid-topic.yaml --dry-run
```

**Fix and save to new file:**
```bash
policy-agent fix \
  --file invalid-topic.yaml \
  --output fixed-topic.yaml
```

**Interactive mode:**
```bash
policy-agent fix --file topic.yaml --interactive
```

**Fix and overwrite:**
```bash
policy-agent fix --file topic.yaml
```

#### Output

```
Validating invalid-topic.yaml...

❌ Found 3 violation(s):

1. [HIGH] kafka.topics.replication
   Topic 'test-topic' has insufficient replication factor 1 (minimum: 3)
   Field: spec.replicas
   Current: 1
   Expected: 3

2. [HIGH] kafka.topics.replication
   Topic 'test-topic' missing min.insync.replicas configuration
   Field: spec.config.min.insync.replicas
   Expected: 2

3. [MEDIUM] kafka.topics.compression
   Topic 'test-topic' missing compression configuration
   Field: spec.config.compression.type
   Expected: lz4

🤖 Calling Claude AI to fix violations...

================================================================================
FIXED CONFIGURATION:
================================================================================
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: test-topic
spec:
  replicas: 3  # Fixed: Increased from 1 to 3
  partitions: 3
  config:
    retention.ms: "604800000"
    min.insync.replicas: "2"  # Added: Required for durability
    compression.type: "lz4"   # Added: Reduces bandwidth and storage

================================================================================
CHANGES MADE:
================================================================================

Field: spec.replicas
  From: 1
  To: 3
  Why: Insufficient replication factor

Field: spec.config.min.insync.replicas
  From: None
  To: 2
  Why: Missing min.insync.replicas configuration

================================================================================
RE-VALIDATING FIXED CONFIGURATION:
================================================================================
Status: passed
Violations: 0
Warnings: 0
Passed checks: 3

✅ Fixed configuration saved to: fixed-topic.yaml

🎉 All violations fixed successfully!
```

## Python API

### Basic Validation

```python
from policy_agent.validators.kafka import KafkaValidator
import yaml

# Load configuration
with open("topic.yaml") as f:
    data = yaml.safe_load(f)

# Create validator
validator = KafkaValidator()

# Validate
result = validator.validate(data, file_path="topic.yaml")

# Check results
if result.status == "failed":
    print("Violations found:")
    for v in result.violations:
        print(f"  [{v.severity.value}] {v.message}")
else:
    print("✅ All checks passed!")
```

### Custom Validator Configuration

```python
from policy_agent.validators.kafka import KafkaValidator, KafkaConfig

# Custom configuration
config = KafkaConfig(
    min_replication_factor=5,  # Higher than default
    require_compression=True,
    allowed_compression_types=["zstd"],  # Only zstd
    max_retention_days=30,  # Shorter retention
)

validator = KafkaValidator(config=config)
result = validator.validate(data)
```

### Using the Orchestrator

```python
from policy_agent.agent import Orchestrator
from policy_agent.validators.kafka import KafkaValidator
from policy_agent.ai.claude import ClaudeClient

# Initialize components
orchestrator = Orchestrator(
    validators=[KafkaValidator()],
    ai_client=ClaudeClient()
)

# Validate
with open("topic.yaml") as f:
    content = f.read()

result = orchestrator.validate(content, domain="kafka")

# Generate
from policy_agent.types.result import GenerateRequest

response = orchestrator.generate(
    GenerateRequest(
        domain="kafka",
        requirements="Audit log topic with compliance requirements"
    )
)
print(response.configuration)
```

### AI Generation

```python
from policy_agent.ai.claude import ClaudeClient
from policy_agent.types.result import GenerateRequest

# Create client
client = ClaudeClient(
    api_key="your-key",
    model="claude-sonnet-4-5-20250929",
    enable_cache=True
)

# Generate
request = GenerateRequest(
    domain="kafka",
    requirements="""
    Create a topic for user analytics events:
    - Expected 5000 messages/second
    - Need 90 day retention for compliance
    - Critical data, needs high durability
    """,
    policies=[
        "kafka.topics.replication",
        "kafka.topics.compression",
        "kafka.topics.retention"
    ]
)

response = client.generate(request)

# Save configuration
with open("analytics-topic.yaml", "w") as f:
    f.write(response.configuration)

print(f"Explanation: {response.explanation}")
print(f"Policies met: {response.policies_met}")
```

### AI Remediation

```python
from policy_agent.ai.claude import ClaudeClient
from policy_agent.validators.kafka import KafkaValidator
import yaml

# Load and validate
with open("invalid-topic.yaml") as f:
    content = f.read()
    data = yaml.safe_load(content)

validator = KafkaValidator()
result = validator.validate(data)

# Fix if violations found
if result.violations:
    client = ClaudeClient()
    fix_response = client.remediate(
        domain="kafka",
        original_config=content,
        violations=result.violations
    )

    # Review changes
    print("Changes:")
    for change in fix_response.changes:
        print(f"  {change['field']}: {change['from']} → {change['to']}")

    # Save fixed version
    with open("fixed-topic.yaml", "w") as f:
        f.write(fix_response.fixed)
```

## Examples

### Example 1: Validate Multiple Files

```python
import glob
from policy_agent.validators.kafka import KafkaValidator
import yaml

validator = KafkaValidator()

for file_path in glob.glob("configs/*.yaml"):
    print(f"\nValidating: {file_path}")

    with open(file_path) as f:
        data = yaml.safe_load(f)

    result = validator.validate(data, file_path=file_path)

    print(f"  Status: {result.status}")
    print(f"  Violations: {len(result.violations)}")
```

### Example 2: Generate Multiple Topics

```python
from policy_agent.ai.claude import ClaudeClient
from policy_agent.types.result import GenerateRequest

client = ClaudeClient()

topics = [
    ("user-events", "High-volume user activity events"),
    ("orders", "E-commerce order transactions"),
    ("notifications", "User notification delivery"),
]

for name, description in topics:
    request = GenerateRequest(
        domain="kafka",
        requirements=description,
        output=f"{name}.yaml"
    )

    response = client.generate(request)

    with open(f"{name}.yaml", "w") as f:
        f.write(response.configuration)

    print(f"✅ Generated {name}.yaml")
```

### Example 3: CI/CD Integration

```python
#!/usr/bin/env python3
"""CI/CD validation script."""

import sys
import glob
from policy_agent.validators.kafka import KafkaValidator
import yaml

def main():
    validator = KafkaValidator()
    all_passed = True

    for file_path in glob.glob("kafka/*.yaml"):
        with open(file_path) as f:
            data = yaml.safe_load(f)

        result = validator.validate(data, file_path=file_path)

        if result.status == "failed":
            all_passed = False
            print(f"❌ {file_path}: {len(result.violations)} violations")
            for v in result.violations:
                print(f"   - {v.message}")
        else:
            print(f"✅ {file_path}: passed")

    sys.exit(0 if all_passed else 1)

if __name__ == "__main__":
    main()
```

## Development

### Setup Development Environment

```bash
# Clone and install
cd policy-agent-py
pip install -e ".[dev]"

# Install pre-commit hooks (if using pre-commit)
pre-commit install
```

### Code Quality Tools

```bash
# Format code
black policy_agent/

# Lint
ruff check policy_agent/

# Type check
mypy policy_agent/

# Run all checks
black policy_agent/ && ruff check policy_agent/ && mypy policy_agent/
```

### Running Tests

```bash
# Run all tests
pytest

# With coverage
pytest --cov=policy_agent --cov-report=html

# Run specific test file
pytest tests/test_kafka_validator.py

# Run with verbose output
pytest -v
```

### Adding New Validators

1. Create validator class:

```python
# policy_agent/validators/mydomain.py
from policy_agent.validators.base import Validator
from policy_agent.types.result import ValidationResult

class MyDomainValidator(Validator):
    def domain(self) -> str:
        return "mydomain"

    def supported_types(self) -> List[str]:
        return ["MyResource"]

    def validate(self, input_data: Dict[str, Any]) -> ValidationResult:
        # Implementation
        pass
```

2. Add to CLI choices:

```python
# policy_agent/cli/validate.py
@click.option(
    "--domain",
    type=click.Choice(["kafka", "mydomain", ...])
)
```

3. Register with orchestrator:

```python
from policy_agent.validators.mydomain import MyDomainValidator

orchestrator = Orchestrator(
    validators=[KafkaValidator(), MyDomainValidator()]
)
```

### Publishing to PyPI

```bash
# Build package
python -m build

# Upload to Test PyPI
python -m twine upload --repository testpypi dist/*

# Upload to PyPI
python -m twine upload dist/*
```

## Troubleshooting

### API Key Issues

```bash
# Check if API key is set
echo $ANTHROPIC_API_KEY

# Set API key
export ANTHROPIC_API_KEY="your-key-here"

# Or pass directly
policy-agent generate --api-key "your-key" --domain kafka --requirements "..."
```

### Import Errors

```bash
# Reinstall package
pip install -e .

# Check installation
pip show policy-agent
```

### Rate Limiting

If you hit API rate limits, the client will automatically wait. You can adjust:

```python
from policy_agent.ai.claude import ClaudeClient
from datetime import timedelta

client = ClaudeClient(
    requests_per_minute=30,  # Lower rate
    cache_ttl=timedelta(hours=2)  # Longer cache
)
```

## Best Practices

1. **Use caching** - Enable AI response caching to reduce API costs
2. **Validate before deploy** - Always validate generated configs
3. **Version control** - Track policy changes in git
4. **Test policies** - Write tests for custom validators
5. **Monitor costs** - Track AI API usage in production

## Next Steps

- Check out [examples/](./examples/) for complete working examples
- Read [PYTHON_IMPLEMENTATION_COMPLETE.md](../PYTHON_IMPLEMENTATION_COMPLETE.md) for implementation details
- See main [README.md](../README.md) for Go vs Python comparison
- Review [COMPLETE_SYSTEM_GUIDE.md](../COMPLETE_SYSTEM_GUIDE.md) for the full system
