"""AI client interface for policy generation and remediation."""

from abc import ABC, abstractmethod
from typing import List, Optional

from policy_agent.types.result import (
    GenerateRequest,
    GenerateResponse,
    FixResponse,
    Violation,
)


class AIClient(ABC):
    """Abstract base class for AI clients."""

    @abstractmethod
    def generate(self, request: GenerateRequest) -> GenerateResponse:
        """Generate policy-compliant configuration.

        Args:
            request: Generation request with requirements

        Returns:
            GenerateResponse with generated configuration
        """
        pass

    @abstractmethod
    def remediate(
        self, domain: str, original_config: str, violations: List[Violation]
    ) -> FixResponse:
        """Suggest fixes for policy violations.

        Args:
            domain: Domain name (kafka, kubernetes, etc.)
            original_config: Original configuration with violations
            violations: List of policy violations to fix

        Returns:
            FixResponse with fixed configuration and explanations
        """
        pass

    @abstractmethod
    def explain(self, policy: str, violation: Optional[Violation] = None) -> str:
        """Explain a policy or violation.

        Args:
            policy: Policy name to explain
            violation: Optional specific violation to explain

        Returns:
            Human-readable explanation
        """
        pass
