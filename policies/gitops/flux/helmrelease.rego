package gitops.flux.helmrelease

import future.keywords.if
import future.keywords.in

# Configuration
default require_rollback := true
default require_test := false
default max_history := 10
default require_timeout := true
default default_timeout := 300  # 5 minutes

# Deny if HelmRelease has no chart reference
deny[msg] {
    release := input

    not release.spec.chart

    msg := "HelmRelease must specify chart reference"
}

# Deny if no interval specified
deny[msg] {
    release := input

    not release.spec.interval

    msg := "HelmRelease must specify reconciliation interval"
}

# Deny if rollback not configured for production
deny[msg] {
    require_rollback
    release := input

    is_production(release)
    not release.spec.rollback

    msg := "Production HelmRelease should configure automatic rollback"
}

deny[msg] {
    require_rollback
    release := input

    is_production(release)
    rollback := release.spec.rollback
    rollback.enable == false

    msg := "Production HelmRelease has rollback disabled"
}

# Deny if history limit too high
deny[msg] {
    release := input
    install := release.spec.install

    install.remediation
    retries := install.remediation.retries
    retries > max_history

    msg := sprintf(
        "HelmRelease install retries %d exceeds maximum %d",
        [retries, max_history]
    )
}

# Deny if no timeout set
deny[msg] {
    require_timeout
    release := input

    not release.spec.timeout

    msg := sprintf(
        "HelmRelease should set timeout (default: %ds) to prevent hanging installations",
        [default_timeout]
    )
}

# Warn if no upgrade configuration
warn[msg] {
    release := input

    not release.spec.upgrade

    msg := "HelmRelease should configure upgrade behavior (force, cleanupOnFail, etc.)"
}

# Warn if force upgrade enabled
warn[msg] {
    release := input
    upgrade := release.spec.upgrade

    upgrade.force == true

    msg := "HelmRelease has force upgrade enabled - can cause unexpected resource replacements"
}

# Deny if values from untrusted sources
deny[msg] {
    release := input
    values_from := release.spec.valuesFrom[_]

    kind := values_from.kind
    kind == "ConfigMap"

    not values_from.name

    msg := "HelmRelease valuesFrom ConfigMap must specify name"
}

# Warn if no test configured for production
warn[msg] {
    require_test
    release := input

    is_production(release)
    not release.spec.test

    msg := "Production HelmRelease should enable Helm tests to verify deployment"
}

# Deny if crds policy is Skip for new charts
deny[msg] {
    release := input
    install := release.spec.install

    install.crds == "Skip"

    msg := "HelmRelease install.crds set to Skip - CRDs won't be installed"
}

# Warn if no dependency update configured
warn[msg] {
    release := input

    not release.spec.dependsOn

    msg := "HelmRelease should declare dependencies using dependsOn for proper ordering"
}

# Deny if chart version not pinned
deny[msg] {
    release := input
    chart := release.spec.chart.spec

    not chart.version

    msg := "HelmRelease should pin chart version for reproducible deployments"
}

warn[msg] {
    release := input
    chart := release.spec.chart.spec

    version := chart.version
    is_string(version)

    # Using ranges like "1.x" or "^1.0.0"
    mutable_patterns := ["x", "*", "~", "^"]
    pattern := mutable_patterns[_]
    contains(version, pattern)

    msg := sprintf(
        "HelmRelease chart version '%s' uses range - pin to specific version",
        [version]
    )
}

# Deny if suspend is true (frozen releases)
warn[msg] {
    release := input

    release.spec.suspend == true

    msg := "HelmRelease is suspended - won't reconcile until unsuspended"
}

# Recommend using post-renderers for customization
recommend[suggestion] {
    release := input

    # Has values but no post-renderer
    release.spec.values
    not release.spec.postRenderers

    suggestion := {
        "field": "spec.postRenderers",
        "recommendation": "Consider using Kustomize post-renderer for advanced customization",
        "reason": "Allows applying patches without modifying values"
    }
}

# Recommend service account for RBAC
recommend[suggestion] {
    release := input

    is_production(release)
    not release.spec.serviceAccountName

    suggestion := {
        "field": "spec.serviceAccountName",
        "recommended": "Use dedicated ServiceAccount",
        "reason": "Enables proper RBAC and least privilege"
    }
}

# Helper functions
is_production(release) {
    namespace := release.metadata.namespace
    is_string(namespace)
    production_namespaces := ["production", "prod", "live"]
    namespace in production_namespaces
}

is_production(release) {
    target := release.spec.targetNamespace
    is_string(target)
    production_namespaces := ["production", "prod", "live"]
    target in production_namespaces
}
