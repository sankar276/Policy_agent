"""Enhanced fix command implementation."""

import sys
from pathlib import Path

import click
import yaml

from policy_agent.ai.client import ClaudeClient
from policy_agent.cli.output import OutputFormatter
from policy_agent.cli.registry import setup_validators, get_validator_for_file
from policy_agent.policy.engine import PolicyEngine
from policy_agent.types.result import FixRequest


@click.command()
@click.option(
    "--file",
    "-f",
    "file_path",
    required=True,
    type=click.Path(exists=True, path_type=Path),
    help="File to fix",
)
@click.option(
    "--output",
    "-o",
    type=click.Path(path_type=Path),
    help="Output file (default: overwrite original)",
)
@click.option(
    "--interactive",
    "-i",
    is_flag=True,
    help="Interactive mode (preview before applying)",
)
@click.option(
    "--api-key",
    envvar="ANTHROPIC_API_KEY",
    help="Anthropic API key (or set ANTHROPIC_API_KEY env var)",
)
@click.option(
    "--policy-path",
    "-p",
    type=click.Path(exists=True, path_type=Path),
    default="./policies",
    help="Path to policy files directory",
)
def fix(file_path, output, interactive, api_key, policy_path):
    """Fix policy violations using AI.

    Automatically fixes policy violations in configuration files using
    Claude AI to suggest and apply corrections.

    Examples:

      \b
      # Fix violations interactively
      policy-agent fix --file invalid-topic.yaml --interactive

      \b
      # Auto-fix and save
      policy-agent fix --file deployment.yaml

      \b
      # Fix and save to different file
      policy-agent fix --file old.yaml --output new.yaml
    """
    if not api_key:
        click.echo("Error: Anthropic API key required. Set ANTHROPIC_API_KEY environment variable.", err=True)
        sys.exit(1)

    try:
        click.echo(f"🔧 Fixing policy violations in: {file_path}\n")

        # Read file
        with open(file_path) as f:
            content = f.read()

        # Initialize policy engine
        policy_engine = None
        if policy_path and policy_path.exists():
            policy_engine = PolicyEngine(str(policy_path))

        # Setup validators
        validators = setup_validators(policy_engine=policy_engine)

        # Validate first to find violations
        click.echo("📋 Validating current configuration...")

        # Parse content
        data = yaml.safe_load(content)

        # Get validator
        validator = get_validator_for_file(str(file_path), validators)
        if not validator:
            # Try to detect from content
            kind = data.get("kind", "").lower() if isinstance(data, dict) else ""
            if "kafkatopic" in kind:
                validator = validators.get("kafka")
            elif "deployment" in kind or "pod" in kind:
                validator = validators.get("kubernetes")
            else:
                raise ValueError(f"Could not determine validator for {file_path}")

        # Validate
        result = validator.validate(data, file_path=str(file_path))

        if len(result.violations) == 0:
            click.echo("✅ No violations found! Configuration is already compliant.")
            return

        click.echo(f"Found {len(result.violations)} violation(s)\n")

        # Display violations
        formatter = OutputFormatter(format_type="text", color=True)
        output_text = formatter.format_result(result)
        click.echo(output_text)

        # Use AI to fix
        click.echo("\n⏳ Generating fixes using Claude AI...")

        client = ClaudeClient(api_key=api_key)

        request = FixRequest(
            file=content,
            interactive=interactive,
        )

        response = client.fix(request)

        click.echo("\n✅ Fixes generated!\n")
        click.echo("=" * 70)
        click.echo("FIXED CONFIGURATION:")
        click.echo("=" * 70)
        click.echo(response.fixed)
        click.echo()

        if response.explanation:
            click.echo("-" * 70)
            click.echo("CHANGES MADE:")
            click.echo("-" * 70)
            click.echo(response.explanation)
            click.echo()

        # Interactive mode: ask before saving
        if interactive:
            if not click.confirm("Apply these fixes?"):
                click.echo("Fixes not applied.")
                return

        # Determine output file
        if not output:
            output = file_path  # Overwrite original

        # Write fixed configuration
        output.write_text(response.fixed)
        click.echo(f"✅ Fixed configuration saved to: {output}")

    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        sys.exit(1)
