"""Enhanced main CLI entry point."""

import click

from policy_agent.cli.validate_enhanced import validate
from policy_agent.cli.generate_enhanced import generate
from policy_agent.cli.fix_enhanced import fix
from policy_agent.cli.policy_cmd import policy
from policy_agent.cli.hooks_cmd import hooks


@click.group()
@click.version_option(version="0.1.0", prog_name="policy-agent")
def cli():
    """Policy AI Agent - Unified policy validation and enforcement.

    AI-powered policy validation, generation, and enforcement across
    multiple infrastructure domains.

    \b
    Supported Domains:
      • Kafka - Topics, Connectors, Schema Registry
      • Kubernetes - Deployments, Pods, Services
      • IaC - Terraform, CloudFormation
      • CI/CD - GitHub Actions, GitLab CI
      • GitOps - Flux CD, ArgoCD

    \b
    Examples:
      # Validate a file
      policy-agent validate --file config.yaml

      # Generate with AI
      policy-agent generate --domain kafka \\
        --requirements "High-throughput topic with compression"

      # Fix violations
      policy-agent fix --file invalid.yaml --interactive

      # Install Git hooks
      policy-agent hooks install --all

    For more information, visit: https://policy-agent.io/docs
    """
    pass


# Register commands
cli.add_command(validate)
cli.add_command(generate)
cli.add_command(fix)
cli.add_command(policy)
cli.add_command(hooks)


if __name__ == "__main__":
    cli()
