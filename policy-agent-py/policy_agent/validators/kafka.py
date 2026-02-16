"""Kafka domain validator."""

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
class KafkaConfig:
    """Kafka validation configuration."""

    min_replication_factor: int = 3
    require_min_insync_replicas: bool = True
    min_insync_replicas_value: int = 2
    require_compression: bool = True
    allowed_compression_types: List[str] = None
    max_retention_days: int = 90
    warn_retention_days: int = 60

    def __post_init__(self):
        if self.allowed_compression_types is None:
            self.allowed_compression_types = ["lz4", "snappy", "zstd"]


class KafkaValidator(Validator):
    """Validator for Kafka resources (topics, connectors, etc.)."""

    def __init__(self, policy_engine: Optional[PolicyEngine] = None, config: Optional[KafkaConfig] = None):
        """Initialize Kafka validator.

        Args:
            policy_engine: Policy engine for OPA evaluation
            config: Kafka-specific configuration
        """
        self.policy_engine = policy_engine
        self.config = config or KafkaConfig()

    def domain(self) -> str:
        """Return domain name."""
        return "kafka"

    def supported_types(self) -> List[str]:
        """Return supported Kafka resource types."""
        return ["KafkaTopic", "KafkaConnector", "KafkaUser", "KafkaConnect"]

    def validate(self, input_data: Dict[str, Any], file_path: Optional[str] = None) -> ValidationResult:
        """Validate Kafka configuration.

        Args:
            input_data: Parsed YAML/JSON configuration
            file_path: Optional source file path

        Returns:
            ValidationResult with violations and warnings
        """
        kind = input_data.get("kind", "")
        if kind not in self.supported_types():
            raise ValueError(f"Unsupported Kafka resource type: {kind}")

        # Extract resource name
        resource_name = self._extract_resource_name(input_data)

        # Create result
        result = ValidationResult(
            status="passed",
            domain=self.domain(),
            resource=resource_name,
            resource_type=kind,
            file=file_path,
            timestamp=datetime.now(),
        )

        # Validate based on resource type
        if kind == "KafkaTopic":
            self._validate_topic(input_data, result)
        else:
            # Other resource types not yet implemented
            result.passed.append(
                PolicyCheck(
                    policy=f"{self.domain()}.{kind.lower()}",
                    message="Validation not yet implemented",
                )
            )

        # Set overall status
        if result.violations:
            result.status = "failed"
        elif result.warnings:
            result.status = "warning"

        return result

    def _validate_topic(self, data: Dict[str, Any], result: ValidationResult) -> None:
        """Validate Kafka topic configuration.

        Args:
            data: Topic configuration data
            result: ValidationResult to populate
        """
        spec = data.get("spec", {})
        config = spec.get("config", {})
        topic_name = self._extract_resource_name(data)

        # Validate replication
        self._check_replication(topic_name, spec, config, result)

        # Validate compression
        self._check_compression(topic_name, config, result)

        # Validate retention
        self._check_retention(topic_name, config, result)

    def _check_replication(
        self, topic_name: str, spec: Dict, config: Dict, result: ValidationResult
    ) -> None:
        """Check replication factor policies."""
        rf = spec.get("replicas", 0)
        min_rf = self.config.min_replication_factor

        # Check minimum replication factor
        if rf < min_rf:
            result.violations.append(
                Violation(
                    policy="kafka.topics.replication",
                    severity=Severity.HIGH,
                    message=f"Topic '{topic_name}' has insufficient replication factor {rf} (minimum: {min_rf})",
                    field="spec.replicas",
                    current_value=rf,
                    expected_value=min_rf,
                    remediation=Remediation(
                        auto_fixable=True,
                        suggestion=f"Ensures data durability and high availability across multiple brokers",
                    ),
                )
            )
        else:
            result.passed.append(
                PolicyCheck(policy="kafka.topics.replication.factor", message="Replication factor meets minimum")
            )

        # Check min.insync.replicas
        if self.config.require_min_insync_replicas:
            min_isr = config.get("min.insync.replicas")
            if not min_isr:
                result.violations.append(
                    Violation(
                        policy="kafka.topics.replication",
                        severity=Severity.HIGH,
                        message=f"Topic '{topic_name}' missing min.insync.replicas configuration",
                        field="spec.config.min.insync.replicas",
                        expected_value=str(self.config.min_insync_replicas_value),
                        remediation=Remediation(
                            auto_fixable=True,
                            suggestion="Guarantees that writes are acknowledged by at least this many replicas",
                        ),
                    )
                )
            else:
                isr_value = int(min_isr)
                if isr_value < self.config.min_insync_replicas_value:
                    result.violations.append(
                        Violation(
                            policy="kafka.topics.replication",
                            severity=Severity.HIGH,
                            message=f"Topic '{topic_name}' has min.insync.replicas {isr_value} (minimum: {self.config.min_insync_replicas_value})",
                            field="spec.config.min.insync.replicas",
                            current_value=isr_value,
                            expected_value=self.config.min_insync_replicas_value,
                        )
                    )

        # Warn for single replication
        if rf == 1:
            result.warnings.append(
                Warning(
                    policy="kafka.topics.replication",
                    severity=Severity.MEDIUM,
                    message=f"Topic '{topic_name}' has no replication (RF=1), significant data loss risk",
                    field="spec.replicas",
                )
            )

    def _check_compression(self, topic_name: str, config: Dict, result: ValidationResult) -> None:
        """Check compression policies."""
        compression = config.get("compression.type")

        if self.config.require_compression:
            if not compression:
                result.violations.append(
                    Violation(
                        policy="kafka.topics.compression",
                        severity=Severity.MEDIUM,
                        message=f"Topic '{topic_name}' missing compression configuration",
                        field="spec.config.compression.type",
                        expected_value="lz4",
                        remediation=Remediation(
                            auto_fixable=True,
                            suggestion="Reduces network bandwidth and storage costs with minimal CPU overhead",
                        ),
                    )
                )
                return

            if compression not in self.config.allowed_compression_types:
                result.violations.append(
                    Violation(
                        policy="kafka.topics.compression",
                        severity=Severity.MEDIUM,
                        message=f"Topic '{topic_name}' uses unsupported compression '{compression}' (allowed: {self.config.allowed_compression_types})",
                        field="spec.config.compression.type",
                        current_value=compression,
                        expected_value=self.config.allowed_compression_types[0],
                    )
                )
                return

            # Warn for suboptimal compression
            if compression == "gzip":
                result.warnings.append(
                    Warning(
                        policy="kafka.topics.compression",
                        severity=Severity.MEDIUM,
                        message=f"Topic '{topic_name}' uses 'gzip' compression which has higher CPU overhead (consider 'lz4' or 'zstd')",
                        field="spec.config.compression.type",
                    )
                )
            else:
                result.passed.append(
                    PolicyCheck(policy="kafka.topics.compression", message="Compression configured correctly")
                )

    def _check_retention(self, topic_name: str, config: Dict, result: ValidationResult) -> None:
        """Check retention policies."""
        retention_ms = config.get("retention.ms")

        if not retention_ms:
            result.violations.append(
                Violation(
                    policy="kafka.topics.retention",
                    severity=Severity.MEDIUM,
                    message=f"Topic '{topic_name}' missing retention.ms configuration",
                    field="spec.config.retention.ms",
                )
            )
            return

        # Convert to days
        try:
            retention_ms_val = int(retention_ms)
            retention_days = retention_ms_val / (1000 * 60 * 60 * 24)
        except (ValueError, TypeError):
            result.violations.append(
                Violation(
                    policy="kafka.topics.retention",
                    severity=Severity.MEDIUM,
                    message=f"Topic '{topic_name}' has invalid retention.ms value: {retention_ms}",
                    field="spec.config.retention.ms",
                )
            )
            return

        # Check maximum retention
        if retention_days > self.config.max_retention_days:
            result.violations.append(
                Violation(
                    policy="kafka.topics.retention",
                    severity=Severity.MEDIUM,
                    message=f"Topic '{topic_name}' retention {retention_days:.1f} days exceeds maximum {self.config.max_retention_days} days",
                    field="spec.config.retention.ms",
                    current_value=retention_ms_val,
                    expected_value=self.config.max_retention_days * 24 * 60 * 60 * 1000,
                )
            )
            return

        # Warn for high retention
        if retention_days > self.config.warn_retention_days:
            result.warnings.append(
                Warning(
                    policy="kafka.topics.retention",
                    severity=Severity.MEDIUM,
                    message=f"Topic '{topic_name}' retention {retention_days:.1f} days is high (consider reviewing storage costs)",
                    field="spec.config.retention.ms",
                )
            )
        else:
            result.passed.append(
                PolicyCheck(policy="kafka.topics.retention", message="Retention within limits")
            )

    def _extract_resource_name(self, data: Dict[str, Any]) -> str:
        """Extract resource name from configuration."""
        metadata = data.get("metadata", {})
        return metadata.get("name", "unknown")
