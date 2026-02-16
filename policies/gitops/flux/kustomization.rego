package gitops.flux.kustomization

import future.keywords.if
import future.keywords.in

# Configuration
default require_prune := true
default require_health_checks := true
default max_retry_interval := 300  # 5 minutes
default require_service_account := true
default allow_force_sync := false

# Deny if Kustomization doesn't have prune enabled
deny[msg] {
    require_prune
    kustomization := input

    not kustomization.spec.prune

    msg := "Kustomization should enable 'prune' to clean up deleted resources"
}

deny[msg] {
    require_prune
    kustomization := input

    kustomization.spec.prune == false

    msg := "Kustomization has 'prune' disabled - orphaned resources won't be cleaned up"
}

# Deny if no source reference
deny[msg] {
    kustomization := input

    not kustomization.spec.sourceRef

    msg := "Kustomization must specify sourceRef (GitRepository, Bucket, etc.)"
}

# Deny if interval is too short
deny[msg] {
    kustomization := input
    interval := kustomization.spec.interval

    is_string(interval)
    interval_seconds := parse_duration_to_seconds(interval)
    interval_seconds < 60

    msg := sprintf(
        "Kustomization interval '%s' is too short - minimum 1 minute to avoid excessive reconciliation",
        [interval]
    )
}

# Deny if retry interval is too long
deny[msg] {
    kustomization := input
    retry := kustomization.spec.retryInterval

    is_string(retry)
    retry_seconds := parse_duration_to_seconds(retry)
    retry_seconds > max_retry_interval

    msg := sprintf(
        "Kustomization retryInterval '%s' exceeds maximum %d seconds",
        [retry, max_retry_interval]
    )
}

# Deny if force sync is enabled (dangerous)
deny[msg] {
    not allow_force_sync
    kustomization := input

    kustomization.spec.force == true

    msg := "Kustomization has 'force' enabled - can cause unexpected resource replacements"
}

# Warn if no health checks configured
warn[msg] {
    require_health_checks
    kustomization := input

    not kustomization.spec.healthChecks

    msg := "Kustomization should define healthChecks to verify deployment success"
}

# Deny if service account not specified for production
deny[msg] {
    require_service_account
    kustomization := input

    is_production_namespace(kustomization)
    not kustomization.spec.serviceAccountName

    msg := "Production Kustomization should use dedicated ServiceAccount for least privilege"
}

# Warn if no timeout set
warn[msg] {
    kustomization := input

    not kustomization.spec.timeout

    msg := "Kustomization should set timeout to prevent hanging reconciliations"
}

# Deny if targeting multiple namespaces without proper RBAC
deny[msg] {
    kustomization := input

    kustomization.spec.targetNamespace
    not kustomization.spec.serviceAccountName

    msg := "Kustomization targeting specific namespace should use ServiceAccount with proper RBAC"
}

# Warn if no validation configured
warn[msg] {
    kustomization := input

    validation := kustomization.spec.validation
    validation == "none"

    msg := "Kustomization has validation disabled - manifests won't be validated before apply"
}

# Deny if path traversal in path
deny[msg] {
    kustomization := input
    path := kustomization.spec.path

    is_string(path)
    contains(path, "..")

    msg := sprintf(
        "Kustomization path '%s' contains path traversal - security risk",
        [path]
    )
}

# Recommend using wait for production
recommend[suggestion] {
    kustomization := input

    is_production_namespace(kustomization)
    not kustomization.spec.wait

    suggestion := {
        "field": "spec.wait",
        "recommended": "true",
        "reason": "Wait for all resources to be ready in production deployments"
    }
}

# Helper functions
parse_duration_to_seconds(duration) = seconds {
    # Simple parser for durations like "1m", "5m", "1h"
    contains(duration, "s")
    trimmed := trim_suffix(duration, "s")
    seconds := to_number(trimmed)
}

parse_duration_to_seconds(duration) = seconds {
    contains(duration, "m")
    trimmed := trim_suffix(duration, "m")
    minutes := to_number(trimmed)
    seconds := minutes * 60
}

parse_duration_to_seconds(duration) = seconds {
    contains(duration, "h")
    trimmed := trim_suffix(duration, "h")
    hours := to_number(trimmed)
    seconds := hours * 3600
}

is_production_namespace(kustomization) {
    namespace := kustomization.metadata.namespace
    is_string(namespace)
    production_namespaces := ["production", "prod", "live"]
    namespace in production_namespaces
}

is_production_namespace(kustomization) {
    target := kustomization.spec.targetNamespace
    is_string(target)
    production_namespaces := ["production", "prod", "live"]
    target in production_namespaces
}
