package iac.terraform.state

import future.keywords.if
import future.keywords.in

# Configuration
default require_remote_state := true
default require_state_encryption := true
default allowed_backends := ["s3", "azurerm", "gcs", "remote"]

# Deny if using local backend in production
deny[msg] {
    require_remote_state
    backend := input.terraform.backend[name]
    name == "local"

    msg := "Local backend detected - use remote backend (s3, azurerm, gcs) for production"
}

# Deny if backend is missing
deny[msg] {
    require_remote_state
    not input.terraform.backend

    msg := "No backend configuration found - configure remote state backend"
}

# Deny if S3 backend without encryption
deny[msg] {
    require_state_encryption
    backend := input.terraform.backend.s3
    not backend.encrypt

    msg := "S3 backend missing encryption - set 'encrypt = true'"
}

deny[msg] {
    require_state_encryption
    backend := input.terraform.backend.s3
    backend.encrypt == false

    msg := "S3 backend has encryption disabled - set 'encrypt = true'"
}

# Deny if S3 backend without versioning
deny[msg] {
    backend := input.terraform.backend.s3
    not backend.versioning

    msg := "S3 backend should enable versioning for state recovery"
}

# Warn if S3 backend without DynamoDB locking
warn[msg] {
    backend := input.terraform.backend.s3
    not backend.dynamodb_table

    msg := "S3 backend missing DynamoDB table for state locking - consider adding for concurrent access protection"
}

# Deny if Azure backend without encryption
deny[msg] {
    require_state_encryption
    backend := input.terraform.backend.azurerm
    not backend.use_microsoft_graph
    not backend.use_azuread_auth

    msg := "Azure backend should use managed identity or service principal authentication"
}

# Recommend best practices
recommend[suggestion] {
    backend := input.terraform.backend.s3
    not backend.dynamodb_table

    suggestion := {
        "field": "terraform.backend.s3.dynamodb_table",
        "recommended": "terraform-state-lock",
        "reason": "Enables state locking to prevent concurrent modifications"
    }
}
