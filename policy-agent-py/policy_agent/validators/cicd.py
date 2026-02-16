"""CI/CD pipeline validator for GitHub Actions, GitLab CI, etc."""

from dataclasses import dataclass
from datetime import datetime
from typing import Any, Dict, List, Optional

from policy_agent.validators.base import Validator
from policy_agent.policy.engine import PolicyEngine
from policy_agent.types.result import (
    ValidationResult,
    Violation,
    Warning,
    PolicyCheck,
    Remediation,
    Severity,
)


@dataclass
class CICDConfig:
    """CI/CD validation configuration."""

    require_security_scanning: bool = True
    require_testing: bool = True
    allowed_registries: List[str] = None
    max_job_timeout: int = 60  # minutes
    require_approval: bool = True

    def __post_init__(self):
        if self.allowed_registries is None:
            self.allowed_registries = ["docker.io", "ghcr.io", "gcr.io"]


class CICDValidator(Validator):
    """Validator for CI/CD pipeline configurations."""

    def __init__(self, policy_engine: Optional[PolicyEngine] = None, config: Optional[CICDConfig] = None):
        """Initialize CI/CD validator.

        Args:
            policy_engine: Policy engine for OPA evaluation
            config: CI/CD-specific configuration
        """
        self.policy_engine = policy_engine
        self.config = config or CICDConfig()

    def domain(self) -> str:
        """Return domain name."""
        return "cicd"

    def supported_types(self) -> List[str]:
        """Return supported CI/CD pipeline types."""
        return ["GitHubWorkflow", "GitLabCI", "JenkinsFile"]

    def validate(self, input_data: Dict[str, Any], file_path: Optional[str] = None) -> ValidationResult:
        """Validate CI/CD pipeline configuration.

        Args:
            input_data: Parsed pipeline configuration
            file_path: Optional source file path

        Returns:
            ValidationResult with violations and warnings
        """
        # Determine resource type
        resource_type = input_data.get("kind", "GitHubWorkflow")
        if resource_type not in self.supported_types():
            raise ValueError(f"Unsupported CI/CD resource type: {resource_type}")

        # Extract resource name
        resource_name = input_data.get("name", "pipeline")

        # Create result
        result = ValidationResult(
            status="passed",
            domain=self.domain(),
            resource=resource_name,
            resource_type=resource_type,
            file=file_path,
            timestamp=datetime.now(),
        )

        # Validate based on type
        if resource_type == "GitHubWorkflow":
            self._validate_github_workflow(input_data, result)
        elif resource_type == "GitLabCI":
            self._validate_gitlab_ci(input_data, result)
        else:
            result.warnings.append(
                Warning(
                    policy=f"{self.domain()}.{resource_type.lower()}",
                    severity=Severity.LOW,
                    message=f"Validation not yet implemented for {resource_type}",
                )
            )

        # Set overall status
        if result.violations:
            result.status = "failed"
        elif result.warnings:
            result.status = "warning"

        return result

    def _validate_github_workflow(self, config: Dict[str, Any], result: ValidationResult) -> None:
        """Validate GitHub Actions workflow."""
        # Check workflow name
        if "name" not in config:
            result.violations.append(
                Violation(
                    policy="cicd.github.workflows",
                    severity=Severity.MEDIUM,
                    message="Workflow must have a descriptive name",
                    field="name",
                )
            )

        # Check jobs
        jobs = config.get("jobs", {})
        if not jobs:
            result.violations.append(
                Violation(
                    policy="cicd.github.workflows",
                    severity=Severity.HIGH,
                    message="Workflow must define at least one job",
                    field="jobs",
                )
            )
            return

        has_test_job = False
        for job_name, job_data in jobs.items():
            if not isinstance(job_data, dict):
                continue

            self._validate_github_job(job_name, job_data, result)

            # Check for test jobs
            if "test" in job_name.lower():
                has_test_job = True

        # Require test job
        if self.config.require_testing and not has_test_job:
            result.violations.append(
                Violation(
                    policy="cicd.github.workflows",
                    severity=Severity.HIGH,
                    message="Workflow should include a test job to validate code quality",
                )
            )

        # Check security scanning
        if self.config.require_security_scanning and not self._has_security_scanning(jobs):
            result.warnings.append(
                Warning(
                    policy="cicd.github.security",
                    severity=Severity.MEDIUM,
                    message="Workflow should include security scanning (CodeQL, dependency scanning, or SAST)",
                )
            )

    def _validate_github_job(self, job_name: str, job: Dict[str, Any], result: ValidationResult) -> None:
        """Validate individual GitHub Actions job."""
        # Check timeout
        timeout = job.get("timeout-minutes")
        if timeout:
            if timeout > self.config.max_job_timeout:
                result.violations.append(
                    Violation(
                        policy="cicd.github.workflows",
                        severity=Severity.MEDIUM,
                        message=f"Job '{job_name}' timeout of {timeout} minutes exceeds maximum {self.config.max_job_timeout} minutes",
                        field=f"jobs.{job_name}.timeout-minutes",
                        current_value=timeout,
                        expected_value=self.config.max_job_timeout,
                    )
                )
        else:
            result.warnings.append(
                Warning(
                    policy="cicd.github.workflows",
                    severity=Severity.LOW,
                    message=f"Job '{job_name}' should set timeout-minutes to prevent hanging builds",
                    field=f"jobs.{job_name}.timeout-minutes",
                )
            )

        # Check steps for security issues
        steps = job.get("steps", [])
        if steps:
            self._validate_github_steps(job_name, steps, result)

        # Check permissions
        permissions = job.get("permissions", {})
        if permissions:
            self._validate_permissions(job_name, permissions, result)

    def _validate_github_steps(self, job_name: str, steps: List[Dict[str, Any]], result: ValidationResult) -> None:
        """Validate workflow steps."""
        for i, step in enumerate(steps):
            if not isinstance(step, dict):
                continue

            # Check for unpinned actions
            uses = step.get("uses", "")
            if uses:
                if not self._is_action_pinned(uses) and not self._is_official_action(uses):
                    result.violations.append(
                        Violation(
                            policy="cicd.github.security",
                            severity=Severity.HIGH,
                            message=f"Job '{job_name}' uses unpinned third-party action '{uses}' - pin to commit SHA for security",
                            field=f"jobs.{job_name}.steps[{i}].uses",
                            remediation=Remediation(
                                auto_fixable=False,
                                suggestion="Pin actions to commit SHA (e.g., action@a1b2c3d4...)",
                            ),
                        )
                    )

            # Check for secret exposure
            run = step.get("run", "")
            if run and self._may_expose_secrets(run):
                result.violations.append(
                    Violation(
                        policy="cicd.github.security",
                        severity=Severity.CRITICAL,
                        message=f"Job '{job_name}' may expose secrets in logs - avoid echoing secret values",
                        field=f"jobs.{job_name}.steps[{i}].run",
                    )
                )

    def _validate_permissions(self, job_name: str, permissions: Dict[str, str], result: ValidationResult) -> None:
        """Validate workflow permissions."""
        has_write = any(value == "write" for value in permissions.values())

        if has_write:
            result.warnings.append(
                Warning(
                    policy="cicd.github.security",
                    severity=Severity.MEDIUM,
                    message=f"Job '{job_name}' has write permissions - use minimal required permissions",
                )
            )

    def _validate_gitlab_ci(self, config: Dict[str, Any], result: ValidationResult) -> None:
        """Validate GitLab CI configuration."""
        has_test_stage = False

        # Iterate through jobs
        for job_name, job_data in config.items():
            # Skip special keys
            if self._is_gitlab_special_key(job_name):
                continue

            if not isinstance(job_data, dict):
                continue

            self._validate_gitlab_job(job_name, job_data, result)

            # Check for test stage
            stage = job_data.get("stage", "")
            if "test" in stage.lower():
                has_test_stage = True

        # Require test stage
        if self.config.require_testing and not has_test_stage:
            result.violations.append(
                Violation(
                    policy="cicd.gitlab.security",
                    severity=Severity.HIGH,
                    message="Pipeline should include testing stage to validate code quality",
                )
            )

    def _validate_gitlab_job(self, job_name: str, job: Dict[str, Any], result: ValidationResult) -> None:
        """Validate individual GitLab CI job."""
        # Check for privileged Docker
        services = job.get("services", [])
        for service in services:
            if isinstance(service, dict):
                command = service.get("command", [])
                if any("--privileged" in str(cmd) for cmd in command):
                    result.violations.append(
                        Violation(
                            policy="cicd.gitlab.security",
                            severity=Severity.CRITICAL,
                            message=f"Job '{job_name}' uses privileged Docker mode - security risk",
                            field=f"{job_name}.services",
                        )
                    )

        # Check image tag
        image = job.get("image", "")
        if isinstance(image, str) and image.endswith(":latest"):
            result.violations.append(
                Violation(
                    policy="cicd.gitlab.security",
                    severity=Severity.MEDIUM,
                    message=f"Job '{job_name}' uses ':latest' Docker image tag - pin to specific version",
                    field=f"{job_name}.image",
                    current_value=image,
                    expected_value="specific version tag",
                )
            )

        # Check for production deployment without manual approval
        environment = job.get("environment", "")
        if isinstance(environment, str) and "prod" in environment.lower():
            when = job.get("when", "")
            if when != "manual":
                result.violations.append(
                    Violation(
                        policy="cicd.gitlab.security",
                        severity=Severity.HIGH,
                        message=f"Production deployment job '{job_name}' should require manual approval",
                        field=f"{job_name}.when",
                    )
                )

        # Check artifacts expiration
        artifacts = job.get("artifacts", {})
        if isinstance(artifacts, dict) and artifacts.get("paths"):
            if "expire_in" not in artifacts:
                result.warnings.append(
                    Warning(
                        policy="cicd.gitlab.security",
                        severity=Severity.LOW,
                        message=f"Job '{job_name}' artifacts should have expire_in set to manage storage",
                        field=f"{job_name}.artifacts.expire_in",
                    )
                )

    # Helper methods

    def _has_security_scanning(self, jobs: Dict[str, Any]) -> bool:
        """Check if workflow has security scanning."""
        for job in jobs.values():
            if not isinstance(job, dict):
                continue

            steps = job.get("steps", [])
            for step in steps:
                if not isinstance(step, dict):
                    continue

                uses = step.get("uses", "")
                security_actions = ["github/codeql-action", "aquasecurity/trivy", "snyk/actions"]

                if any(action in uses for action in security_actions):
                    return True

        return False

    def _is_action_pinned(self, action: str) -> bool:
        """Check if action is pinned to commit SHA."""
        parts = action.split("@")
        if len(parts) != 2:
            return False

        ref = parts[1]
        # SHA is 40 hex characters
        if len(ref) == 40:
            try:
                int(ref, 16)  # Check if hex
                return True
            except ValueError:
                return False

        return False

    def _is_official_action(self, action: str) -> bool:
        """Check if action is from official sources."""
        official_prefixes = ["actions/", "github/"]
        return any(action.startswith(prefix) for prefix in official_prefixes)

    def _may_expose_secrets(self, script: str) -> bool:
        """Check if script may expose secrets."""
        if "echo" in script and "secrets." in script:
            return True
        if "echo" in script and "${{" in script:
            return True
        return False

    def _is_gitlab_special_key(self, key: str) -> bool:
        """Check if key is GitLab CI special key."""
        special_keys = ["stages", "variables", "default", "workflow", "include"]
        return key in special_keys
