"""Main CLI entry point."""

import click

from policy_agent.cli.validate import validate
from policy_agent.cli.generate import generate
from policy_agent.cli.fix import fix


@click.group()
@click.version_option(version="0.1.0", prog_name="policy-agent")
def cli():
    """Policy AI Agent - Unified policy validation and enforcement.

    Validate, generate, and fix policy-compliant configurations across
    multiple domains (Kafka, Kubernetes, IaC, CI/CD, AppConfig).
    """
    pass


# Register commands
cli.add_command(validate)
cli.add_command(generate)
cli.add_command(fix)


if __name__ == "__main__":
    cli()
