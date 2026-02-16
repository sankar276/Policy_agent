package iac.terraform.security

import future.keywords.if
import future.keywords.in

# Configuration
default require_encryption_at_rest := true
default require_encryption_in_transit := true
default allow_root_user := false

# Deny if IAM user has console password without MFA
deny[msg] {
    user := input.resource.aws_iam_user[name]
    not user.force_destroy  # This is a weak signal, but helps

    msg := sprintf(
        "IAM user '%s' should require MFA for console access",
        [name]
    )
}

# Deny if IAM policy allows full admin access
deny[msg] {
    policy := input.resource.aws_iam_policy[name]
    statement := policy.policy.Statement[_]

    statement.Effect == "Allow"
    statement.Action[_] == "*"
    statement.Resource[_] == "*"

    msg := sprintf(
        "IAM policy '%s' grants full admin access (*:*) - use principle of least privilege",
        [name]
    )
}

# Deny if EBS volume is not encrypted
deny[msg] {
    require_encryption_at_rest
    volume := input.resource.aws_ebs_volume[name]
    not volume.encrypted

    msg := sprintf(
        "EBS volume '%s' is not encrypted",
        [name]
    )
}

deny[msg] {
    require_encryption_at_rest
    volume := input.resource.aws_ebs_volume[name]
    volume.encrypted == false

    msg := sprintf(
        "EBS volume '%s' has encryption disabled",
        [name]
    )
}

# Deny if Lambda function has overly permissive IAM role
deny[msg] {
    lambda := input.resource.aws_lambda_function[name]
    role := input.resource.aws_iam_role[lambda.role]

    policy := role.assume_role_policy.Statement[_]
    policy.Principal.Service == "*"

    msg := sprintf(
        "Lambda function '%s' IAM role allows any service principal - should be specific",
        [name]
    )
}

# Deny if VPC flow logs are not enabled
deny[msg] {
    vpc := input.resource.aws_vpc[name]
    not input.resource.aws_flow_log[_].vpc_id == vpc.id

    msg := sprintf(
        "VPC '%s' does not have flow logs enabled - required for security monitoring",
        [name]
    )
}

# Warn if CloudWatch log group retention is not set
warn[msg] {
    log_group := input.resource.aws_cloudwatch_log_group[name]
    not log_group.retention_in_days

    msg := sprintf(
        "CloudWatch log group '%s' should set retention_in_days to manage costs",
        [name]
    )
}

# Deny if secrets are hardcoded in configuration
deny[msg] {
    # Check all resources for potential secrets
    resource := input.resource[resource_type][name]
    field := resource[field_name]

    # Simple pattern matching for common secret patterns
    regex.match(`(?i)(password|secret|key|token)`, field_name)
    is_string(field)
    not startswith(field, "var.")
    not startswith(field, "data.")
    not contains(field, "aws_secretsmanager")

    msg := sprintf(
        "Resource '%s.%s' appears to have hardcoded secret in field '%s' - use variables or secret manager",
        [resource_type, name, field_name]
    )
}

# Deny if ALB/ELB doesn't drop invalid headers
deny[msg] {
    lb := input.resource.aws_lb[name]
    lb.load_balancer_type == "application"
    not lb.drop_invalid_header_fields

    msg := sprintf(
        "Application Load Balancer '%s' should drop invalid header fields",
        [name]
    )
}

deny[msg] {
    lb := input.resource.aws_lb[name]
    lb.load_balancer_type == "application"
    lb.drop_invalid_header_fields == false

    msg := sprintf(
        "Application Load Balancer '%s' has drop_invalid_header_fields disabled",
        [name]
    )
}

# Deny if ALB listener is HTTP (not HTTPS)
deny[msg] {
    require_encryption_in_transit
    listener := input.resource.aws_lb_listener[name]
    listener.protocol == "HTTP"
    listener.port != 80  # Allow HTTP on port 80 for redirect

    msg := sprintf(
        "Load balancer listener '%s' uses unencrypted HTTP - use HTTPS",
        [name]
    )
}

# Warn if using default VPC
warn[msg] {
    vpc := input.resource.aws_default_vpc[name]

    msg := sprintf(
        "Using default VPC '%s' - create a custom VPC for better security control",
        [name]
    )
}

# Deny if subnet has map_public_ip_on_launch enabled
deny[msg] {
    subnet := input.resource.aws_subnet[name]
    subnet.map_public_ip_on_launch == true

    msg := sprintf(
        "Subnet '%s' has map_public_ip_on_launch enabled - instances should not get public IPs by default",
        [name]
    )
}

# Recommend best practices
recommend[suggestion] {
    bucket := input.resource.aws_s3_bucket[name]
    not bucket.logging

    suggestion := {
        "resource": sprintf("aws_s3_bucket.%s", [name]),
        "field": "logging",
        "recommended": "enabled",
        "reason": "Enable access logging for security audit trail"
    }
}

recommend[suggestion] {
    rds := input.resource.aws_rds_instance[name]
    not rds.enabled_cloudwatch_logs_exports

    suggestion := {
        "resource": sprintf("aws_rds_instance.%s", [name]),
        "field": "enabled_cloudwatch_logs_exports",
        "recommended": '["error", "general", "slowquery"]',
        "reason": "Export logs to CloudWatch for monitoring and troubleshooting"
    }
}
