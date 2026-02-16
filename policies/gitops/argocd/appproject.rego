package gitops.argocd.appproject

import future.keywords.if
import future.keywords.in

# Configuration
default require_source_repos := true
default require_destinations := true
default require_cluster_resource_whitelist := true
default allow_cluster_admin := false
default require_orphaned_resources := true

# Deny if AppProject has no source repositories
deny[msg] {
    require_source_repos
    project := input

    not project.spec.sourceRepos

    msg := "AppProject must specify allowed source repositories"
}

deny[msg] {
    require_source_repos
    project := input

    source_repos := project.spec.sourceRepos
    count(source_repos) == 0

    msg := "AppProject has no source repositories defined"
}

# Deny if using wildcard for all repositories
deny[msg] {
    project := input
    source_repos := project.spec.sourceRepos

    repo := source_repos[_]
    repo == "*"

    msg := "AppProject allows all repositories (*) - specify explicit repositories for security"
}

# Deny if no destinations specified
deny[msg] {
    require_destinations
    project := input

    not project.spec.destinations

    msg := "AppProject must specify allowed destinations"
}

deny[msg] {
    require_destinations
    project := input

    destinations := project.spec.destinations
    count(destinations) == 0

    msg := "AppProject has no destinations defined"
}

# Deny if allowing all namespaces
deny[msg] {
    project := input
    destinations := project.spec.destinations[_]

    destinations.namespace == "*"
    destinations.server != "https://kubernetes.default.svc"

    msg := "AppProject allows deployment to all namespaces (*) - specify explicit namespaces"
}

# Deny if no cluster resource whitelist for production
deny[msg] {
    require_cluster_resource_whitelist
    project := input

    is_production_project(project)
    not project.spec.clusterResourceWhitelist

    msg := "Production AppProject should define clusterResourceWhitelist"
}

# Deny if allowing all cluster resources
deny[msg] {
    not allow_cluster_admin
    project := input

    whitelist := project.spec.clusterResourceWhitelist[_]
    whitelist.group == "*"
    whitelist.kind == "*"

    msg := "AppProject allows all cluster resources (*/*) - specify explicit resources"
}

# Warn if no orphaned resources policy
warn[msg] {
    require_orphaned_resources
    project := input

    not project.spec.orphanedResources

    msg := "AppProject should configure orphanedResources policy to detect orphaned resources"
}

# Warn if orphaned resources not set to warn/error
warn[msg] {
    project := input
    orphaned := project.spec.orphanedResources

    orphaned.warn == false

    msg := "AppProject orphanedResources.warn is disabled - won't alert on orphaned resources"
}

# Deny if no namespace resource blacklist for sensitive resources
deny[msg] {
    project := input

    is_production_project(project)
    not has_namespace_resource_blacklist(project)

    msg := "Production AppProject should blacklist sensitive namespace resources (Secrets, etc.)"
}

# Deny if allowing direct Secret management
warn[msg] {
    project := input

    not blocks_secrets(project)

    msg := "AppProject should consider blacklisting direct Secret management - use sealed-secrets or external-secrets"
}

# Deny if sync windows allow unrestricted syncing
warn[msg] {
    project := input

    is_production_project(project)
    not project.spec.syncWindows

    msg := "Production AppProject should configure syncWindows for change control"
}

# Warn if no signature keys configured
warn[msg] {
    project := input

    is_production_project(project)
    not project.spec.signatureKeys

    msg := "Production AppProject should configure signatureKeys for commit verification"
}

# Deny if no RBAC policies defined
deny[msg] {
    project := input

    is_production_project(project)
    not project.spec.roles

    msg := "Production AppProject should define RBAC roles for access control"
}

# Warn if roles have overly broad permissions
warn[msg] {
    project := input
    role := project.spec.roles[_]

    policy := role.policies[_]
    contains(policy, "*")

    msg := sprintf(
        "AppProject role '%s' has wildcard permissions - use specific actions",
        [role.name]
    )
}

# Deny if default project used for production
deny[msg] {
    project := input

    project.metadata.name == "default"
    is_production_project(project)

    msg := "Production applications should use dedicated AppProject (not 'default')"
}

# Recommend using namespace resource whitelist
recommend[suggestion] {
    project := input

    not project.spec.namespaceResourceWhitelist

    suggestion := {
        "field": "spec.namespaceResourceWhitelist",
        "recommended": "Define allowed namespace resources",
        "reason": "Restricts what resources can be deployed for better security"
    }
}

# Recommend configuring permitted clusters
recommend[suggestion] {
    project := input

    destinations := project.spec.destinations
    all_clusters := [dest | dest := destinations[_]; dest.server == "*"]

    count(all_clusters) > 0

    suggestion := {
        "field": "spec.destinations",
        "recommended": "Specify explicit cluster servers",
        "reason": "Prevents deployment to unintended clusters"
    }
}

# Helper functions
is_production_project(project) {
    name := project.metadata.name
    is_string(name)
    production_indicators := ["prod", "production", "live"]
    indicator := production_indicators[_]
    contains(lower(name), indicator)
}

is_production_project(project) {
    namespace := project.metadata.namespace
    is_string(namespace)
    production_namespaces := ["argocd-prod", "gitops-prod"]
    namespace in production_namespaces
}

has_namespace_resource_blacklist(project) {
    project.spec.namespaceResourceBlacklist
    count(project.spec.namespaceResourceBlacklist) > 0
}

blocks_secrets(project) {
    blacklist := project.spec.namespaceResourceBlacklist[_]
    blacklist.kind == "Secret"
}
