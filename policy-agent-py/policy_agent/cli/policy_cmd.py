"""Policy management commands."""

from pathlib import Path

import click

from policy_agent.cli.registry import setup_validators


@click.group()
def policy():
    """Policy management commands.

    Manage OPA/Rego policies: list, test, validate.
    """
    pass


@policy.command(name="list")
@click.option(
    "--policy-path",
    "-p",
    type=click.Path(exists=True, path_type=Path),
    default="./policies",
    help="Path to policy files directory",
)
def list_policies(policy_path):
    """List available policies.

    Shows all policies available in the policy directory organized by domain.
    """
    try:
        if not policy_path.exists():
            click.echo(f"Error: Policy path not found: {policy_path}", err=True)
            return

        click.echo(f"Policy path: {policy_path}\n")

        # List policy files by domain
        domains = {}
        for policy_file in policy_path.rglob("*.rego"):
            # Extract domain from path (e.g., policies/kafka/topics/replication.rego -> kafka)
            parts = policy_file.relative_to(policy_path).parts
            if len(parts) > 0:
                domain = parts[0]
                if domain not in domains:
                    domains[domain] = []
                domains[domain].append(policy_file.relative_to(policy_path))

        click.echo("Available policies by domain:\n")
        for domain, files in sorted(domains.items()):
            click.echo(f"📁 {domain}:")
            for file in sorted(files):
                click.echo(f"   • {file}")
            click.echo()

        # Show validators
        validators = setup_validators()
        click.echo("Registered validators:")
        for name in sorted(validators.keys()):
            validator = validators[name]
            types = ", ".join(validator.supported_types())
            click.echo(f"   • {name}: {types}")

    except Exception as e:
        click.echo(f"Error: {e}", err=True)


@policy.command(name="test")
def test_policies():
    """Test policies.

    Run tests for OPA/Rego policies to ensure they work correctly.
    """
    click.echo("Policy testing - Coming soon!")
    click.echo("Will run OPA test suites for all policy files.")


@policy.command(name="validate")
@click.option(
    "--policy-path",
    "-p",
    type=click.Path(exists=True, path_type=Path),
    default="./policies",
    help="Path to policy files directory",
)
def validate_policies(policy_path):
    """Validate policy syntax.

    Check that all policy files have valid Rego syntax.
    """
    click.echo("Policy validation - Coming soon!")
    click.echo(f"Will validate syntax of policies in: {policy_path}")
