"""Fix command implementation."""

import os
import sys
from pathlib import Path

import click
import yaml

from policy_agent.ai.claude import ClaudeClient
from policy_agent.agent import Orchestrator
from policy_agent.validators.kafka import KafkaValidator
from policy_agent.types.result import FixRequest


@click.command()
@click.option(
    "--file",
    "-f",
    "file_path",
    type=click.Path(exists=True, path_type=Path),
    required=True,
    help="Configuration file to fix",
)
@click.option(
    "--output",
    "-o",
    "output_file",
    type=click.Path(path_type=Path),
    help="Output file for fixed configuration (default: overwrite original)",
)
@click.option(
    "--dry-run",
    is_flag=True,
    help="Show fixes without saving",
)
@click.option(
    "--interactive",
    "-i",
    is_flag=True,
    help="Confirm each fix before applying",
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
def fix(file_path, output_file, dry_run, interactive, api_key, model):
    """Fix policy violations using AI.

    Analyzes policy violations in a configuration file and uses
    Claude AI to suggest and apply fixes.

    Examples:

      \b
      # Fix violations (dry run)
      export ANTHROPIC_API_KEY="your-key"
      policy-agent fix --file topic.yaml --dry-run

      \b
      # Fix and save to new file
      policy-agent fix --file topic.yaml --output topic-fixed.yaml

      \b
      # Interactive fixing
      policy-agent fix --file topic.yaml --interactive

      \b
      # Fix and overwrite original
      policy-agent fix --file topic.yaml
    """
    if not api_key:
        click.echo(
            "Error: ANTHROPIC_API_KEY not set. "
            "Either set the environment variable or use --api-key",
            err=True,
        )
        sys.exit(1)

    try:
        # Read original file
        with open(file_path) as f:
            original_content = f.read()
            data = yaml.safe_load(original_content)

        # Validate to find violations
        click.echo(f"Validating {file_path}...")

        kafka_validator = KafkaValidator()
        orchestrator = Orchestrator(validators=[kafka_validator])

        result = orchestrator.validate(original_content, file_path=str(file_path))

        # Check for violations
        if not result.violations:
            click.echo("✅ No violations found. Configuration is already compliant!")
            sys.exit(0)

        # Display violations
        click.echo(f"\n❌ Found {len(result.violations)} violation(s):")
        for i, v in enumerate(result.violations, 1):
            severity_color = {
                "critical": "red",
                "high": "red",
                "medium": "yellow",
                "low": "blue",
            }.get(v.severity.value, "white")

            click.echo(
                f"\n{i}. "
                + click.style(f"[{v.severity.value.upper()}]", fg=severity_color, bold=True)
                + f" {v.policy}"
            )
            click.echo(f"   {v.message}")
            if v.field:
                click.echo(f"   Field: {v.field}")
                if v.current_value is not None:
                    click.echo(f"   Current: {v.current_value}")
                if v.expected_value is not None:
                    click.echo(f"   Expected: {v.expected_value}")

        # Interactive confirmation
        if interactive and not dry_run:
            if not click.confirm("\nAttempt to fix these violations with AI?"):
                click.echo("Aborted.")
                sys.exit(0)

        # Use AI to fix
        click.echo("\n🤖 Calling Claude AI to fix violations...")

        client = ClaudeClient(api_key=api_key, model=model)
        fix_response = client.remediate(result.domain, original_content, result.violations)

        # Display fixed configuration
        click.echo("\n" + "=" * 80)
        click.echo("FIXED CONFIGURATION:")
        click.echo("=" * 80)
        click.echo(fix_response.fixed)

        # Display changes
        if fix_response.changes:
            click.echo("\n" + "=" * 80)
            click.echo("CHANGES MADE:")
            click.echo("=" * 80)
            for change in fix_response.changes:
                click.echo(f"\nField: {change['field']}")
                click.echo(f"  From: {change['from']}")
                click.echo(f"  To:   {change['to']}")
                if change.get('reason'):
                    click.echo(f"  Why:  {change['reason']}")

        # Display explanation
        click.echo("\n" + "=" * 80)
        click.echo("EXPLANATION:")
        click.echo("=" * 80)
        click.echo(fix_response.explanation)

        # Re-validate
        click.echo("\n" + "=" * 80)
        click.echo("RE-VALIDATING FIXED CONFIGURATION:")
        click.echo("=" * 80)

        fixed_data = yaml.safe_load(fix_response.fixed)
        fixed_result = kafka_validator.validate(fixed_data)

        click.echo(f"Status: {fixed_result.status}")
        click.echo(f"Violations: {len(fixed_result.violations)}")
        click.echo(f"Warnings: {len(fixed_result.warnings)}")
        click.echo(f"Passed checks: {len(fixed_result.passed)}")

        # Save if not dry run
        if dry_run:
            click.echo("\n🔍 DRY RUN - No changes saved")
        else:
            # Determine output file
            save_path = output_file if output_file else file_path

            # Interactive confirmation for overwrite
            if not output_file and interactive:
                if not click.confirm(f"\nOverwrite {file_path} with fixed version?"):
                    click.echo("Aborted.")
                    sys.exit(0)

            # Save
            with open(save_path, "w") as f:
                f.write(fix_response.fixed)

            click.echo(f"\n✅ Fixed configuration saved to: {save_path}")

            # Success message
            if fixed_result.status == "passed":
                click.echo("\n🎉 All violations fixed successfully!")
            elif fixed_result.status == "warning":
                click.echo("\n⚠️  Violations fixed, but some warnings remain.")
            else:
                click.echo("\n⚠️  Some violations could not be fixed automatically.")

    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        import traceback

        if os.getenv("DEBUG"):
            traceback.print_exc()
        sys.exit(1)
