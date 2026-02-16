"""Generate command implementation."""

import os
import sys

import click

from policy_agent.ai.claude import ClaudeClient
from policy_agent.types.result import GenerateRequest


@click.command()
@click.option(
    "--domain",
    "-d",
    type=click.Choice(["kafka", "kubernetes", "iac", "cicd", "appconfig"], case_sensitive=False),
    required=True,
    help="Target domain for generation",
)
@click.option(
    "--requirements",
    "-r",
    required=True,
    help="Natural language requirements for the configuration",
)
@click.option(
    "--output",
    "-o",
    "output_file",
    type=click.Path(),
    help="Output file (default: stdout)",
)
@click.option(
    "--policies",
    "-p",
    multiple=True,
    help="Specific policies to satisfy (can be specified multiple times)",
)
@click.option(
    "--api-key",
    envvar="ANTHROPIC_API_KEY",
    help="Anthropic API key (or set ANTHROPIC_API_KEY env var)",
)
@click.option(
    "--model",
    default="claude-sonnet-4-5-20250929",
    help="Claude model to use",
)
def generate(domain, requirements, output_file, policies, api_key, model):
    """Generate policy-compliant configuration using AI.

    Uses Claude AI to generate production-ready configurations that
    satisfy all policy requirements.

    Examples:

      \b
      # Generate a Kafka topic
      export ANTHROPIC_API_KEY="your-key"
      policy-agent generate \\
        --domain kafka \\
        --requirements "High-throughput user events topic with 30 day retention"

      \b
      # Generate and save to file
      policy-agent generate \\
        --domain kafka \\
        --requirements "Low-latency payments topic" \\
        --output payments-topic.yaml

      \b
      # Specify policies
      policy-agent generate \\
        --domain kafka \\
        --requirements "Audit log topic" \\
        --policies kafka.topics.replication \\
        --policies kafka.topics.compression
    """
    if not api_key:
        click.echo(
            "Error: ANTHROPIC_API_KEY not set. "
            "Either set the environment variable or use --api-key",
            err=True,
        )
        sys.exit(1)

    try:
        # Create AI client
        client = ClaudeClient(api_key=api_key, model=model)

        # Create request
        request = GenerateRequest(
            domain=domain,
            requirements=requirements,
            policies=list(policies) if policies else [],
            output=output_file,
        )

        # Display generation info
        click.echo(f"Generating {domain} configuration...")
        click.echo(f"Requirements: {requirements}")
        if policies:
            click.echo(f"Policies: {', '.join(policies)}")
        click.echo("\nCalling Claude AI API...\n")

        # Generate
        response = client.generate(request)

        # Output configuration
        if output_file:
            with open(output_file, "w") as f:
                f.write(response.configuration)
            click.echo(f"✅ Configuration saved to: {output_file}")
        else:
            click.echo("=" * 80)
            click.echo("GENERATED CONFIGURATION:")
            click.echo("=" * 80)
            click.echo(response.configuration)

        # Show explanation
        click.echo("\n" + "=" * 80)
        click.echo("EXPLANATION:")
        click.echo("=" * 80)
        click.echo(response.explanation)

        # Show policies met
        if response.policies_met:
            click.echo("\n" + "=" * 80)
            click.echo("POLICIES SATISFIED:")
            click.echo("=" * 80)
            for policy in response.policies_met:
                click.echo(f"  ✓ {policy}")

        # Suggest validation
        if output_file:
            click.echo(f"\nNext steps:")
            click.echo(f"  1. Review the generated configuration")
            click.echo(f"  2. Validate: policy-agent validate --file {output_file}")
            click.echo(f"  3. Deploy if validation passes")

    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        import traceback

        if os.getenv("DEBUG"):
            traceback.print_exc()
        sys.exit(1)
