# Policy AI Agent (Python)

Python implementation of the unified policy validation and enforcement tool powered by OPA and Claude AI.

## Features

- **Multi-domain support**: Kafka, Kubernetes, IaC, CI/CD, AppConfig
- **Policy validation**: OPA/Rego-based policy enforcement
- **AI-powered generation**: Generate policy-compliant configs using Claude
- **Intelligent remediation**: AI suggests fixes for policy violations
- **Python-friendly**: Perfect for scripting, data engineering, and ML workflows

## Quick Start

### Installation

```bash
# Install from source
cd policy-agent-py
pip install -e .

# Or install with development dependencies
pip install -e ".[dev]"
```

### Usage

```bash
# Set API key
export ANTHROPIC_API_KEY="your-key-here"

# Validate a configuration
policy-agent validate --file kafka-topic.yaml

# Generate a configuration with AI
policy-agent generate \
  --domain kafka \
  --requirements "Create a high-throughput user events topic"

# Fix violations interactively
policy-agent fix --file topic.yaml --interactive
```

## Python API Usage

You can also use the policy agent as a Python library:

```python
from policy_agent.agent.orchestrator import Orchestrator
from policy_agent.validators.kafka import KafkaValidator
from policy_agent.policy.engine import PolicyEngine
from policy_agent.ai.claude import ClaudeClient

# Initialize components
engine = PolicyEngine(policy_path="./policies")
validator = KafkaValidator(engine)
ai_client = ClaudeClient(api_key="your-key")

# Create orchestrator
orchestrator = Orchestrator(validators=[validator], ai_client=ai_client)

# Validate
with open("topic.yaml") as f:
    content = f.read()

result = orchestrator.validate(content, domain="kafka")
print(f"Status: {result.status}")
print(f"Violations: {len(result.violations)}")

# Generate
response = orchestrator.generate(
    domain="kafka",
    requirements="High-throughput orders topic"
)
print(response.configuration)
```

## Development

```bash
# Install development dependencies
pip install -r requirements-dev.txt

# Run tests
pytest

# Format code
black policy_agent/
ruff check policy_agent/

# Type check
mypy policy_agent/
```

## Comparison with Go Version

| Feature | Go | Python | Notes |
|---------|----|----|-------|
| CLI Tool | ✅ | ✅ | Both feature-complete |
| Performance | Faster | Good | Go ~2x faster for validation |
| Deployment | Single binary | Pip install | Go easier to distribute |
| Scripting | Limited | Excellent | Python better for automation |
| API Usage | SDK | Native | Python easier to integrate |

## Use Cases for Python Version

- **Data Engineering**: Integrate with Airflow, Databricks workflows
- **ML Pipelines**: Validate configs in ML deployment pipelines
- **Scripting**: Automate policy checks in Python scripts
- **Notebooks**: Use in Jupyter notebooks for exploration
- **Testing**: Write integration tests in pytest

## Project Structure

```
policy_agent/
├── __init__.py
├── cli/                    # CLI commands
│   ├── __init__.py
│   ├── main.py
│   ├── validate.py
│   ├── generate.py
│   └── fix.py
├── agent/                  # Core orchestration
│   ├── __init__.py
│   └── orchestrator.py
├── validators/             # Domain validators
│   ├── __init__.py
│   ├── base.py
│   ├── kafka.py
│   ├── kubernetes.py
│   └── ...
├── policy/                 # OPA integration
│   ├── __init__.py
│   └── engine.py
├── ai/                     # AI integration
│   ├── __init__.py
│   ├── client.py
│   ├── claude.py
│   └── prompts.py
└── types/                  # Shared types
    ├── __init__.py
    ├── result.py
    └── violation.py
```

## Documentation

See the main [COMPLETE_SYSTEM_GUIDE.md](../COMPLETE_SYSTEM_GUIDE.md) for comprehensive documentation.

## License

MIT
