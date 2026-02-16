"""Validator registry setup."""

from typing import Dict, List, Optional

from policy_agent.validators.base import Validator
from policy_agent.validators.kafka import KafkaValidator
from policy_agent.validators.kubernetes import KubernetesValidator
from policy_agent.validators.iac import IaCValidator
from policy_agent.validators.cicd import CICDValidator
from policy_agent.validators.gitops import GitOpsValidator
from policy_agent.policy.engine import PolicyEngine


def setup_validators(
    policy_engine: Optional[PolicyEngine] = None,
    enabled_domains: Optional[List[str]] = None,
) -> Dict[str, Validator]:
    """Setup and register all domain validators.

    Args:
        policy_engine: Policy engine for OPA evaluation
        enabled_domains: List of enabled domains (None = all)

    Returns:
        Dictionary of validators by domain
    """
    validators = {}

    if enabled_domains is None:
        enabled_domains = ["kafka", "kubernetes", "iac", "cicd", "gitops"]

    # Register Kafka validator
    if "kafka" in enabled_domains:
        validators["kafka"] = KafkaValidator(policy_engine=policy_engine)

    # Register Kubernetes validator
    if "kubernetes" in enabled_domains:
        validators["kubernetes"] = KubernetesValidator(policy_engine=policy_engine)

    # Register IaC validator
    if "iac" in enabled_domains:
        validators["iac"] = IaCValidator(policy_engine=policy_engine)

    # Register CI/CD validator
    if "cicd" in enabled_domains:
        validators["cicd"] = CICDValidator(policy_engine=policy_engine)

    # Register GitOps validator
    if "gitops" in enabled_domains:
        validators["gitops"] = GitOpsValidator(policy_engine=policy_engine)

    return validators


def get_validator_for_file(file_path: str, validators: Dict[str, Validator]) -> Optional[Validator]:
    """Get appropriate validator for a file based on content/name.

    Args:
        file_path: Path to the file
        validators: Dictionary of available validators

    Returns:
        Appropriate validator or None
    """
    file_path_lower = file_path.lower()

    # Check filename patterns
    if "topic" in file_path_lower or "kafka" in file_path_lower:
        return validators.get("kafka")

    if "deployment" in file_path_lower or "pod" in file_path_lower or "service" in file_path_lower:
        return validators.get("kubernetes")

    if file_path.endswith((".tf", ".hcl")):
        return validators.get("iac")

    if "workflow" in file_path_lower or ".github" in file_path_lower:
        return validators.get("cicd")

    if ".gitlab-ci" in file_path_lower:
        return validators.get("cicd")

    if "kustomization" in file_path_lower or "gitrepository" in file_path_lower:
        return validators.get("gitops")

    if "application" in file_path_lower and ("argocd" in file_path_lower or "argo" in file_path_lower):
        return validators.get("gitops")

    # Default: try all validators
    return None


def detect_format(file_path: str) -> str:
    """Detect file format from extension.

    Args:
        file_path: Path to the file

    Returns:
        Format string (yaml, json, hcl, tf)
    """
    file_path_lower = file_path.lower()

    if file_path_lower.endswith((".yaml", ".yml")):
        return "yaml"
    elif file_path_lower.endswith(".json"):
        return "json"
    elif file_path_lower.endswith(".tf"):
        return "tf"
    elif file_path_lower.endswith(".hcl"):
        return "hcl"
    else:
        return "yaml"  # Default
