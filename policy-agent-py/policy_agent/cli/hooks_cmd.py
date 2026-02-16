"""Git hooks management commands."""

import os
import shutil
import subprocess
import sys
from pathlib import Path

import click


@click.group()
def hooks():
    """Manage Git hooks.

    Install, uninstall, or list Git hooks for automatic policy validation.
    """
    pass


@hooks.command(name="install")
@click.option(
    "--pre-commit",
    is_flag=True,
    help="Install pre-commit hook",
)
@click.option(
    "--pre-push",
    is_flag=True,
    help="Install pre-push hook",
)
@click.option(
    "--all",
    "install_all",
    is_flag=True,
    help="Install all hooks",
)
def install_hooks(pre_commit, pre_push, install_all):
    """Install Git hooks.

    Installs Git hooks that automatically validate configurations
    when committing or pushing.

    Examples:

      \b
      # Install all hooks
      policy-agent hooks install --all

      \b
      # Install only pre-commit
      policy-agent hooks install --pre-commit
    """
    try:
        # Find git directory
        git_dir = find_git_dir()
        if not git_dir:
            click.echo("Error: Not a git repository", err=True)
            sys.exit(1)

        hooks_dir = git_dir / "hooks"
        hooks_dir.mkdir(exist_ok=True)

        # Find policy-agent executable
        policy_agent_path = find_policy_agent()
        if not policy_agent_path:
            click.echo("Error: policy-agent not found in PATH", err=True)
            sys.exit(1)

        # Determine which hooks to install
        if not pre_commit and not pre_push and not install_all:
            install_all = True  # Default to all

        hooks_to_install = []
        if install_all or pre_commit:
            hooks_to_install.append("pre-commit")
        if install_all or pre_push:
            hooks_to_install.append("pre-push")

        # Install each hook
        for hook_name in hooks_to_install:
            install_hook(hooks_dir, hook_name, policy_agent_path)
            click.echo(f"✅ Installed {hook_name} hook")

        click.echo("\n🎉 Git hooks installed successfully!")
        click.echo("Configuration files will now be validated automatically.")

    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        sys.exit(1)


@hooks.command(name="uninstall")
def uninstall_hooks():
    """Uninstall Git hooks.

    Removes all policy-agent Git hooks.
    """
    try:
        # Find git directory
        git_dir = find_git_dir()
        if not git_dir:
            click.echo("Error: Not a git repository", err=True)
            sys.exit(1)

        hooks_dir = git_dir / "hooks"

        # Remove hooks
        for hook_name in ["pre-commit", "pre-push"]:
            hook_path = hooks_dir / hook_name

            if not hook_path.exists():
                continue

            # Check if it's a policy-agent hook
            content = hook_path.read_text()
            if "policy-agent" not in content:
                click.echo(f"⚠️  Skipping {hook_name} (not installed by policy-agent)")
                continue

            hook_path.unlink()
            click.echo(f"✅ Removed {hook_name} hook")

        click.echo("\n🎉 Git hooks uninstalled successfully!")

    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        sys.exit(1)


@hooks.command(name="list")
def list_hooks():
    """List installed Git hooks.

    Shows which policy-agent hooks are currently installed.
    """
    try:
        # Find git directory
        git_dir = find_git_dir()
        if not git_dir:
            click.echo("Error: Not a git repository", err=True)
            sys.exit(1)

        hooks_dir = git_dir / "hooks"

        click.echo("Git hooks status:\n")

        for hook_name in ["pre-commit", "pre-push", "commit-msg"]:
            hook_path = hooks_dir / hook_name

            if not hook_path.exists():
                click.echo(f"  {hook_name}: ❌ Not installed")
                continue

            # Check if it's a policy-agent hook
            content = hook_path.read_text()
            if "policy-agent" in content:
                click.echo(f"  {hook_name}: ✅ Installed")
            else:
                click.echo(f"  {hook_name}: ⚠️  Installed (not by policy-agent)")

    except Exception as e:
        click.echo(f"Error: {e}", err=True)
        sys.exit(1)


