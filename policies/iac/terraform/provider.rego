package iac.terraform.provider

import future.keywords.if
import future.keywords.in

# Configuration
default require_version_pinning := true
default require_version_constraint := true

# Deny if provider version is not pinned
deny[msg] {
    require_version_pinning
    provider := input.terraform.required_providers[_]
    not provider.version
    msg := sprintf(
        "Provider '%s' missing version constraint - should pin to specific version",
        [provider.source]
    )
}

# Deny if provider version uses loose constraints
deny[msg] {
    require_version_constraint
    provider := input.terraform.required_providers[name]
    version := provider.version

    # Check for loose constraints (>=, ~>, etc. without upper bound)
    regex.match(`^>=\s*\d+`, version)
    not regex.match(`^>=\s*\d+.*<`, version)

    msg := sprintf(
        "Provider '%s' uses loose version constraint '%s' - should use ~> or specify upper bound",
        [name, version]
    )
}

# Warn if using very old Terraform version
warn[msg] {
    terraform_version := input.terraform.required_version
    regex.match(`^(0\.|1\.0|1\.1|1\.2)`, terraform_version)

    msg := sprintf(
        "Terraform version '%s' is outdated - consider upgrading to 1.5+",
        [terraform_version]
    )
}

# Deny if required_version is missing
deny[msg] {
    not input.terraform.required_version
    msg := "Missing terraform.required_version - should specify minimum Terraform version"
}

# Recommend best practices
recommend[suggestion] {
    provider := input.terraform.required_providers[name]
    not provider.version

    suggestion := {
        "provider": name,
        "field": sprintf("terraform.required_providers.%s.version", [name]),
        "recommended": "~> 5.0",
        "reason": "Pin provider versions for reproducible deployments"
    }
}
