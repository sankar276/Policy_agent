"""GitOps validator for Flux CD and ArgoCD."""

from dataclasses import dataclass
from datetime import datetime
from typing import Any, Dict, List, Optional
import re

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
class GitOpsConfig:
    """GitOps validation configuration."""

    require_prune: bool = True
    require_health_checks: bool = True
    require_gpg_verification: bool = False
    max_interval_seconds: int = 600
    require_rollback: bool = True
    allow_auto_sync: bool = True


class GitOpsValidator(Validator):
    """Validator for GitOps configurations (Flux CD and ArgoCD)."""

    def __init__(self, policy_engine: Optional[PolicyEngine] = None, config: Optional[GitOpsConfig] = None):
        """Initialize GitOps validator.

        Args:
            policy_engine: Policy engine for OPA evaluation
            config: GitOps-specific configuration
        """
        self.policy_engine = policy_engine
        self.config = config or GitOpsConfig()

    def domain(self) -> str:
        """Return domain name."""
        return "gitops"

    def supported_types(self) -> List[str]:
        """Return supported GitOps resource types."""
        return [
            # Flux CD
            "Kustomization",
            "GitRepository",
            "HelmRelease",
            # ArgoCD
            "Application",
            "AppProject",
        ]

    def validate(self, input_data: Dict[str, Any], file_path: Optional[str] = None) -> ValidationResult:
        """Validate GitOps configuration.

        Args:
            input_data: Parsed GitOps configuration
            file_path: Optional source file path

        Returns:
            ValidationResult with violations and warnings
        """
        # Determine resource type
        resource_type = input_data.get("kind", "Unknown")
        if resource_type not in self.supported_types():
            raise ValueError(f"Unsupported GitOps resource type: {resource_type}")

        # Extract resource name
        resource_name = input_data.get("metadata", {}).get("name", "unknown")

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
        if resource_type == "Kustomization":
            self._validate_flux_kustomization(input_data, result)
        elif resource_type == "GitRepository":
            self._validate_flux_gitrepository(input_data, result)
        elif resource_type == "HelmRelease":
            self._validate_flux_helmrelease(input_data, result)
        elif resource_type == "Application":
            self._validate_argocd_application(input_data, result)
        elif resource_type == "AppProject":
            self._validate_argocd_appproject(input_data, result)
        else:
            result.warnings.append(
                Warning(
                    policy=f"{self.domain()}.{resource_type.lower()}",
                    severity=Severity.MEDIUM,
                    message=f"Validation not yet implemented for {resource_type}",
                )
            )

        # Set overall status
        if result.violations:
            result.status = "failed"
        elif result.warnings:
            result.status = "warning"

        return result

    def _validate_flux_kustomization(self, config: Dict[str, Any], result: ValidationResult) -> None:
        """Validate Flux Kustomization resources."""
        spec = config.get("spec", {})
        if not spec:
            result.violations.append(
                Violation(
                    policy="gitops.flux.kustomization",
                    severity=Severity.HIGH,
                    message="Kustomization must have spec",
                    field="spec",
                )
            )
            return

        # Check prune enabled
        if self.config.require_prune and not spec.get("prune", False):
            result.violations.append(
                Violation(
                    policy="gitops.flux.kustomization",
                    severity=Severity.HIGH,
                    message="Kustomization should enable prune to clean up deleted resources",
                    field="spec.prune",
                )
            )

        # Check source reference
        if "sourceRef" not in spec:
            result.violations.append(
                Violation(
                    policy="gitops.flux.kustomization",
                    severity=Severity.HIGH,
                    message="Kustomization must specify sourceRef",
                    field="spec.sourceRef",
                )
            )

        # Check interval
        interval = spec.get("interval")
        if interval:
            if not self._is_valid_interval(interval):
                result.warnings.append(
                    Warning(
                        policy="gitops.flux.kustomization",
                        severity=Severity.LOW,
                        message=f"Kustomization interval '{interval}' may be too short or too long",
                        field="spec.interval",
                    )
                )
        else:
            result.violations.append(
                Violation(
                    policy="gitops.flux.kustomization",
                    severity=Severity.MEDIUM,
                    message="Kustomization must specify reconciliation interval",
                    field="spec.interval",
                )
            )

        # Check force sync (dangerous)
        if spec.get("force", False):
            result.warnings.append(
                Warning(
                    policy="gitops.flux.kustomization",
                    severity=Severity.HIGH,
                    message="Kustomization has force enabled - can cause unexpected resource replacements",
                    field="spec.force",
                )
            )

        # Check health checks for production
        if self.config.require_health_checks and self._is_production(config):
            if "healthChecks" not in spec:
                result.warnings.append(
                    Warning(
                        policy="gitops.flux.kustomization",
                        severity=Severity.MEDIUM,
                        message="Production Kustomization should configure health checks",
                        field="spec.healthChecks",
                    )
                )

        # Check service account
        if "serviceAccountName" not in spec:
            result.warnings.append(
                Warning(
                    policy="gitops.flux.kustomization",
                    severity=Severity.LOW,
                    message="Kustomization should specify serviceAccountName for RBAC",
                    field="spec.serviceAccountName",
                )
            )

        # Check path traversal
        path = spec.get("path", "")
        if ".." in path:
            result.violations.append(
                Violation(
                    policy="gitops.flux.kustomization",
                    severity=Severity.CRITICAL,
                    message="Kustomization path contains '..' - potential path traversal vulnerability",
                    field="spec.path",
                )
            )

    def _validate_flux_gitrepository(self, config: Dict[str, Any], result: ValidationResult) -> None:
        """Validate Flux GitRepository sources."""
        spec = config.get("spec", {})
        if not spec:
            result.violations.append(
                Violation(
                    policy="gitops.flux.gitrepository",
                    severity=Severity.HIGH,
                    message="GitRepository must have spec",
                    field="spec",
                )
            )
            return

        # Check URL protocol
        url = spec.get("url", "")
        if url:
            if url.startswith("http://"):
                result.violations.append(
                    Violation(
                        policy="gitops.flux.gitrepository",
                        severity=Severity.HIGH,
                        message=f"GitRepository URL '{url}' uses insecure HTTP - use HTTPS or SSH",
                        field="spec.url",
                    )
                )
        else:
            result.violations.append(
                Violation(
                    policy="gitops.flux.gitrepository",
                    severity=Severity.HIGH,
                    message="GitRepository must specify URL",
                    field="spec.url",
                )
            )

        # Check GPG verification for production
        if self.config.require_gpg_verification and self._is_production(config):
            verify = spec.get("verify", {})
            if not verify or not verify.get("mode"):
                result.warnings.append(
                    Warning(
                        policy="gitops.flux.gitrepository",
                        severity=Severity.MEDIUM,
                        message="Production GitRepository should enable GPG verification",
                        field="spec.verify",
                    )
                )

        # Check interval
        interval = spec.get("interval")
        if interval:
            if not self._is_valid_interval(interval):
                result.warnings.append(
                    Warning(
                        policy="gitops.flux.gitrepository",
                        severity=Severity.LOW,
                        message=f"GitRepository interval '{interval}' may be too short or too long",
                        field="spec.interval",
                    )
                )
        else:
            result.violations.append(
                Violation(
                    policy="gitops.flux.gitrepository",
                    severity=Severity.MEDIUM,
                    message="GitRepository must specify reconciliation interval",
                    field="spec.interval",
                )
            )

        # Check ref specified
        ref = spec.get("ref", {})
        if not ref:
            result.warnings.append(
                Warning(
                    policy="gitops.flux.gitrepository",
                    severity=Severity.MEDIUM,
                    message="GitRepository should specify explicit ref (branch, tag, or commit)",
                    field="spec.ref",
                )
            )

    def _validate_flux_helmrelease(self, config: Dict[str, Any], result: ValidationResult) -> None:
        """Validate Flux HelmRelease resources."""
        spec = config.get("spec", {})
        if not spec:
            result.violations.append(
                Violation(
                    policy="gitops.flux.helmrelease",
                    severity=Severity.HIGH,
                    message="HelmRelease must have spec",
                    field="spec",
                )
            )
            return

        # Check chart reference
        if "chart" not in spec:
            result.violations.append(
                Violation(
                    policy="gitops.flux.helmrelease",
                    severity=Severity.HIGH,
                    message="HelmRelease must specify chart reference",
                    field="spec.chart",
                )
            )

        # Check interval
        if "interval" not in spec:
            result.violations.append(
                Violation(
                    policy="gitops.flux.helmrelease",
                    severity=Severity.MEDIUM,
                    message="HelmRelease must specify reconciliation interval",
                    field="spec.interval",
                )
            )

        # Check rollback for production
        if self.config.require_rollback and self._is_production(config):
            rollback = spec.get("rollback", {})
            if not rollback:
                result.violations.append(
                    Violation(
                        policy="gitops.flux.helmrelease",
                        severity=Severity.HIGH,
                        message="Production HelmRelease should configure automatic rollback",
                        field="spec.rollback",
                    )
                )
            elif rollback.get("enable") is False:
                result.violations.append(
                    Violation(
                        policy="gitops.flux.helmrelease",
                        severity=Severity.HIGH,
                        message="Production HelmRelease has rollback disabled",
                        field="spec.rollback.enable",
                    )
                )

        # Check timeout
        if "timeout" not in spec:
            result.warnings.append(
                Warning(
                    policy="gitops.flux.helmrelease",
                    severity=Severity.MEDIUM,
                    message="HelmRelease should set timeout to prevent hanging installations",
                    field="spec.timeout",
                )
            )

        # Check upgrade configuration
        upgrade = spec.get("upgrade", {})
        if upgrade and upgrade.get("force", False):
            result.warnings.append(
                Warning(
                    policy="gitops.flux.helmrelease",
                    severity=Severity.MEDIUM,
                    message="HelmRelease has force upgrade enabled - can cause unexpected resource replacements",
                    field="spec.upgrade.force",
                )
            )

        # Check chart version pinned
        chart = spec.get("chart", {})
        if chart:
            chart_spec = chart.get("spec", {})
            if chart_spec:
                version = chart_spec.get("version")
                if version:
                    if self._is_mutable_version(version):
                        result.warnings.append(
                            Warning(
                                policy="gitops.flux.helmrelease",
                                severity=Severity.MEDIUM,
                                message=f"HelmRelease chart version '{version}' uses range - pin to specific version",
                                field="spec.chart.spec.version",
                            )
                        )
                else:
                    result.violations.append(
                        Violation(
                            policy="gitops.flux.helmrelease",
                            severity=Severity.MEDIUM,
                            message="HelmRelease should pin chart version for reproducible deployments",
                            field="spec.chart.spec.version",
                        )
                    )

    def _validate_argocd_application(self, config: Dict[str, Any], result: ValidationResult) -> None:
        """Validate ArgoCD Application resources."""
        spec = config.get("spec", {})
        if not spec:
            result.violations.append(
                Violation(
                    policy="gitops.argocd.application",
                    severity=Severity.HIGH,
                    message="Application must have spec",
                    field="spec",
                )
            )
            return

        # Check source
        source = spec.get("source", {})
        if source:
            # Check repo URL protocol
            repo_url = source.get("repoURL", "")
            if repo_url.startswith("http://"):
                result.violations.append(
                    Violation(
                        policy="gitops.argocd.application",
                        severity=Severity.HIGH,
                        message=f"Application source '{repo_url}' uses insecure HTTP - use HTTPS or SSH",
                        field="spec.source.repoURL",
                    )
                )

            # Check target revision
            target_revision = source.get("targetRevision")
            if not target_revision:
                result.warnings.append(
                    Warning(
                        policy="gitops.argocd.application",
                        severity=Severity.MEDIUM,
                        message="Application should specify targetRevision for predictability",
                        field="spec.source.targetRevision",
                    )
                )
            elif target_revision == "HEAD":
                result.warnings.append(
                    Warning(
                        policy="gitops.argocd.application",
                        severity=Severity.MEDIUM,
                        message="Application targetRevision 'HEAD' is unpredictable - use specific branch or tag",
                        field="spec.source.targetRevision",
                    )
                )
        else:
            result.violations.append(
                Violation(
                    policy="gitops.argocd.application",
                    severity=Severity.HIGH,
                    message="Application must specify source repository",
                    field="spec.source",
                )
            )

        # Check destination
        destination = spec.get("destination", {})
        if destination:
            namespace = destination.get("namespace")
            if not namespace:
                result.violations.append(
                    Violation(
                        policy="gitops.argocd.application",
                        severity=Severity.MEDIUM,
                        message="Application destination must specify namespace",
                        field="spec.destination.namespace",
                    )
                )
        else:
            result.violations.append(
                Violation(
                    policy="gitops.argocd.application",
                    severity=Severity.HIGH,
                    message="Application must specify destination cluster and namespace",
                    field="spec.destination",
                )
            )

        # Check project assignment
        project = spec.get("project", "")
        if project == "default":
            result.violations.append(
                Violation(
                    policy="gitops.argocd.application",
                    severity=Severity.MEDIUM,
                    message="Application should use a dedicated Project (not 'default') for proper isolation",
                    field="spec.project",
                )
            )

        # Check sync policy
        sync_policy = spec.get("syncPolicy", {})
        if sync_policy:
            # Check auto-sync with prune
            automated = sync_policy.get("automated", {})
            if automated:
                if self._is_production(config):
                    result.warnings.append(
                        Warning(
                            policy="gitops.argocd.application",
                            severity=Severity.MEDIUM,
                            message="Production Application has automated sync - consider requiring manual approval",
                            field="spec.syncPolicy.automated",
                        )
                    )

                # Check prune enabled
                if self.config.require_prune and not automated.get("prune", False):
                    result.violations.append(
                        Violation(
                            policy="gitops.argocd.application",
                            severity=Severity.HIGH,
                            message="Application with automated sync should enable prune to clean up deleted resources",
                            field="spec.syncPolicy.automated.prune",
                        )
                    )

            # Check sync options
            sync_options = sync_policy.get("syncOptions", [])
            for opt in sync_options:
                if opt == "Replace=true":
                    result.violations.append(
                        Violation(
                            policy="gitops.argocd.application",
                            severity=Severity.HIGH,
                            message="Application uses Replace=true sync option - can cause data loss",
                            field="spec.syncPolicy.syncOptions",
                        )
                    )

    def _validate_argocd_appproject(self, config: Dict[str, Any], result: ValidationResult) -> None:
        """Validate ArgoCD AppProject resources."""
        spec = config.get("spec", {})
        if not spec:
            result.violations.append(
                Violation(
                    policy="gitops.argocd.appproject",
                    severity=Severity.HIGH,
                    message="AppProject must have spec",
                    field="spec",
                )
            )
            return

        # Check source repositories
        source_repos = spec.get("sourceRepos", [])
        if not source_repos:
            result.violations.append(
                Violation(
                    policy="gitops.argocd.appproject",
                    severity=Severity.HIGH,
                    message="AppProject must specify allowed source repositories",
                    field="spec.sourceRepos",
                )
            )
        else:
            # Check for wildcard
            if "*" in source_repos:
                result.violations.append(
                    Violation(
                        policy="gitops.argocd.appproject",
                        severity=Severity.HIGH,
                        message="AppProject allows all repositories (*) - specify explicit repositories for security",
                        field="spec.sourceRepos",
                    )
                )

        # Check destinations
        destinations = spec.get("destinations", [])
        if not destinations:
            result.violations.append(
                Violation(
                    policy="gitops.argocd.appproject",
                    severity=Severity.HIGH,
                    message="AppProject must specify allowed destinations",
                    field="spec.destinations",
                )
            )
        else:
            # Check for wildcard namespaces
            for dest in destinations:
                if isinstance(dest, dict):
                    namespace = dest.get("namespace")
                    server = dest.get("server")
                    if namespace == "*" and server != "https://kubernetes.default.svc":
                        result.violations.append(
                            Violation(
                                policy="gitops.argocd.appproject",
                                severity=Severity.HIGH,
                                message="AppProject allows deployment to all namespaces (*) - specify explicit namespaces",
                                field="spec.destinations",
                            )
                        )

        # Check RBAC roles for production
        if self._is_production(config):
            roles = spec.get("roles", [])
            if not roles:
                result.violations.append(
                    Violation(
                        policy="gitops.argocd.appproject",
                        severity=Severity.MEDIUM,
                        message="Production AppProject should define RBAC roles for access control",
                        field="spec.roles",
                    )
                )

        # Check cluster resource whitelist
        cluster_resource_whitelist = spec.get("clusterResourceWhitelist", [])
        for resource in cluster_resource_whitelist:
            if isinstance(resource, dict):
                group = resource.get("group")
                kind = resource.get("kind")
                if group == "*" and kind == "*":
                    result.violations.append(
                        Violation(
                            policy="gitops.argocd.appproject",
                            severity=Severity.HIGH,
                            message="AppProject allows all cluster resources (*/*) - specify explicit resources",
                            field="spec.clusterResourceWhitelist",
                        )
                    )

        # Check orphaned resources policy
        if "orphanedResources" not in spec:
            result.warnings.append(
                Warning(
                    policy="gitops.argocd.appproject",
                    severity=Severity.MEDIUM,
                    message="AppProject should configure orphanedResources policy to detect orphaned resources",
                    field="spec.orphanedResources",
                )
            )

    # Helper methods

    def _is_production(self, config: Dict[str, Any]) -> bool:
        """Check if resource is for production environment."""
        metadata = config.get("metadata", {})

        # Check name
        name = metadata.get("name", "").lower()
        if any(ind in name for ind in ["prod", "production", "live"]):
            return True

        # Check namespace
        namespace = metadata.get("namespace", "").lower()
        if any(ind in namespace for ind in ["prod", "production", "live"]):
            return True

        # Check spec for ArgoCD Application
        spec = config.get("spec", {})
        destination = spec.get("destination", {})
        dest_namespace = destination.get("namespace", "").lower()
        if any(ind in dest_namespace for ind in ["prod", "production", "live"]):
            return True

        # Check targetNamespace for HelmRelease
        target_namespace = spec.get("targetNamespace", "").lower()
        if any(ind in target_namespace for ind in ["prod", "production", "live"]):
            return True

        return False

    def _is_valid_interval(self, interval: str) -> bool:
        """Check if interval is within valid range."""
        # Parse interval like "1m", "5m", "10m", "1h"
        match = re.match(r"^(\d+)([smh])$", interval)
        if not match:
            return True  # Unknown format, let it pass

        value = int(match.group(1))
        unit = match.group(2)

        # Convert to seconds
        if unit == "s":
            seconds = value
        elif unit == "m":
            seconds = value * 60
        elif unit == "h":
            seconds = value * 3600
        else:
            return True

        return 60 <= seconds <= self.config.max_interval_seconds

    def _is_mutable_version(self, version: str) -> bool:
        """Check if version uses mutable patterns."""
        mutable_patterns = ["x", "*", "~", "^"]
        return any(pattern in version for pattern in mutable_patterns)
