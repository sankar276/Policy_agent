#  Policy AI Agent

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev)
[![Python Version](https://img.shields.io/badge/Python-3.9+-3776AB?style=flat&logo=python)](https://python.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> AI-powered policy validation, generation, and enforcement for DevOps/GitOps workflows

Validate, generate, and fix configurations across multiple infrastructure domains (Kafka, Kubernetes, IaC, CI/CD) using OPA/Rego policies and Claude AI.

---

## 📋 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Development](#development)
- [Documentation](#documentation)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

---

## 🎯 Overview

The Policy AI Agent helps platform teams ensure compliance, security, and best practices across their entire technology stack. It combines:

- **OPA/Rego** for policy-as-code validation
- **Claude AI** for intelligent config generation and remediation
- **Multi-domain support** for Kafka, Kubernetes, IaC, CI/CD, and more

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    User Interface                            │
├──────────┬───────────┬────────────┬──────────────────────────┤
│ CLI Tool │ Git Hooks │ CI/CD API  │ K8s Admission Webhook    │
│ (Go/Py)  │ (Go/Py)   │ (Go)       │ (Go)                     │
└──────────┴───────────┴────────────┴──────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                   Core Engine                                │
│   Validator Orchestrator + Policy Engine + AI Service       │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│            Domain Validators + OPA Policies                  │
│     Kafka | Kubernetes | IaC | CI/CD | AppConfig           │
└─────────────────────────────────────────────────────────────┘
```

---

## ✨ Features

### Core Capabilities

- ✅ **Policy Validation** - Validate configs against OPA/Rego policies
- ✅ **AI Generation** - Generate policy-compliant configs from natural language
- ✅ **AI Remediation** - Automatically fix policy violations
- ✅ **Multi-Domain** - Kafka, Kubernetes, IaC, CI/CD, AppConfig
- ✅ **Dual Implementation** - Go (performance) + Python (flexibility)

### Supported Domains

| Domain | Status | Policies | Description |
|--------|--------|----------|-------------|
| **Kafka (CFK)** | ✅ Complete | 15+ policies | Topics, connectors, schema registry |
| **Kubernetes** | ✅ Complete | 10+ policies | Deployments, pods, services, config maps |
| **IaC (Terraform)** | ✅ Complete | 12+ policies | Provider versioning, state encryption, security |
| **CI/CD** | ✅ Complete | 8+ policies | GitHub Actions, GitLab CI workflows |
| **GitOps** | ✅ Complete | 10+ policies | Flux CD, ArgoCD applications |

### Kafka Policies (Example)

- **Replication**: Min RF=3, min.insync.replicas=2
- **Compression**: Required (lz4, snappy, zstd)
- **Retention**: Max 90 days, warn at 60+ days

---

## 🚀 Quick Start

### Prerequisites

**For Go Implementation:**
- Go 1.22 or higher ([download](https://go.dev/dl/))

**For Python Implementation:**
- Python 3.9 or higher ([download](https://python.org/downloads/))

**For AI Features (Optional):**
- Anthropic API key ([get here](https://console.anthropic.com))

### 5-Minute Quick Start

```bash
# Clone the repository (or navigate to your project directory)
git clone https://github.com/your-org/policy-agent.git
cd policy-agent

# Option 1: Go (Recommended for CLI tools)
cd policy-agent
go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/valid-topic.yaml \
  --config ../config/policy-agent.yaml

# Option 2: Python (Recommended for scripting)
cd policy-agent-py
pip install -e .
python << 'EOF'
import yaml
from policy_agent.validators.kafka import KafkaValidator

with open("../examples/kafka/valid-topic.yaml") as f:
    data = yaml.safe_load(f)

validator = KafkaValidator()
result = validator.validate(data)
print(f"Status: {result.status}")
EOF
```

---

## 📦 Installation

### Go Implementation

#### Step 1: Install Go

**macOS:**
```bash
brew install go
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install golang-go
```

**Verify:**
```bash
go version
# Should output: go version go1.22.x or higher
```

#### Step 2: Build the Binary

```bash
cd policy-agent

# Download dependencies
go mod download

# Build
make build

# Or build manually
go build -o bin/policy-agent ./cmd/policy-agent

# Verify
./bin/policy-agent --version
```

#### Step 3: Install Globally (Optional)

```bash
# Install to $GOPATH/bin
go install ./cmd/policy-agent

# Add to PATH (if not already)
export PATH=$PATH:$(go env GOPATH)/bin

# Now available system-wide
policy-agent --help
```

### Python Implementation

#### Step 1: Install Python

**macOS:**
```bash
brew install python@3.11
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install python3.11 python3-pip
```

**Verify:**
```bash
python3 --version
# Should output: Python 3.9.x or higher
```

#### Step 2: Create Virtual Environment (Recommended)

```bash
cd policy-agent-py

# Create venv
python3 -m venv venv

# Activate
source venv/bin/activate  # Linux/macOS
# or
venv\Scripts\activate     # Windows

# Verify
which python
# Should show path to venv
```

#### Step 3: Install Package

```bash
# Install in development mode
pip install -e .

# Or install with dev dependencies
pip install -e ".[dev]"

# Verify
python -c "from policy_agent import Orchestrator; print('✅ Installed')"
```

### AI Features Setup (Optional)

#### Step 1: Get Anthropic API Key

1. Visit [console.anthropic.com](https://console.anthropic.com)
2. Sign up or log in
3. Navigate to API Keys
4. Create a new API key

#### Step 2: Set Environment Variable

**Linux/macOS:**
```bash
# Add to ~/.bashrc or ~/.zshrc
export ANTHROPIC_API_KEY="sk-ant-your-key-here"

# Or create .env file
echo 'ANTHROPIC_API_KEY=sk-ant-your-key-here' > .env
```

**Verify:**
```bash
echo $ANTHROPIC_API_KEY
# Should display your API key
```

---

## 💻 Usage

### Command-Line Interface (Go)

#### 1. Validate Configuration

```bash
cd policy-agent

# Validate single file
go run ./cmd/policy-agent/main.go validate \
  --file ../examples/kafka/invalid-topic.yaml \
  --config ../config/policy-agent.yaml

# Validate with specific domain
go run ./cmd/policy-agent/main.go validate \
  --file topic.yaml \
  --domain kafka

# JSON output
go run ./cmd/policy-agent/main.go validate \
  --file topic.yaml \
  --format json
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
```

#### 2. Generate Configuration with AI

```bash
# Set API key
export ANTHROPIC_API_KEY="your-key-here"

# Generate Kafka topic
go run ./cmd/policy-agent/main.go generate \
  --domain kafka \
  --requirements "Create a high-throughput topic for user login events.
                   Retain for 7 days. Expected: 10,000 events/second." \
  --output user-login-events.yaml

# Generate to stdout
go run ./cmd/policy-agent/main.go generate \
  --domain kafka \
  --requirements "CDC topic for user table changes"
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
spec:
  replicas: 3
  partitions: 24
  config:
    compression.type: "lz4"
    retention.ms: "604800000"
    min.insync.replicas: "2"

✅ Saved to: user-login-events.yaml
```

#### 3. Fix Violations Automatically

```bash
# Interactive mode (preview before applying)
go run ./cmd/policy-agent/main.go fix \
  --file ../examples/kafka/invalid-topic.yaml \
  --interactive \
  --output fixed-topic.yaml

# Auto-fix (no confirmation)
go run ./cmd/policy-agent/main.go fix \
  --file invalid-topic.yaml

# Dry-run (show changes without applying)
go run ./cmd/policy-agent/main.go fix \
  --file invalid-topic.yaml \
  --dry-run
```

### Python API

#### 1. Basic Validation

```python
#!/usr/bin/env python3
"""Validate Kafka topic using Python API."""

import yaml
from policy_agent.validators.kafka import KafkaValidator, KafkaConfig

# Load configuration
with open("examples/kafka/invalid-topic.yaml") as f:
    data = yaml.safe_load(f)

# Create validator with custom config
validator = KafkaValidator(
    config=KafkaConfig(
        min_replication_factor=3,
        require_compression=True,
        max_retention_days=90,
    )
)

# Validate
result = validator.validate(data, file_path="invalid-topic.yaml")

# Display results
print(f"Status: {result.status}")
print(f"Violations: {len(result.violations)}")
print(f"Warnings: {len(result.warnings)}")
print(f"Passed: {len(result.passed)}")

# Show violations
for v in result.violations:
    print(f"\n[{v.severity.value}] {v.policy}")
    print(f"  → {v.message}")
    if v.remediation:
        print(f"  💡 {v.remediation.suggestion}")
```

#### 2. Using Orchestrator

```python
#!/usr/bin/env python3
"""Orchestrated validation example."""

from policy_agent.agent import Orchestrator
from policy_agent.validators.kafka import KafkaValidator
from policy_agent.policy.engine import PolicyEngine

# Initialize components
policy_engine = PolicyEngine(policy_path="policies")
kafka_validator = KafkaValidator(policy_engine)

# Create orchestrator
orchestrator = Orchestrator(
    validators=[kafka_validator],
)

# Validate from file
with open("examples/kafka/valid-topic.yaml") as f:
    content = f.read()

result = orchestrator.validate(content, domain="kafka")
print(f"Result: {result.status}")
```

#### 3. Batch Validation

```python
#!/usr/bin/env python3
"""Validate multiple files."""

import glob
from policy_agent.agent import Orchestrator
from policy_agent.validators.kafka import KafkaValidator

orchestrator = Orchestrator(validators=[KafkaValidator()])

for file_path in glob.glob("examples/kafka/*.yaml"):
    print(f"\n{'='*60}")
    print(f"Validating: {file_path}")
    print('='*60)

    with open(file_path) as f:
        content = f.read()

    result = orchestrator.validate(content, file_path=file_path)

    print(f"Status: {result.status}")
    print(f"Violations: {len(result.violations)}")
    print(f"Duration: {result.duration:.3f}s")
```

---

## ⚙️ Configuration

### Main Configuration File

Edit `config/policy-agent.yaml`:

```yaml
# Policy engine
policy:
  engine: "opa"
  policy_path: "./policies"
  enabled_domains:
    - kafka
    - kubernetes
  enforcement_mode: "strict"  # strict|warn|permissive

# AI configuration
ai:
  provider: "anthropic"
  api_key_env: "ANTHROPIC_API_KEY"
  model: "claude-sonnet-4-5-20250929"
  features:
    auto_remediation: true
    generation: true
  rate_limiting:
    requests_per_minute: 50
  cache:
    enabled: true
    ttl: "1h"

# Domain-specific policies
domains:
  kafka:
    min_replication_factor: 3
    require_compression: true
    allowed_compression_types: ["lz4", "snappy", "zstd"]
    max_retention_days: 90
    warn_retention_days: 60
```

### Environment Variables

```bash
# Required for AI features
export ANTHROPIC_API_KEY="sk-ant-your-key"

# Optional overrides
export POLICY_PATH="./custom-policies"
export LOG_LEVEL="debug"
export ENFORCEMENT_MODE="warn"
```

---

## 🔧 Development

### Project Structure

```
.
├── policy-agent/          # Go implementation
│   ├── cmd/
│   │   └── policy-agent/  # CLI entry point
│   ├── internal/
│   │   ├── agent/         # Orchestrator
│   │   ├── validator/     # Validators
│   │   ├── policy/        # OPA engine
│   │   └── ai/            # AI client
│   └── pkg/types/         # Shared types
│
├── policy-agent-py/       # Python implementation
│   └── policy_agent/
│       ├── agent/         # Orchestrator
│       ├── validators/    # Validators
│       ├── policy/        # OPA engine
│       └── ai/            # AI client
│
├── policies/              # Shared OPA/Rego policies
│   └── kafka/topics/
│       ├── replication.rego
│       ├── compression.rego
│       └── retention.rego
│
├── config/
│   └── policy-agent.yaml  # Configuration
│
└── examples/
    └── kafka/             # Example configs
```

### Development Setup (Go)

```bash
cd policy-agent

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Download dependencies
go mod download
go mod tidy

# Run linter
golangci-lint run

# Run tests
go test ./...

# Build
make build

# Run locally
go run ./cmd/policy-agent/main.go --help
```

### Development Setup (Python)

```bash
cd policy-agent-py

# Create venv
python3 -m venv venv
source venv/bin/activate

# Install with dev dependencies
pip install -e ".[dev]"

# Format code
black policy_agent/

# Lint
ruff check policy_agent/

# Type check
mypy policy_agent/

# Run tests
pytest

# Run tests with coverage
pytest --cov=policy_agent --cov-report=html
```

### Running Tests

#### Go Tests

```bash
cd policy-agent

# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test
go test -v ./internal/validator/kafka -run TestKafkaValidator
```

#### Python Tests

```bash
cd policy-agent-py

# Run all tests
pytest

# Run with verbose output
pytest -v

# Run specific test file
pytest tests/test_kafka_validator.py

# Run with coverage
pytest --cov=policy_agent --cov-report=term-missing
```

### Adding a New Policy

#### Step 1: Create Rego Policy

```bash
# Create new policy file
cat > policies/kafka/topics/partitions.rego <<'EOF'
package kafka.topics.partitions

import future.keywords.if

default min_partitions := 3
default max_partitions := 100

deny[msg] {
    input.kind == "KafkaTopic"
    partitions := input.spec.partitions
    partitions < min_partitions
    msg := sprintf(
        "Topic '%s' has too few partitions %d (minimum: %d)",
        [input.metadata.name, partitions, min_partitions]
    )
}

deny[msg] {
    input.kind == "KafkaTopic"
    partitions := input.spec.partitions
    partitions > max_partitions
    msg := sprintf(
        "Topic '%s' has too many partitions %d (maximum: %d)",
        [input.metadata.name, partitions, max_partitions]
    )
}
EOF
```

#### Step 2: Test Policy

```bash
# Test with OPA
opa check policies/kafka/topics/partitions.rego

# Test evaluation
opa eval -d policies/kafka/topics/partitions.rego \
  -i test-input.json \
  'data.kafka.topics.partitions.deny'
```

#### Step 3: Update Validator

Policies are automatically loaded by the policy engine. No code changes needed!

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| **[COMPLETE_SYSTEM_GUIDE.md](COMPLETE_SYSTEM_GUIDE.md)** | Comprehensive system guide |
| **[QUICKSTART.md](QUICKSTART.md)** | Quick start tutorial |
| **[TEST_GUIDE.md](TEST_GUIDE.md)** | Testing instructions |
| **[AI_EXAMPLES.md](examples/AI_EXAMPLES.md)** | AI usage examples |
| **[AI_INTEGRATION_SUMMARY.md](AI_INTEGRATION_SUMMARY.md)** | AI technical details |
| **[PYTHON_IMPLEMENTATION_COMPLETE.md](PYTHON_IMPLEMENTATION_COMPLETE.md)** | Python implementation |
| **[PROJECT_STATUS.md](PROJECT_STATUS.md)** | Implementation status |

---

## 🐛 Troubleshooting

### Common Issues

#### "Go is not installed"

```bash
# Install Go
brew install go  # macOS
# or download from https://go.dev/dl/

# Verify
go version
```

#### "ANTHROPIC_API_KEY not set"

```bash
# Set environment variable
export ANTHROPIC_API_KEY="sk-ant-your-key-here"

# Or create .env file
echo 'ANTHROPIC_API_KEY=sk-ant-your-key' > .env

# Verify
echo $ANTHROPIC_API_KEY
```

#### "Policy engine failed to load policies"

```bash
# Check policy path exists
ls -la policies/

# Verify Rego syntax
opa check policies/kafka/topics/*.rego

# Check config file
cat config/policy-agent.yaml | grep policy_path
```

#### "Module import error" (Go)

```bash
cd policy-agent

# Clean and rebuild
go clean
go mod download
go mod tidy

# Rebuild
go build ./cmd/policy-agent
```

#### "Import error" (Python)

```bash
cd policy-agent-py

# Reinstall in development mode
pip install -e .

# Or with dependencies
pip install -r requirements.txt
pip install -e .
```

### Debug Mode

**Go:**
```bash
# Enable debug logging
LOG_LEVEL=debug go run ./cmd/policy-agent/main.go validate --file topic.yaml
```

**Python:**
```python
import logging
logging.basicConfig(level=logging.DEBUG)
```

### Getting Help

```bash
# Go CLI help
./bin/policy-agent --help
./bin/policy-agent validate --help

# Python help
python -c "from policy_agent import Orchestrator; help(Orchestrator)"
```

---

## 🤝 Contributing

We welcome contributions! Here's how to get started:

### 1. Fork & Clone

```bash
git clone https://github.com/your-username/policy-agent.git
cd policy-agent
```

### 2. Create Branch

```bash
git checkout -b feature/my-new-feature
```

### 3. Make Changes

```bash
# Make your changes
# Add tests
# Update documentation
```

### 4. Run Tests

```bash
# Go
cd policy-agent
go test ./...

# Python
cd policy-agent-py
pytest
```

### 5. Commit & Push

```bash
git add .
git commit -m "Add: my new feature"
git push origin feature/my-new-feature
```

### 6. Create Pull Request

Open a PR on GitHub with:
- Description of changes
- Test results
- Documentation updates

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

Built with:
- [Open Policy Agent (OPA)](https://www.openpolicyagent.org/)
- [Claude AI (Anthropic)](https://www.anthropic.com/)
- [Cobra (CLI)](https://github.com/spf13/cobra)
- [Click (Python CLI)](https://click.palletsprojects.com/)

---

## 📞 Support

- **Documentation**: See [docs](./COMPLETE_SYSTEM_GUIDE.md)
- **Issues**: [GitHub Issues](https://github.com/policy-agent/policy-agent/issues)
- **Examples**: See [examples/](./examples/)

---

**Built with ❤️ for DevOps Teams**
