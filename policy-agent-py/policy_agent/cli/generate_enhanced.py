"""Enhanced generate command implementation."""

import sys
from pathlib import Path

import click

from policy_agent.ai.client import ClaudeClient
from policy_agent.types.result import GenerateRequest


@click.command()
@click.option(
    "--domain",
    required=True,
    type=click.Choice(["kafka", "kubernetes", "iac", "cicd", "gitops"], case_sensitive=False),
    help="Target domain for generation",
)
@click.option(
    "--requirements",
    "-r",
    required=True,
    type=str,
    help="Requirements in natural language",
)
@click.option(
    "--output",
    "-o",
    type=click.Path(path_type=Path),
    help="Output file path (default: print to stdout)",
)
@click.option(
    "--api-key",
    envvar="ANTHROPIC_API_KEY",
    help="Anthropic API key (or set ANTHROPIC_API_KEY env var)",
)
def generate(domain, requirements, output, api_key):
    """Generate policy-compliant configurations using AI.

    Uses Claude AI to generate configuration files that comply with
    all relevant policies for the specified domain.

    Examples:

      \b
      # Generate a Kafka topic
      policy-agent generate --domain kafka \\
        --requirements "High-throughput user events topic with compression"

      \b
      # Generate and save to file
      policy-agent generate --domain kubernetes \\
        --requirements "Production deployment with 3 replicas" \\
        --output deployment.yaml

      \b
      # Generate GitOps Application
      policy-agent generate --domain gitops \\
        --requirements "ArgoCD application for staging with auto-sync"
    """
    if not api_key:
        click.echo("Error: Anthropic API key required. Set ANTHROPIC_API_KEY environment variable.", err=True)
        sys.exit(1)

    try:
        click.echo(f"🤖 Generating {domain} configuration using Claude AI...\n")
        click.echo(f"Requirements: {requirements}\n")

        # Create AI client
        client = ClaudeClient(api_key=api_key)

        # Prepare request
        request = GenerateRequest(
            domain=domain,
            requirements=requirements,
        )

        # Generate
        click.echo("⏳ Calling Claude API...")
        response = client.generate(request)

        click.echo("✅ Configuration generated!\n")
        click.echo("=" * 70)
        click.echo("GENERATED CONFIGURATION:")
        click.echo("=" * 70)
        click.echo(response.configuration)
        click.echo()

        if response.explanation:
            click.echo("-" * 70)
            click.echo("EXPLANATION:")
            click.echo("-" * 70)
            click.echo(response.explanation)
            click.echo()

        # Save to file if output specified
        if output:
            output.write_text(response.configuration)
            click.echo(f"✅ Saved to: {output}")

    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        sys.exit(1)
