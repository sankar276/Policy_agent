"""Base validator interface and registry."""

from abc import ABC, abstractmethod
from typing import Any, Dict, List, Optional

from policy_agent.types.result import ValidationResult, Format


class Validator(ABC):
    """Base interface for domain validators."""

    @abstractmethod
    def validate(self, input_data: Dict[str, Any], file_path: Optional[str] = None) -> ValidationResult:
        """Validate input against policies.

        Args:
            input_data: Parsed configuration data
            file_path: Optional source file path

        Returns:
            ValidationResult with violations, warnings, and passed checks
        """
        pass

    @abstractmethod
    def domain(self) -> str:
        """Return the domain name (kafka, kubernetes, etc.)."""
        pass

    @abstractmethod
    def supported_types(self) -> List[str]:
        """Return supported resource types.

        Returns:
            List of resource type names (e.g., ["KafkaTopic", "KafkaConnector"])
        """
        pass

    def can_validate(self, input_data: Dict[str, Any]) -> bool:
        """Determine if this validator can handle the input.

        Args:
            input_data: Parsed configuration data

        Returns:
            True if this validator can handle the input
        """
        kind = input_data.get("kind", "")
        return kind in self.supported_types()


class ValidatorRegistry:
    """Registry for managing validators."""

    def __init__(self) -> None:
        """Initialize empty registry."""
        self._validators: Dict[str, Validator] = {}
        self._type_map: Dict[str, Validator] = {}

    def register(self, validator: Validator) -> None:
        """Register a validator.

        Args:
            validator: Validator instance to register

        Raises:
            ValueError: If domain already registered
        """
        domain = validator.domain()
        if domain in self._validators:
            raise ValueError(f"Validator for domain '{domain}' already registered")

        self._validators[domain] = validator

        # Map resource types to validators
        for resource_type in validator.supported_types():
            self._type_map[resource_type] = validator

    def get(self, domain: str) -> Validator:
        """Get validator for a domain.

        Args:
            domain: Domain name

        Returns:
            Validator instance

        Raises:
            ValueError: If no validator found for domain
        """
        if domain not in self._validators:
            raise ValueError(f"No validator found for domain: {domain}")
        return self._validators[domain]

    def get_for_resource_type(self, resource_type: str) -> Validator:
        """Get validator that can handle a resource type.

        Args:
            resource_type: Resource type name

        Returns:
            Validator instance

        Raises:
            ValueError: If no validator found for resource type
        """
        if resource_type not in self._type_map:
            raise ValueError(f"No validator found for resource type: {resource_type}")
        return self._type_map[resource_type]

    def list(self) -> List[Validator]:
        """Get all registered validators.

        Returns:
            List of validator instances
        """
        return list(self._validators.values())

    def domains(self) -> List[str]:
        """Get all registered domain names.

        Returns:
            List of domain names
        """
        return list(self._validators.keys())
