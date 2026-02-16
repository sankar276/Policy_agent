"""Infrastructure as Code (IaC) domain validator."""

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
class IaCConfig:
    """IaC validation configuration."""

    required_tags: List[str] = None
    require_version_pinning: bool = True
    require_remote_state: bool = True
    require_encryption: bool = True
    allowed_environments: List[str] = None
    require_naming_convention: bool = True

    def __post_init__(self):
        if self.required_tags is None:
            self.required_tags = ["Environment", "Owner", "Project", "ManagedBy"]
        if self.allowed_environments is None:
            self.allowed_environments = ["dev", "staging", "prod"]


class IaCValidator(Validator):
    """Validator for Infrastructure as Code configurations (Terraform, CloudFormation, etc.)."""

    # Taggable AWS resources
    TAGGABLE_RESOURCES = {
        "aws_instance",
        "aws_s3_bucket",
        "aws_rds_instance",
        "aws_ebs_volume",
        "aws_vpc",
        "aws_subnet",
        "aws_security_group",
        "aws_lambda_function",
        "azurerm_resource_group",
        "azurerm_virtual_machine",
        "azurerm_storage_account",
        "google_compute_instance",
        "google_storage_bucket",
    }

    def __init__(self, policy_engine: Optional[PolicyEngine] = None, config: Optional[IaCConfig] = None):
        """Initialize IaC validator.

        Args:
            policy_engine: Policy engine for OPA evaluation
            config: IaC-specific configuration
        """
        self.policy_engine = policy_engine
        self.config = config or IaCConfig()

    def domain(self) -> str:
        """Return domain name."""
        return "iac"

    def supported_types(self) -> List[str]:
        """Return supported IaC resource types."""
        return ["TerraformConfig", "CloudFormation", "Pulumi"]

    def validate(self, input_data: Dict[str, Any], file_path: Optional[str] = None) -> ValidationResult:
        """Validate IaC configuration.

        Args:
            input_data: Parsed configuration
            file_path: Optional source file path

        Returns:
            ValidationResult with violations and warnings
        """
        # Determine resource type
        resource_type = input_data.get("kind", "TerraformConfig")
        if resource_type not in self.supported_types():
            raise ValueError(f"Unsupported IaC resource type: {resource_type}")

        # Extract resource name
        resource_name = self._extract_resource_name(input_data)

        # Create result
        result = ValidationResult(
            status="passed",
            domain=self.domain(),
            resource=resource_name,
            resource_type=resource_type,
            file=file_path,
            timestamp=datetime.now(),
        )

        # Validate based on resource type
        if resource_type == "TerraformConfig":
            self._validate_terraform(input_data, result)
        elif resource_type == "CloudFormation":
            self._validate_cloudformation(input_data, result)
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

    def _validate_terraform(self, data: Dict[str, Any], result: ValidationResult) -> None:
        """Validate Terraform configuration.

        Args:
            data: Terraform configuration data
            result: ValidationResult to populate
        """
        # Check provider configuration
        self._check_provider_config(data, result)

        # Check state backend configuration
        self._check_state_backend(data, result)

        # Check resources
        self._check_resources(data, result)

    def _check_provider_config(self, data: Dict[str, Any], result: ValidationResult) -> None:
        """Check provider configuration."""
        terraform = data.get("terraform", {})

        if not terraform:
            result.violations.append(
                Violation(
                    policy="iac.terraform.provider",
                    severity=Severity.HIGH,
                    message="Missing terraform configuration block",
                    field="terraform",
                )
            )
            return

        # Check required_version
        if "required_version" not in terraform:
            result.violations.append(
                Violation(
                    policy="iac.terraform.provider",
                    severity=Severity.MEDIUM,
                    message="Missing terraform.required_version - should specify minimum Terraform version",
                    field="terraform.required_version",
                    remediation=Remediation(
                        auto_fixable=True,
                        suggestion="Pin Terraform version for reproducible deployments",
                    ),
                )
            )

        # Check provider versions
        if self.config.require_version_pinning:
            providers = terraform.get("required_providers", {})
            if not providers:
                result.violations.append(
                    Violation(
                        policy="iac.terraform.provider",
                        severity=Severity.HIGH,
                        message="Missing terraform.required_providers configuration",
                        field="terraform.required_providers",
                    )
                )
                return

            for name, provider_config in providers.items():
                if isinstance(provider_config, dict) and "version" not in provider_config:
                    result.violations.append(
                        Violation(
                            policy="iac.terraform.provider",
                            severity=Severity.HIGH,
                            message=f"Provider '{name}' missing version constraint",
                            field=f"terraform.required_providers.{name}.version",
                            remediation=Remediation(
                                auto_fixable=True,
                                suggestion="Pin provider versions to ensure reproducible deployments",
                            ),
                        )
                    )

        result.passed.append(
            PolicyCheck(policy="iac.terraform.provider.configured", message="Provider configuration present")
        )

    def _check_state_backend(self, data: Dict[str, Any], result: ValidationResult) -> None:
        """Check state backend configuration."""
        if not self.config.require_remote_state:
            return

        terraform = data.get("terraform", {})
        if not terraform:
            return

        backend = terraform.get("backend", {})

        if not backend:
            result.violations.append(
                Violation(
                    policy="iac.terraform.state",
                    severity=Severity.HIGH,
                    message="No backend configuration found - configure remote state backend",
                    field="terraform.backend",
                    remediation=Remediation(
                        auto_fixable=False,
                        suggestion="Use remote backend (s3, azurerm, gcs) for production state management",
                    ),
                )
            )
            return

        # Check for local backend
        if "local" in backend:
            result.violations.append(
                Violation(
                    policy="iac.terraform.state",
                    severity=Severity.HIGH,
                    message="Local backend detected - use remote backend (s3, azurerm, gcs) for production",
                    field="terraform.backend.local",
                )
            )
            return

        # Check S3 backend encryption
        if "s3" in backend:
            s3_config = backend["s3"]
            if self.config.require_encryption:
                encrypt = s3_config.get("encrypt", False)
                if not encrypt:
                    result.violations.append(
                        Violation(
                            policy="iac.terraform.state",
                            severity=Severity.HIGH,
                            message="S3 backend missing encryption - set 'encrypt = true'",
                            field="terraform.backend.s3.encrypt",
                            current_value=encrypt,
                            expected_value=True,
                            remediation=Remediation(
                                auto_fixable=True,
                                suggestion="Enable S3 encryption to protect state file contents",
                            ),
                        )
                    )

            # Check for DynamoDB locking
            if "dynamodb_table" not in s3_config:
                result.warnings.append(
                    Warning(
                        policy="iac.terraform.state",
                        severity=Severity.MEDIUM,
                        message="S3 backend missing DynamoDB table for state locking",
                        field="terraform.backend.s3.dynamodb_table",
                    )
                )
            else:
                result.passed.append(
                    PolicyCheck(
                        policy="iac.terraform.state.locking",
                        message="State locking configured with DynamoDB",
                    )
                )

    def _check_resources(self, data: Dict[str, Any], result: ValidationResult) -> None:
        """Check resource configurations."""
        resources = data.get("resource", {})

        for resource_type, resources_of_type in resources.items():
            if not isinstance(resources_of_type, dict):
                continue

            for name, resource_config in resources_of_type.items():
                if not isinstance(resource_config, dict):
                    continue

                self._check_resource_tags(resource_type, name, resource_config, result)
                self._check_resource_security(resource_type, name, resource_config, result)

    def _check_resource_tags(
        self, resource_type: str, name: str, resource: Dict[str, Any], result: ValidationResult
    ) -> None:
        """Check resource tagging."""
        if resource_type not in self.TAGGABLE_RESOURCES:
            return

        tags = resource.get("tags", {})
        labels = resource.get("labels", {})  # GCP uses labels

        if not tags and not labels:
            result.violations.append(
                Violation(
                    policy="iac.terraform.resources",
                    severity=Severity.MEDIUM,
                    message=f"Resource '{resource_type}.{name}' missing tags",
                    field=f"resource.{resource_type}.{name}.tags",
                )
            )
            return

        # Check required tags
        tag_map = tags if tags else labels
        for required_tag in self.config.required_tags:
            if required_tag not in tag_map:
                result.violations.append(
                    Violation(
                        policy="iac.terraform.resources",
                        severity=Severity.MEDIUM,
                        message=f"Resource '{resource_type}.{name}' missing required tag '{required_tag}'",
                        field=f"resource.{resource_type}.{name}.tags.{required_tag}",
                    )
                )

    def _check_resource_security(
        self, resource_type: str, name: str, resource: Dict[str, Any], result: ValidationResult
    ) -> None:
        """Check resource security configurations."""
        if resource_type == "aws_s3_bucket":
            self._check_s3_security(name, resource, result)
        elif resource_type == "aws_rds_instance":
            self._check_rds_security(name, resource, result)
        elif resource_type == "aws_security_group":
            self._check_security_group_rules(name, resource, result)

    def _check_s3_security(self, name: str, bucket: Dict[str, Any], result: ValidationResult) -> None:
        """Check S3 bucket security."""
        # Check for public access
        acl = bucket.get("acl", "")
        if "public" in acl.lower():
            result.violations.append(
                Violation(
                    policy="iac.terraform.security",
                    severity=Severity.CRITICAL,
                    message=f"S3 bucket '{name}' has public ACL - should be private",
                    field=f"resource.aws_s3_bucket.{name}.acl",
                    current_value=acl,
                    expected_value="private",
                )
            )

        # Check encryption
        if self.config.require_encryption:
            if "server_side_encryption_configuration" not in bucket:
                result.violations.append(
                    Violation(
                        policy="iac.terraform.security",
                        severity=Severity.HIGH,
                        message=f"S3 bucket '{name}' missing server-side encryption",
                        field=f"resource.aws_s3_bucket.{name}.server_side_encryption_configuration",
                    )
                )

    def _check_rds_security(self, name: str, rds: Dict[str, Any], result: ValidationResult) -> None:
        """Check RDS security."""
        # Check public accessibility
        if rds.get("publicly_accessible", False):
            result.violations.append(
                Violation(
                    policy="iac.terraform.security",
                    severity=Severity.CRITICAL,
                    message=f"RDS instance '{name}' is publicly accessible",
                    field=f"resource.aws_rds_instance.{name}.publicly_accessible",
                    current_value=True,
                    expected_value=False,
                )
            )

        # Check encryption
        if self.config.require_encryption:
            if not rds.get("storage_encrypted", False):
                result.violations.append(
                    Violation(
                        policy="iac.terraform.security",
                        severity=Severity.HIGH,
                        message=f"RDS instance '{name}' does not have storage encryption enabled",
                        field=f"resource.aws_rds_instance.{name}.storage_encrypted",
                    )
                )

        # Check backup retention
        retention = rds.get("backup_retention_period", 0)
        if retention < 7:
            result.violations.append(
                Violation(
                    policy="iac.terraform.security",
                    severity=Severity.MEDIUM,
                    message=f"RDS instance '{name}' has backup retention period {retention} days - should be at least 7 days",
                    field=f"resource.aws_rds_instance.{name}.backup_retention_period",
                    current_value=retention,
                    expected_value=7,
                )
            )

    def _check_security_group_rules(
        self, name: str, sg: Dict[str, Any], result: ValidationResult
    ) -> None:
        """Check security group rules."""
        ingress_rules = sg.get("ingress", [])

        for i, rule in enumerate(ingress_rules):
            if not isinstance(rule, dict):
                continue

            cidr_blocks = rule.get("cidr_blocks", [])
            from_port = rule.get("from_port", 0)

            if "0.0.0.0/0" in cidr_blocks and from_port not in [80, 443]:
                result.violations.append(
                    Violation(
                        policy="iac.terraform.security",
                        severity=Severity.HIGH,
                        message=f"Security group '{name}' has overly permissive rule allowing 0.0.0.0/0 on port {from_port}",
                        field=f"resource.aws_security_group.{name}.ingress[{i}].cidr_blocks",
                    )
                )

    def _validate_cloudformation(self, data: Dict[str, Any], result: ValidationResult) -> None:
        """Validate CloudFormation templates.

        Args:
            data: CloudFormation template data
            result: ValidationResult to populate
        """
        result.warnings.append(
            Warning(
                policy="iac.cloudformation",
                severity=Severity.LOW,
                message="CloudFormation validation not yet implemented",
            )
        )

    def _extract_resource_name(self, data: Dict[str, Any]) -> str:
        """Extract resource name from configuration."""
        metadata = data.get("metadata", {})
        return metadata.get("name", "terraform-config")
