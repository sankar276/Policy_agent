# Policy AI Agent (Go)

Unified policy validation, generation, and enforcement tool powered by OPA and Claude AI.

## Features

- **Multi-domain support**: Kafka, Kubernetes, IaC, CI/CD, AppConfig
- **Policy validation**: OPA/Rego-based policy enforcement
- **AI-powered generation**: Generate policy-compliant configs using Claude
- **Intelligent remediation**: AI suggests fixes for policy violations
- **Multiple integrations**: CLI, Git hooks, CI/CD, Kubernetes webhooks

## Quick Start

```bash
# Install
go install github.com/policy-agent/policy-agent/cmd/policy-agent@latest

# Validate a configuration
policy-agent validate --file kafka-topic.yaml

# Generate a configuration with AI
export ANTHROPIC_API_KEY=your-api-key
policy-agent generate --domain kafka \
  --requirements "Create a high-throughput user events topic"

# Fix violations interactively
policy-agent fix --file topic.yaml --interactive
```

## Project Structure

- `cmd/` - Entry points (CLI and webhook server)
- `internal/` - Internal packages
  - `agent/` - Core orchestration logic
  - `validator/` - Domain-specific validators
  - `policy/` - OPA engine integration
  - `ai/` - Claude API client
  - `webhook/` - Kubernetes admission controller
- `pkg/` - Public API packages
- `policies/` - Shared OPA/Rego policies

## Development

```bash
# Build
make build

# Run tests
make test

# Run locally
go run cmd/policy-agent/main.go validate --file test.yaml
```

## Documentation

See [docs/](./docs) for detailed documentation.
