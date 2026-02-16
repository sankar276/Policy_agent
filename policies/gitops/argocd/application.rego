package gitops.argocd.application

import future.keywords.if
import future.keywords.in

# Configuration
default require_auto_sync := false
default require_prune := true
default require_self_heal := false
default allow_empty_namespace := false
default require_sync_options := true

# Deny if Application has no source
deny[msg] {
    app := input

    not app.spec.source

    msg := "Application must specify source repository"
}

# Deny if Application has no destination
deny[msg] {
    app := input

    not app.spec.destination

    msg := "Application must specify destination cluster and namespace"
}

# Deny if destination namespace is empty
deny[msg] {
    not allow_empty_namespace
    app := input

    destination := app.spec.destination
    not destination.namespace

    msg := "Application destination must specify namespace"
}

# Deny if using in-cluster without proper RBAC
deny[msg] {
    app := input
    destination := app.spec.destination

    destination.server == "https://kubernetes.default.svc"
    not app.spec.project

    msg := "In-cluster Application should specify project for RBAC isolation"
}

# Deny if auto-sync enabled without prune
deny[msg] {
    require_prune
    app := input

    auto_sync := app.spec.syncPolicy.automated
    auto_sync

    not auto_sync.prune

    msg := "Application with automated sync should enable prune to clean up deleted resources"
}

deny[msg] {
    require_prune
    app := input

    auto_sync := app.spec.syncPolicy.automated
    auto_sync
    auto_sync.prune == false

    msg := "Application has prune disabled - deleted resources won't be cleaned up"
}

# Warn if auto-sync enabled for production without approval
warn[msg] {
    app := input

    is_production(app)
    auto_sync := app.spec.syncPolicy.automated
    auto_sync

    msg := "Production Application has automated sync - consider requiring manual approval"
}

# Warn if self-heal not enabled for production
warn[msg] {
    require_self_heal
    app := input

    is_production(app)
    auto_sync := app.spec.syncPolicy.automated

    not auto_sync.selfHeal

    msg := "Production Application should enable selfHeal to auto-recover from drift"
}

# Deny if using default project
deny[msg] {
    app := input

    project := app.spec.project
    project == "default"

    msg := "Application should use a dedicated Project (not 'default') for proper isolation"
}

# Deny if source is HTTP (not HTTPS or SSH)
deny[msg] {
    app := input
    repo_url := app.spec.source.repoURL

    startswith(repo_url, "http://")

    msg := sprintf(
        "Application source '%s' uses insecure HTTP - use HTTPS or SSH",
        [repo_url]
    )
}

# Deny if target revision is not specified
warn[msg] {
    app := input
    source := app.spec.source

    not source.targetRevision

    msg := "Application should specify targetRevision (branch, tag, or commit) for predictability"
}

# Deny if using HEAD as target revision
warn[msg] {
    app := input
    source := app.spec.source

    target := source.targetRevision
    target == "HEAD"

    msg := "Application targetRevision 'HEAD' is unpredictable - use specific branch, tag, or commit"
}

# Warn if no sync options configured
warn[msg] {
    require_sync_options
    app := input

    not app.spec.syncPolicy.syncOptions

    msg := "Application should configure syncOptions for better control (CreateNamespace, etc.)"
}

# Deny if using Replace sync option (dangerous)
deny[msg] {
    app := input
    sync_options := app.spec.syncPolicy.syncOptions[_]

    sync_options == "Replace=true"

    msg := "Application uses Replace=true sync option - can cause data loss"
}

# Warn if no retry configured
warn[msg] {
    app := input

    not app.spec.syncPolicy.retry

    msg := "Application should configure retry policy for transient failures"
}

# Deny if retry limit too high
deny[msg] {
    app := input
    retry := app.spec.syncPolicy.retry

    limit := retry.limit
    to_number(limit) > 10

    msg := sprintf(
        "Application retry limit %s is too high - max 10 retries recommended",
        [limit]
    )
}

# Deny if ignoreDifferences too broad
deny[msg] {
    app := input
    ignore := app.spec.ignoreDifferences[_]

    # Ignoring all fields
    ignore.jsonPointers
    pointer := ignore.jsonPointers[_]
    pointer == "/"

    msg := "Application ignoreDifferences with '/' ignores all fields - too broad"
}

# Warn if Application has no finalizers
warn[msg] {
    app := input

    is_production(app)
    not app.metadata.finalizers

    msg := "Production Application should have finalizers to prevent accidental deletion"
}

# Deny if Helm values from untrusted sources
deny[msg] {
    app := input
    source := app.spec.source
    helm := source.helm

    helm.valueFiles
    value_file := helm.valueFiles[_]

    # Check for URLs
    startswith(value_file, "http")

    msg := sprintf(
        "Application Helm valueFile '%s' from HTTP source - use Git repository",
        [value_file]
    )
}

# Recommend using ApplicationSet for multi-cluster
recommend[suggestion] {
    app := input

    # Has multiple similar apps (heuristic)
    name := app.metadata.name
    contains(name, "-")
    parts := split(name, "-")
    count(parts) > 2

    suggestion := {
        "recommendation": "Consider using ApplicationSet for managing multiple similar Applications",
        "reason": "Reduces duplication and improves maintainability"
    }
}

# Recommend using sync waves for dependencies
recommend[suggestion] {
    app := input

    # Has resources but no sync waves
    source := app.spec.source
    not has_sync_wave_annotation(app)

    suggestion := {
        "field": "metadata.annotations",
        "recommended": "argocd.argoproj.io/sync-wave",
        "reason": "Control deployment order with sync waves"
    }
}

# Helper functions
is_production(app) {
    namespace := app.spec.destination.namespace
    is_string(namespace)
    production_namespaces := ["production", "prod", "live"]
    namespace in production_namespaces
}

is_production(app) {
    name := app.metadata.name
    is_string(name)
    contains(name, "prod")
}

has_sync_wave_annotation(app) {
    annotations := app.metadata.annotations
    annotations["argocd.argoproj.io/sync-wave"]
}
