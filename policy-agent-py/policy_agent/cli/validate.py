"""Validate command implementation."""

import sys
from pathlib import Path

import click
import yaml

from policy_agent.agent import Orchestrator
from policy_agent.validators.kafka import KafkaValidator
from policy_agent.policy.engine import PolicyEngine


@click.command()
@click.option(
    "--file",
    "-f",
    "file_path",
    type=click.Path(exists=True, path_type=Path),
    required=True,
    help="Configuration file to validate",
)
@click.option(
    "--domain",
    "-d",
    type=click.Choice(["kafka", "kubernetes", "iac", "cicd", "appconfig"], case_sensitive=False),
    help="Specific domain to validate (auto-detected if not specified)",
)
@click.option(
    "--format",
    "-o",
    "output_format",
    type=click.Choice(["text", "json", "yaml"], case_sensitive=False),
    default="text",
    help="Output format",
)
@click.option(
    "--policy-path",
    "-p",
    type=click.Path(exists=True, path_type=Path),
    help="Path to policy files directory",
)
def validate(file_path, domain, output_format, policy_path):
    """Validate configuration against policies.

    Examples:

      \b
      # Validate a Kafka topic
      policy-agent validate --file topic.yaml

      \b
      # Validate with specific domain
      policy-agent validate --file topic.yaml --domain kafka

      \b
      # Get JSON output
      policy-agent validate --file topic.yaml --format json
    """
    try:
        # Read file
        with open(file_path) as f:
            content = f.read()

        # Create orchestrator
        policy_engine = None
        if policy_path:
            policy_engine = PolicyEngine(str(policy_path))

        # Initialize validators (currently only Kafka)
        kafka_validator = KafkaValidator(policy_engine=policy_engine)

        orchestrator = Orchestrator(
            validators=[kafka_validator],
            policy_engine=policy_engine,
        )

        # Validate
        result = orchestrator.validate(content, domain=domain, file_path=str(file_path))

        # Output results
        if output_format == "json":
            _output_json(result)
        elif output_format == "yaml":
            _output_yaml(result)
        else:
            _output_text(result, file_path)

        # Exit with appropriate code
        if result.status == "failed":
            sys.exit(1)
        elif result.status == "warning":
            sys.exit(0)  # Warnings don't fail the build
        else:
            sys.exit(0)

    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        sys.exit(1)


def _output_text(result, file_path):
    """Output validation results in text format."""
    import click

    # Header
    status_icon = "✅" if result.status == "passed" else "⚠️" if result.status == "warning" else "❌"
    click.echo(f"\n{status_icon} {file_path}")
    click.echo(f"   Domain: {result.domain}")
    click.echo(f"   Resource: {result.resource} ({result.resource_type})")
    click.echo(f"   Status: {result.status.upper()}")

    # Violations
    if result.violations:
        click.echo(f"\n   Violations ({len(result.violations)}):")
        for i, v in enumerate(result.violations, 1):
            severity_color = {
                "critical": "red",
                "high": "red",
                "medium": "yellow",
                "low": "blue",
            }.get(v.severity.value, "white")

            click.echo(
                f"\n   {i}. "
                + click.style(f"[{v.severity.value.upper()}]", fg=severity_color, bold=True)
                + f" {v.policy}"
            )
            click.echo(f"      → {v.message}")

            if v.field:
                click.echo(f"      Field: {v.field}")
                if v.current_value is not None:
                    click.echo(f"      Current: {v.current_value}")
                if v.expected_value is not None:
                    click.echo(f"      Expected: {v.expected_value}")

            if v.remediation and v.remediation.suggestion:
                click.echo(f"      💡 {v.remediation.suggestion}")

    # Warnings
    if result.warnings:
        click.echo(f"\n   Warnings ({len(result.warnings)}):")
        for i, w in enumerate(result.warnings, 1):
            click.echo(f"\n   {i}. " + click.style(f"[WARNING]", fg="yellow") + f" {w.policy}")
            click.echo(f"      → {w.message}")

    # Passed checks
    if result.passed:
        click.echo(f"\n   Passed ({len(result.passed)}):")
        for p in result.passed:
            click.echo(f"      ✓ {p.policy}")

    # Summary
    click.echo(f"\n   Duration: {result.duration:.3f}s")
    click.echo()


def _output_json(result):
    """Output validation results in JSON format."""
    import json
    from datetime import datetime

    def json_serializer(obj):
        if isinstance(obj, datetime):
            return obj.isoformat()
        if hasattr(obj, "__dict__"):
            return obj.__dict__
        return str(obj)

    output = {
        "status": result.status,
        "domain": result.domain,
        "resource": result.resource,
        "resource_type": result.resource_type,
        "file": result.file,
        "violations": [
            {
                "policy": v.policy,
                "severity": v.severity.value,
                "message": v.message,
                "field": v.field,
                "current_value": v.current_value,
                "expected_value": v.expected_value,
                "remediation": {
                    "auto_fixable": v.remediation.auto_fixable,
                    "suggestion": v.remediation.suggestion,
                }
                if v.remediation
                else None,
            }
            for v in result.violations
        ],
        "warnings": [
            {
                "policy": w.policy,
                "severity": w.severity.value,
                "message": w.message,
                "field": w.field,
            }
            for w in result.warnings
        ],
        "passed": [{"policy": p.policy, "message": p.message} for p in result.passed],
        "timestamp": result.timestamp.isoformat(),
        "duration": result.duration,
    }

    click.echo(json.dumps(output, indent=2, default=json_serializer))


def _output_yaml(result):
    """Output validation results in YAML format."""
    output = {
        "status": result.status,
        "domain": result.domain,
        "resource": result.resource,
        "resource_type": result.resource_type,
        "file": result.file,
        "violations": [
            {
                "policy": v.policy,
                "severity": v.severity.value,
                "message": v.message,
                "field": v.field,
                "current_value": v.current_value,
                "expected_value": v.expected_value,
            }
            for v in result.violations
        ],
        "warnings": [
            {
                "policy": w.policy,
                "severity": w.severity.value,
                "message": w.message,
            }
            for w in result.warnings
        ],
        "passed": [p.policy for p in result.passed],
        "duration": result.duration,
    }

    click.echo(yaml.dump(output, default_flow_style=False))
