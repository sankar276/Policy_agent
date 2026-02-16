"""Enhanced validate command implementation."""

import sys
from pathlib import Path
from typing import List

import click
import yaml

from policy_agent.cli.output import OutputFormatter
from policy_agent.cli.registry import setup_validators, get_validator_for_file, detect_format
from policy_agent.policy.engine import PolicyEngine
from policy_agent.types.result import ValidationResult


@click.command()
@click.option(
    "--file",
    "-f",
    "file_path",
    type=click.Path(exists=True, path_type=Path),
    help="Configuration file to validate",
)
@click.option(
    "--dir",
    "-d",
    "directory",
    type=click.Path(exists=True, file_okay=False, path_type=Path),
    help="Directory to validate (recursive)",
)
@click.option(
    "--domain",
    type=click.Choice(["kafka", "kubernetes", "iac", "cicd", "gitops"], case_sensitive=False),
    help="Specific domain to validate",
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
    "--no-color",
    is_flag=True,
    help="Disable colored output",
)
@click.option(
    "--fail-on",
    type=click.Choice(["violation", "warning", "never"], case_sensitive=False),
    default="violation",
    help="Exit code behavior",
)
@click.option(
    "--policy-path",
    "-p",
    type=click.Path(exists=True, path_type=Path),
    default="./policies",
    help="Path to policy files directory",
)
def validate(file_path, directory, domain, output_format, no_color, fail_on, policy_path):
    """Validate configuration files against policies.

    Examples:

      \b
      # Validate a single file
      policy-agent validate --file topic.yaml

      \b
      # Validate a directory
      policy-agent validate --dir ./configs

      \b
      # Validate with specific domain
      policy-agent validate --file topic.yaml --domain kafka

      \b
      # Get JSON output
      policy-agent validate --file topic.yaml --format json

      \b
      # Validate directory with fail on warning
      policy-agent validate --dir ./configs --fail-on warning
    """
    # Validate input
    if not file_path and not directory:
        click.echo("Error: Either --file or --dir must be specified", err=True)
        sys.exit(1)

    if file_path and directory:
        click.echo("Error: Cannot specify both --file and --dir", err=True)
        sys.exit(1)

    try:
        # Initialize policy engine
        policy_engine = None
        if policy_path and policy_path.exists():
            click.echo(f"✓ Loading policies from: {policy_path}")
            policy_engine = PolicyEngine(str(policy_path))
        else:
            click.echo("⚠️  No policy path specified, using built-in policies only", err=True)

        # Setup validators
        validators = setup_validators(policy_engine=policy_engine)
        click.echo(f"✓ Registered validators: {', '.join(validators.keys())}\n")

        # Create formatter
        formatter = OutputFormatter(format_type=output_format, color=not no_color)

        # Validate file or directory
        if file_path:
            result = validate_file(file_path, domain, validators)
            output = formatter.format_result(result)
            click.echo(output)

            # Check fail condition
            check_fail_condition(result, fail_on)

        else:  # directory
            results = validate_directory(directory, domain, validators)
            output = formatter.format_batch_results(results)
            click.echo(output)

            # Check fail condition for batch
            for result in results:
                check_fail_condition(result, fail_on)

    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        sys.exit(1)


def validate_file(
    file_path: Path, domain: str = None, validators: dict = None
) -> ValidationResult:
    """Validate a single file.

    Args:
        file_path: Path to file
        domain: Specific domain or None for auto-detect
        validators: Dictionary of validators

    Returns:
        ValidationResult
    """
    # Read file
    with open(file_path) as f:
        content = f.read()

    # Parse content
    try:
        data = yaml.safe_load(content)
    except yaml.YAMLError as e:
        raise ValueError(f"Failed to parse YAML: {e}")

    # Determine validator
    if domain:
        validator = validators.get(domain)
        if not validator:
            raise ValueError(f"Unknown domain: {domain}")
    else:
        validator = get_validator_for_file(str(file_path), validators)
        if not validator:
            # Try to detect from content
            kind = data.get("kind", "").lower() if isinstance(data, dict) else ""
            if "kafkatopic" in kind:
                validator = validators.get("kafka")
            elif "deployment" in kind or "pod" in kind:
                validator = validators.get("kubernetes")
            elif "kustomization" in kind:
                validator = validators.get("gitops")
            elif "application" in kind:
                validator = validators.get("gitops")
            else:
                raise ValueError(f"Could not determine validator for {file_path}")

    # Validate
    click.echo(f"Validating: {file_path}")
    result = validator.validate(data, file_path=str(file_path))

    return result


def validate_directory(
    directory: Path, domain: str = None, validators: dict = None
) -> List[ValidationResult]:
    """Validate all files in a directory.

    Args:
        directory: Directory path
        domain: Specific domain or None
        validators: Dictionary of validators

    Returns:
        List of ValidationResults
    """
    results = []

    # Find all config files
    config_extensions = {".yaml", ".yml", ".json", ".tf", ".hcl"}
    config_files = []

    for ext in config_extensions:
        config_files.extend(directory.rglob(f"*{ext}"))

    if not config_files:
        raise ValueError(f"No configuration files found in {directory}")

    # Validate each file
    for file_path in sorted(config_files):
        try:
            result = validate_file(file_path, domain, validators)
            results.append(result)
        except Exception as e:
            click.echo(f"Warning: Failed to validate {file_path}: {e}", err=True)
            continue

    return results


def check_fail_condition(result: ValidationResult, fail_on: str):
    """Check fail condition and exit if needed.

    Args:
        result: Validation result
        fail_on: Fail condition (violation, warning, never)
    """
    if fail_on == "never":
        return

    if fail_on == "violation" and len(result.violations) > 0:
        sys.exit(1)

    if fail_on == "warning" and (len(result.violations) > 0 or len(result.warnings) > 0):
        sys.exit(1)