# Helper functions

def find_git_dir() -> Path:
    """Find the .git directory."""
    current = Path.cwd()

    while True:
        git_dir = current / ".git"

        if git_dir.exists():
            if git_dir.is_dir():
                return git_dir

            # .git is a file (submodule or worktree)
            content = git_dir.read_text()
            for line in content.split("\n"):
                if line.startswith("gitdir:"):
                    path = line.split(":", 1)[1].strip()
                    if not Path(path).is_absolute():
                        path = current / path
                    return Path(path)

        # Move up one directory
        parent = current.parent
        if parent == current:
            # Reached root
            return None

        current = parent


def find_policy_agent() -> str:
    """Find the policy-agent executable."""
    # Check if running from python -m
    if sys.argv[0].endswith("__main__.py"):
        return f"{sys.executable} -m policy_agent.cli.main"

    # Check PATH
    result = shutil.which("policy-agent")
    if result:
        return result

    # Check common installation paths
    common_paths = [
        Path.home() / ".local" / "bin" / "policy-agent",
        Path("/usr/local/bin/policy-agent"),
        Path("/usr/bin/policy-agent"),
    ]

    for path in common_paths:
        if path.exists():
            return str(path)

    return None


def install_hook(hooks_dir: Path, hook_name: str, policy_agent_path: str):
    """Install a specific Git hook."""
    hook_path = hooks_dir / hook_name

    # Check if hook already exists
    if hook_path.exists():
        content = hook_path.read_text()
        if "policy-agent" in content:
            click.echo(f"⚠️  {hook_name} hook already installed, updating...")
        else:
            raise ValueError(
                f"{hook_name} hook already exists (not installed by policy-agent). "
                "Remove it manually or use --force"
            )

    # Generate hook script
    script = generate_hook_script(hook_name, policy_agent_path)

    # Write hook file
    hook_path.write_text(script)
    hook_path.chmod(0o755)


def generate_hook_script(hook_name: str, policy_agent_path: str) -> str:
    """Generate hook script content."""
    if hook_name == "pre-commit":
        return f"""#!/bin/sh
# policy-agent pre-commit hook
# Auto-generated - do not edit manually

echo "🔍 Running policy validation..."

# Get list of staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)

if [ -z "$STAGED_FILES" ]; then
    echo "No files to validate"
    exit 0
fi

# Validate each staged file
EXIT_CODE=0
for FILE in $STAGED_FILES; do
    # Skip deleted files
    if [ ! -f "$FILE" ]; then
        continue
    fi

    # Validate config files
    case "$FILE" in
        *.yaml|*.yml|*.json|*.tf|*.hcl)
            echo "Validating: $FILE"
            {policy_agent_path} validate --file "$FILE"
            if [ $? -ne 0 ]; then
                EXIT_CODE=1
            fi
            ;;
    esac
done

if [ $EXIT_CODE -ne 0 ]; then
    echo "❌ Policy validation failed. Fix violations or use --no-verify to skip."
    exit 1
fi

echo "✅ All checks passed!"
exit 0
"""

    elif hook_name == "pre-push":
        return f"""#!/bin/sh
# policy-agent pre-push hook
# Auto-generated - do not edit manually

echo "🔍 Running policy validation before push..."

# Validate all tracked configuration files
CONFIG_FILES=$(git ls-files | grep -E '\\.(yaml|yml|json|tf|hcl)$')

if [ -z "$CONFIG_FILES" ]; then
    echo "No configuration files to validate"
    exit 0
fi

EXIT_CODE=0
for FILE in $CONFIG_FILES; do
    if [ -f "$FILE" ]; then
        echo "Validating: $FILE"
        {policy_agent_path} validate --file "$FILE"
        if [ $? -ne 0 ]; then
            EXIT_CODE=1
        fi
    fi
done

if [ $EXIT_CODE -ne 0 ]; then
    echo "❌ Policy validation failed. Fix violations or use --no-verify to skip."
    exit 1
fi

echo "✅ All checks passed!"
exit 0
"""

    return ""
