package cicd.common.bestpractices

import future.keywords.if
import future.keywords.in

# Configuration
default require_stages := true
default require_testing := true
default require_linting := true
default allowed_registries := ["docker.io", "ghcr.io", "gcr.io"]

# Deny if pipeline has no defined stages
deny[msg] {
    require_stages
    pipeline := input

    not has_stages(pipeline)

    msg := "Pipeline should define explicit stages (build, test, deploy)"
}

# Deny if no testing stage
deny[msg] {
    require_testing
    pipeline := input

    not has_testing_stage(pipeline)

    msg := "Pipeline must include testing stage to validate code quality"
}

# Warn if no linting/code quality checks
warn[msg] {
    require_linting
    pipeline := input

    not has_linting_stage(pipeline)

    msg := "Pipeline should include linting or code quality checks"
}

# Deny if using unverified container registries
deny[msg] {
    job := input[job_name]
    image := get_image(job)

    registry := extract_registry(image)
    registry != ""

    not registry in allowed_registries

    msg := sprintf(
        "Job '%s' uses image from unverified registry '%s' - use approved registries only",
        [job_name, registry]
    )
}

# Deny if deployment has no health checks
deny[msg] {
    job := input[job_name]

    is_deployment_job(job)
    not has_health_check(job)

    msg := sprintf(
        "Deployment job '%s' should include health checks to verify deployment",
        [job_name]
    )
}

# Warn if builds don't fail fast on errors
warn[msg] {
    job := input[job_name]
    scripts := get_scripts(job)

    not contains_error_handling(scripts)

    msg := sprintf(
        "Job '%s' should use 'set -e' or equivalent to fail fast on errors",
        [job_name]
    )
}

# Deny if deployment to production without backup/rollback
deny[msg] {
    job := input[job_name]

    is_production_deployment(job)
    not has_rollback_strategy(job)

    msg := sprintf(
        "Production deployment job '%s' should have rollback strategy",
        [job_name]
    )
}

# Warn if no artifact versioning
warn[msg] {
    job := input[job_name]

    builds_artifact(job)
    not uses_versioning(job)

    msg := sprintf(
        "Job '%s' builds artifacts but doesn't use versioning - use semantic versioning",
        [job_name]
    )
}

# Deny if parallel jobs without resource limits
deny[msg] {
    job := input[job_name]

    runs_parallel(job)
    not has_resource_limits(job)

    msg := sprintf(
        "Parallel job '%s' should define resource limits to prevent overload",
        [job_name]
    )
}

# Recommend using matrix builds for multi-environment testing
recommend[suggestion] {
    job := input[job_name]

    is_test_job(job)
    not uses_matrix(job)

    suggestion := {
        "job": job_name,
        "recommendation": "Use matrix/parallel strategy for testing multiple versions",
        "reason": "Test against multiple Node/Python/etc versions efficiently"
    }
}

# Helper functions
has_stages(pipeline) {
    # GitHub Actions
    pipeline.jobs
}

has_stages(pipeline) {
    # GitLab CI
    pipeline.stages
}

has_testing_stage(pipeline) {
    job := pipeline[_]
    test_keywords := ["test", "unittest", "integration", "e2e"]
    keyword := test_keywords[_]
    contains(lower(job_name_or_name(job)), keyword)
}

has_linting_stage(pipeline) {
    job := pipeline[_]
    lint_keywords := ["lint", "format", "style", "quality"]
    keyword := lint_keywords[_]
    contains(lower(job_name_or_name(job)), keyword)
}

job_name_or_name(job) = name {
    name := job.name
}

job_name_or_name(job) = name {
    not job.name
    # Use a default or job key
    name := "unknown"
}

get_image(job) = image {
    image := job.image
    is_string(image)
}

get_image(job) = image {
    image := job.image.name
    is_string(image)
}

get_image(job) = "" {
    not job.image
}

extract_registry(image) = registry {
    parts := split(image, "/")
    count(parts) > 1
    registry := parts[0]
    contains(registry, ".")
}

extract_registry(image) = "" {
    parts := split(image, "/")
    count(parts) <= 1
}

is_deployment_job(job) {
    deploy_keywords := ["deploy", "release", "publish", "rollout"]
    keyword := deploy_keywords[_]
    name := job_name_or_name(job)
    contains(lower(name), keyword)
}

is_production_deployment(job) {
    is_deployment_job(job)

    # Check environment
    env := get_environment(job)
    contains(lower(env), "prod")
}

get_environment(job) = env {
    env := job.environment
    is_string(env)
}

get_environment(job) = env {
    env := job.environment.name
    is_string(env)
}

get_environment(job) = "" {
    not job.environment
}

has_health_check(job) {
    scripts := get_scripts(job)
    script := scripts[_]
    health_check_keywords := ["curl", "health", "ready", "probe"]
    keyword := health_check_keywords[_]
    contains(script, keyword)
}

has_rollback_strategy(job) {
    scripts := get_scripts(job)
    script := scripts[_]
    rollback_keywords := ["rollback", "revert", "previous"]
    keyword := rollback_keywords[_]
    contains(script, keyword)
}

get_scripts(job) = scripts {
    scripts := job.script
    is_array(scripts)
}

get_scripts(job) = scripts {
    scripts := job.run
    is_string(scripts)
    scripts := [scripts]
}

get_scripts(job) = scripts {
    steps := job.steps
    scripts := [step.run | step := steps[_]; step.run]
}

get_scripts(job) = [] {
    not job.script
    not job.run
    not job.steps
}

contains_error_handling(scripts) {
    script := scripts[_]
    error_handling := ["set -e", "set -o pipefail", "exit 1"]
    handler := error_handling[_]
    contains(script, handler)
}

builds_artifact(job) {
    # GitHub Actions
    job.steps
    step := job.steps[_]
    contains(step.uses, "upload-artifact")
}

builds_artifact(job) {
    # GitLab CI
    job.artifacts
}

uses_versioning(job) {
    scripts := get_scripts(job)
    script := scripts[_]
    versioning_keywords := ["version", "tag", "semver", "$CI_COMMIT"]
    keyword := versioning_keywords[_]
    contains(script, keyword)
}

runs_parallel(job) {
    job.strategy
    job.strategy.matrix
}

runs_parallel(job) {
    job.parallel
}

has_resource_limits(job) {
    job.resources
}

has_resource_limits(job) {
    job["timeout-minutes"]
}

is_test_job(job) {
    name := job_name_or_name(job)
    test_keywords := ["test", "spec", "e2e"]
    keyword := test_keywords[_]
    contains(lower(name), keyword)
}

uses_matrix(job) {
    job.strategy
    job.strategy.matrix
}

uses_matrix(job) {
    job.parallel
    job.parallel.matrix
}
