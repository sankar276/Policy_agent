"""Orchestrator for coordinating validation across domains."""

from datetime import datetime
from typing import Any, Dict, List, Optional

import yaml

from policy_agent.validators.base import Validator, ValidatorRegistry
from policy_agent.policy.engine import PolicyEngine
from policy_agent.types.result import (
    ValidationResult,
    GenerateRequest,
    GenerateResponse,
    FixRequest,
    FixResponse,
)


class Orchestrator:
    """Coordinates validation across multiple domains."""

    def __init__(
        self,
        validators: Optional[List[Validator]] = None,
        policy_engine: Optional[PolicyEngine] = None,
        ai_client: Optional[Any] = None,
    ):
        """Initialize orchestrator.

        Args:
            validators: List of validators to register
            policy_engine: Policy engine instance
            ai_client: AI client for generation/remediation
        """
        self.registry = ValidatorRegistry()
        self.policy_engine = policy_engine
        self.ai_client = ai_client

        if validators:
            for validator in validators:
                self.registry.register(validator)

    def validate(
        self,
        content: str,
        domain: Optional[str] = None,
        file_path: Optional[str] = None,
    ) -> ValidationResult:
        """Validate configuration content.

        Args:
            content: YAML/JSON configuration content
            domain: Specific domain to validate (optional)
            file_path: Source file path (optional)

        Returns:
            ValidationResult with violations and warnings
        """
        start_time = datetime.now()

        # Parse YAML
        try:
            data = yaml.safe_load(content)
        except yaml.YAMLError as e:
            raise ValueError(f"Failed to parse YAML: {e}")

        # Determine validator
        if domain:
            validator = self.registry.get(domain)
        else:
            # Auto-detect from resource kind
            kind = data.get("kind", "")
            if not kind:
                raise ValueError("Resource kind not specified and domain not provided")
            validator = self.registry.get_for_resource_type(kind)

        # Validate
        result = validator.validate(data, file_path)

        # Calculate duration
        duration = (datetime.now() - start_time).total_seconds()
        result.duration = duration

        return result

    def generate(self, request: GenerateRequest) -> GenerateResponse:
        """Generate policy-compliant configuration using AI.

        Args:
            request: Generation request

        Returns:
            GenerateResponse with generated configuration

        Raises:
            ValueError: If AI client not configured
        """
        if not self.ai_client:
            raise ValueError("AI client not configured")

        return self.ai_client.generate(request)

    def fix(self, request: FixRequest) -> FixResponse:
        """Fix policy violations using AI.

        Args:
            request: Fix request

        Returns:
            FixResponse with fixed configuration

        Raises:
            ValueError: If AI client not configured
        """
        if not self.ai_client:
            raise ValueError("AI client not configured")

        # First validate to find violations
        with open(request.file, "r") as f:
            content = f.read()

        result = self.validate(content, file_path=request.file)

        if not result.violations:
            # No violations, nothing to fix
            return FixResponse(
                original=content,
                fixed=content,
                changes=[],
                explanation="No policy violations found",
            )

        # Use AI to fix
        return self.ai_client.remediate(result.domain, content, result.violations)

    def get_available_domains(self) -> List[str]:
        """Get list of available validation domains.

        Returns:
            List of domain names
        """
        return self.registry.domains()
