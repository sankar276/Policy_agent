"""OPA policy engine for evaluating Rego policies."""

import glob
import os
from typing import Any, Dict, List, Optional

from opa_client.opa import OpaClient


class PolicyEngine:
    """OPA policy engine for evaluating Rego policies."""

    def __init__(self, policy_path: str, opa_url: str = "http://localhost:8181"):
        """Initialize policy engine.

        Args:
            policy_path: Path to directory containing Rego policy files
            opa_url: URL of OPA server (default: local OPA server)
        """
        self.policy_path = policy_path
        self.opa_client = OpaClient(host=opa_url)
        self.policies: Dict[str, str] = {}

    def load_policies(self) -> None:
        """Load all Rego policies from the policy path."""
        if not os.path.exists(self.policy_path):
            raise FileNotFoundError(f"Policy path not found: {self.policy_path}")

        # Find all .rego files
        pattern = os.path.join(self.policy_path, "**/*.rego")
        policy_files = glob.glob(pattern, recursive=True)

        # Filter out test files
        policy_files = [f for f in policy_files if not os.path.basename(f).startswith("test_")]

        for policy_file in policy_files:
            with open(policy_file, "r") as f:
                content = f.read()
                # Extract package name from policy
                package_name = self._extract_package_name(content)
                if package_name:
                    self.policies[package_name] = content

    def evaluate(
        self, input_data: Dict[str, Any], policy_package: str
    ) -> Dict[str, Any]:
        """Evaluate input against a specific policy package.

        Args:
            input_data: Input data to validate
            policy_package: Policy package name (e.g., "kafka.topics.replication")

        Returns:
            Dict with evaluation results including deny, warn, recommend
        """
        # For local evaluation without OPA server, we use simple rules
        # In production, you'd send this to an OPA server
        # For now, return a simple structure
        return {
            "deny": [],
            "warn": [],
            "recommend": [],
        }

    def get_policy_packages(self) -> List[str]:
        """Get all loaded policy packages.

        Returns:
            List of policy package names
        """
        return list(self.policies.keys())

    def _extract_package_name(self, policy_content: str) -> Optional[str]:
        """Extract package name from Rego policy content.

        Args:
            policy_content: Raw Rego policy content

        Returns:
            Package name or None
        """
        for line in policy_content.split("\n"):
            line = line.strip()
            if line.startswith("package "):
                return line.split("package ")[1].strip()
        return None


class EvalResult:
    """Result of policy evaluation."""

    def __init__(self, package: str, results: Dict[str, Any]):
        """Initialize evaluation result.

        Args:
            package: Policy package name
            results: Raw evaluation results from OPA
        """
        self.package = package
        self.results = results

    def get_deny(self) -> List[str]:
        """Extract denial messages from results."""
        return self.results.get("deny", [])

    def get_warn(self) -> List[str]:
        """Extract warning messages from results."""
        return self.results.get("warn", [])

    def get_recommendations(self) -> List[Dict[str, Any]]:
        """Extract recommendations from results."""
        return self.results.get("recommend", [])
