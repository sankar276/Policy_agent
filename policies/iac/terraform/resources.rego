package iac.terraform.resources

import future.keywords.if
import future.keywords.in

# Configuration
default required_tags := ["Environment", "Owner", "Project", "ManagedBy"]
default require_naming_convention := true
default environment_values := ["dev", "staging", "prod"]

# Helper to check if resource supports tags
supports_tags(resource_type) {
    taggable_resources := {
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
    resource_type in taggable_resources
}

# Deny if required tags are missing
deny[msg] {
    resource := input.resource[resource_type][name]
    supports_tags(resource_type)

    missing_tag := required_tags[_]
    not resource.tags[missing_tag]
    not resource.labels[missing_tag]  # GCP uses labels

    msg := sprintf(
        "Resource '%s.%s' missing required tag '%s'",
        [resource_type, name, missing_tag]
    )
}

# Deny if Environment tag has invalid value
deny[msg] {
    resource := input.resource[resource_type][name]
    env := resource.tags.Environment

    not env in environment_values

    msg := sprintf(
        "Resource '%s.%s' has invalid Environment tag '%s' - must be one of: %v",
        [resource_type, name, env, environment_values]
    )
}

# Warn if resource name doesn't follow convention (lowercase with hyphens)
warn[msg] {
    require_naming_convention
    resource := input.resource[resource_type][name]

    not regex.match(`^[a-z0-9-]+$`, name)

    msg := sprintf(
        "Resource '%s.%s' name should use lowercase letters, numbers, and hyphens only",
        [resource_type, name]
    )
}

# Deny if S3 bucket doesn't have encryption
deny[msg] {
    bucket := input.resource.aws_s3_bucket[name]
    not bucket.server_side_encryption_configuration

    msg := sprintf(
        "S3 bucket '%s' missing server-side encryption configuration",
        [name]
    )
}

# Deny if S3 bucket has public access
deny[msg] {
    bucket := input.resource.aws_s3_bucket[name]
    bucket.acl == "public-read"

    msg := sprintf(
        "S3 bucket '%s' has public-read ACL - should be private",
        [name]
    )
}

deny[msg] {
    bucket := input.resource.aws_s3_bucket[name]
    bucket.acl == "public-read-write"

    msg := sprintf(
        "S3 bucket '%s' has public-read-write ACL - should be private",
        [name]
    )
}

# Deny if RDS instance is publicly accessible
deny[msg] {
    rds := input.resource.aws_rds_instance[name]
    rds.publicly_accessible == true

    msg := sprintf(
        "RDS instance '%s' is publicly accessible - should be private",
        [name]
    )
}

# Deny if RDS instance doesn't have backup retention
deny[msg] {
    rds := input.resource.aws_rds_instance[name]
    rds.backup_retention_period < 7

    msg := sprintf(
        "RDS instance '%s' has backup retention period %d days - should be at least 7 days",
        [name, rds.backup_retention_period]
    )
}

# Deny if RDS instance storage is not encrypted
deny[msg] {
    rds := input.resource.aws_rds_instance[name]
    not rds.storage_encrypted

    msg := sprintf(
        "RDS instance '%s' does not have storage encryption enabled",
        [name]
    )
}

deny[msg] {
    rds := input.resource.aws_rds_instance[name]
    rds.storage_encrypted == false

    msg := sprintf(
        "RDS instance '%s' has storage encryption disabled",
        [name]
    )
}

# Warn if EC2 instance doesn't have monitoring enabled
warn[msg] {
    instance := input.resource.aws_instance[name]
    not instance.monitoring

    msg := sprintf(
        "EC2 instance '%s' should enable detailed monitoring",
        [name]
    )
}

# Deny if security group has overly permissive ingress
deny[msg] {
    sg := input.resource.aws_security_group[name]
    rule := sg.ingress[_]

    rule.cidr_blocks[_] == "0.0.0.0/0"
    rule.from_port != 443
    rule.from_port != 80

    msg := sprintf(
        "Security group '%s' has overly permissive rule allowing 0.0.0.0/0 on port %d",
        [name, rule.from_port]
    )
}

# Recommend best practices
recommend[suggestion] {
    bucket := input.resource.aws_s3_bucket[name]
    not bucket.versioning

    suggestion := {
        "resource": sprintf("aws_s3_bucket.%s", [name]),
        "field": "versioning.enabled",
        "recommended": "true",
        "reason": "Enable versioning for data protection and recovery"
    }
}
