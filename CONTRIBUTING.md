# Contributing to Policy AI Agent

Thank you for your interest in contributing to the Policy AI Agent! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How to Contribute](#how-to-contribute)
- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Submitting Changes](#submitting-changes)
- [Adding New Validators](#adding-new-validators)
- [Testing](#testing)

## Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inspiring community for all. Please be respectful and constructive in all interactions.

### Our Standards

- **Be respectful**: Treat everyone with respect and kindness
- **Be constructive**: Provide helpful feedback and suggestions
- **Be collaborative**: Work together towards common goals
- **Be inclusive**: Welcome contributors of all backgrounds and experience levels

## How to Contribute

There are many ways to contribute:

1. **Report bugs**: Open an issue describing the bug and how to reproduce it
2. **Suggest features**: Open an issue with your feature proposal
3. **Improve documentation**: Fix typos, clarify explanations, add examples
4. **Add validators**: Implement support for new domains (Kubernetes, IaC, etc.)
5. **Add tests**: Improve test coverage
6. **Fix bugs**: Submit pull requests for open issues
7. **Share your experience**: Write blog posts, tutorials, or examples

## Development Setup

### Go Implementation

```bash
# Clone repository
git clone https://github.com/your-org/policy-agent.git
cd policy-agent/policy-agent

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Format code
make fmt

# Lint
make lint
```

### Python Implementation

```bash
cd policy-agent-py

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install in development mode
pip install -e ".[dev]"

# Run tests
pytest

# Format code
black policy_agent/
ruff check policy_agent/

# Type check
mypy policy_agent/
```

## Coding Standards

### Go

- Follow standard Go conventions (`gofmt`, `golint`)
- Write meaningful comments for exported functions
- Use clear, descriptive variable names
- Keep functions focused and concise
- Write tests for new functionality

**Example:**

```go
// ValidateTopic validates a Kafka topic configuration against policies.
// It returns a ValidationResult containing any violations found.
func (v *Validator) ValidateTopic(ctx context.Context, topic *KafkaTopic) (*ValidationResult, error) {
    // Implementation
}
```

### Python

- Follow PEP 8 style guide
- Use type hints for function signatures
- Write docstrings for all public functions
- Use meaningful variable names
- Keep functions focused (single responsibility)
- Format with Black, lint with Ruff

**Example:**

```python
def validate_topic(self, topic: Dict[str, Any]) -> ValidationResult:
    """Validate a Kafka topic configuration.

    Args:
        topic: Kafka topic configuration dictionary

    Returns:
        ValidationResult with violations and warnings

    Raises:
        ValueError: If topic format is invalid
    """
    # Implementation
```

### OPA/Rego Policies

- Write clear, self-documenting policy rules
- Include comments explaining complex logic
- Provide helpful violation messages
- Group related policies together

**Example:**

```rego
package kafka.topics.replication

# Deny if replication factor is below minimum
deny[msg] {
    input.kind == "KafkaTopic"
    rf := input.spec.replicas
    rf < min_replication_factor
    msg := sprintf(
        "Topic '%s' has insufficient replication factor %d (minimum: %d)",
        [input.metadata.name, rf, min_replication_factor]
    )
}
```

## Submitting Changes

### Pull Request Process

1. **Fork the repository** and create a branch from `main`
2. **Make your changes** following the coding standards
3. **Add tests** for new functionality
4. **Update documentation** if needed
5. **Ensure all tests pass** (`make test` for Go, `pytest` for Python)
6. **Format your code** (`make fmt` for Go, `black` for Python)
7. **Commit with clear messages** (see below)
8. **Open a pull request** with a clear description

### Commit Message Format

Use clear, descriptive commit messages:

```
[domain] Brief description of change

Longer explanation of what changed and why (if needed).

Fixes #123
```

**Examples:**

```
[kafka] Add support for connector validation

Implements validation for KafkaConnector resources including
configuration checks and policy enforcement.

Fixes #45
```

```
[docs] Update Python CLI guide with new examples

Adds examples for batch validation and CI/CD integration.
```

```
[fix] Handle missing compression field gracefully

Previously crashed when compression.type was not set.
Now properly reports validation error.

Fixes #78
```

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Performance improvement
- [ ] Code refactoring

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Code follows style guidelines
- [ ] Tests pass locally
- [ ] Documentation updated
- [ ] Commit messages are clear
```

## Adding New Validators

### Go Implementation

1. **Create validator struct:**

```go
// internal/validator/mydomain/validator.go
package mydomain

type Validator struct {
    engine *policy.Engine
    config *Config
}

func (v *Validator) Domain() string {
    return "mydomain"
}

func (v *Validator) SupportedTypes() []string {
    return []string{"MyResource"}
}

func (v *Validator) Validate(ctx context.Context, input *validator.Input) (*types.ValidationResult, error) {
    // Implementation
}
```

2. **Add OPA policies:**

```rego
# policies/mydomain/policy.rego
package mydomain.resource

deny[msg] {
    # Policy rules
}
```

3. **Register validator:**

```go
// cmd/policy-agent/main.go
import "policy-agent/internal/validator/mydomain"

registry.Register(mydomain.NewValidator(engine))
```

### Python Implementation

1. **Create validator class:**

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
```

2. **Add to orchestrator:**

```python
from policy_agent.validators.mydomain import MyDomainValidator

orchestrator = Orchestrator(
    validators=[KafkaValidator(), MyDomainValidator()]
)
```

3. **Update CLI:**

```python
# policy_agent/cli/validate.py
@click.option(
    "--domain",
    type=click.Choice(["kafka", "mydomain", ...])
)
```

## Testing

### Go Tests

```go
// internal/validator/kafka/validator_test.go
func TestKafkaValidator_Replication(t *testing.T) {
    validator := NewValidator(nil, nil)

    topic := &KafkaTopic{
        Spec: KafkaTopicSpec{
            Replicas: 1, // Too low
        },
    }

    result, err := validator.ValidateTopic(context.Background(), topic)
    assert.NoError(t, err)
    assert.Equal(t, "failed", result.Status)
    assert.NotEmpty(t, result.Violations)
}
```

### Python Tests

```python
# tests/test_kafka_validator.py
def test_kafka_validator_replication():
    validator = KafkaValidator()

    data = {
        "kind": "KafkaTopic",
        "metadata": {"name": "test"},
        "spec": {"replicas": 1}  # Too low
    }

    result = validator.validate(data)

    assert result.status == "failed"
    assert len(result.violations) > 0
    assert any("replication" in v.policy for v in result.violations)
```

### Running Tests

```bash
# Go
make test
make test-coverage

# Python
pytest
pytest --cov=policy_agent --cov-report=html
```

## Documentation

- Update README.md for user-facing changes
- Update code comments for implementation changes
- Add examples for new features
- Update CLI help text as needed

## Questions?

- Open an issue for questions
- Join discussions in GitHub Discussions
- Check existing documentation

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to Policy AI Agent!** 🎉
