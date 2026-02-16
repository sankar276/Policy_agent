"""Result types for validation and generation."""

from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from typing import Any, Dict, List, Optional


class Severity(str, Enum):
    """Severity levels for violations."""

    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    INFO = "info"


class Format(str, Enum):
    """Configuration file formats."""

    YAML = "yaml"
    JSON = "json"
    HCL = "hcl"
    TOML = "toml"


@dataclass
class Remediation:
    """Remediation suggestion for a violation."""

    auto_fixable: bool
    suggestion: str
    diff: Optional[str] = None
    ai_explanation: Optional[str] = None


@dataclass
class Violation:
    """Policy violation."""

    policy: str
    severity: Severity
    message: str
    field: Optional[str] = None
    current_value: Optional[Any] = None
    expected_value: Optional[Any] = None
    remediation: Optional[Remediation] = None


@dataclass
class Warning:
    """Policy warning (non-blocking)."""

    policy: str
    severity: Severity
    message: str
    field: Optional[str] = None


@dataclass
class PolicyCheck:
    """Passed policy check."""

    policy: str
    message: Optional[str] = None


@dataclass
class ValidationResult:
    """Result of validating a configuration."""

    status: str  # "passed", "failed", "warning"
    domain: str
    resource: str
    resource_type: str
    violations: List[Violation] = field(default_factory=list)
    warnings: List[Warning] = field(default_factory=list)
    passed: List[PolicyCheck] = field(default_factory=list)
    file: Optional[str] = None
    timestamp: datetime = field(default_factory=datetime.now)
    duration: float = 0.0  # seconds


@dataclass
class GenerateRequest:
    """Request to generate configuration."""

    domain: str
    requirements: str
    policies: List[str] = field(default_factory=list)
    context: Dict[str, Any] = field(default_factory=dict)
    output: Optional[str] = None
    interactive: bool = False


@dataclass
class GenerateResponse:
    """Response from generation."""

    configuration: str
    format: Format
    explanation: str
    policies_met: List[str] = field(default_factory=list)


@dataclass
class FixRequest:
    """Request to fix policy violations."""

    file: str
    interactive: bool = False
    auto: bool = False
    dry_run: bool = False


@dataclass
class FixResponse:
    """Response from fixing violations."""

    original: str
    fixed: str
    changes: List[Dict[str, Any]]
    explanation: str
