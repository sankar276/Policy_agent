package cicd.github.workflows

import future.keywords.if
import future.keywords.in

# Configuration
default require_tests := true
default require_build_artifacts := false
default max_job_timeout := 60  # minutes
default require_concurrency_control := true

# Deny if workflow has no name
deny[msg] {
    workflow := input
    not workflow.name

    msg := "Workflow must have a descriptive name"
}

# Deny if job timeout is too long or missing
deny[msg] {
    job := input.jobs[job_name]

    timeout := job["timeout-minutes"]
    timeout > max_job_timeout

    msg := sprintf(
        "Job '%s' timeout of %d minutes exceeds maximum %d minutes",
        [job_name, timeout, max_job_timeout]
    )
}

warn[msg] {
    job := input.jobs[job_name]
    not job["timeout-minutes"]

    msg := sprintf(
        "Job '%s' should set timeout-minutes to prevent hanging builds",
        [job_name]
    )
}

# Deny if no test job exists
deny[msg] {
    require_tests
    workflow := input

    not has_test_job(workflow)

    msg := "Workflow should include a test job to validate code quality"
}

# Deny if building on main/master without protection
deny[msg] {
    workflow := input

    trigger_branches := get_trigger_branches(workflow)
    protected_branch := trigger_branches[_]
    protected_branch in ["main", "master", "production"]

    not has_required_checks(workflow)

    msg := sprintf(
        "Workflow triggered on protected branch '%s' should have required status checks",
        [protected_branch]
    )
}

# Warn if no concurrency control on deployment jobs
warn[msg] {
    require_concurrency_control
    job := input.jobs[job_name]

    is_deployment_job(job)
    not input.concurrency

    msg := sprintf(
        "Deployment job '%s' should use concurrency control to prevent simultaneous deployments",
        [job_name]
    )
}

# Deny if secrets used in conditions
deny[msg] {
    job := input.jobs[job_name]

    if_condition := job["if"]
    if_condition

    contains(if_condition, "secrets.")

    msg := sprintf(
        "Job '%s' uses secrets in 'if' condition - secrets are not masked in conditions",
        [job_name]
    )
}

# Warn if matrix builds don't use fail-fast
warn[msg] {
    job := input.jobs[job_name]
    strategy := job.strategy

    strategy.matrix
    not strategy["fail-fast"]

    msg := sprintf(
        "Job '%s' with matrix strategy should set fail-fast to avoid wasting resources",
        [job_name]
    )
}

# Deny if environment variables contain sensitive data patterns
deny[msg] {
    job := input.jobs[job_name]
    env := job.env[key]

    is_string(env)
    sensitive_pattern := ["password", "token", "key", "secret"]
    pattern := sensitive_pattern[_]

    contains(lower(key), pattern)
    not contains(env, "secrets.")
    not contains(env, "vars.")

    msg := sprintf(
        "Job '%s' environment variable '%s' appears to contain sensitive data - use secrets",
        [job_name, key]
    )
}

# Warn if using deprecated actions
warn[msg] {
    job := input.jobs[job_name]
    step := job.steps[_]

    deprecated_actions := [
        "actions/setup-node@v1",
        "actions/checkout@v1",
        "actions/cache@v1",
    ]

    action := deprecated_actions[_]
    step.uses == action

    msg := sprintf(
        "Job '%s' uses deprecated action '%s' - upgrade to latest version",
        [job_name, action]
    )
}

# Recommend caching for common package managers
recommend[suggestion] {
    job := input.jobs[job_name]
    step := job.steps[_]

    # Has setup-node but no cache
    contains(step.uses, "actions/setup-node")
    not has_cache_step(job)

    suggestion := {
        "job": job_name,
        "field": "steps",
        "recommended": "Add caching step for node_modules",
        "reason": "Speeds up builds and reduces network usage"
    }
}

# Helper functions
has_test_job(workflow) {
    job := workflow.jobs[_]
    test_indicators := ["test", "lint", "check"]
    indicator := test_indicators[_]
    contains(lower(job.name), indicator)
}

get_trigger_branches(workflow) = branches {
    branches := [branch |
        branch_config := workflow.on.push.branches[_]
        branch := branch_config
    ]
}

get_trigger_branches(workflow) = branches {
    branches := [branch |
        branch_config := workflow.on.pull_request.branches[_]
        branch := branch_config
    ]
}

get_trigger_branches(workflow) = [] {
    not workflow.on.push
    not workflow.on.pull_request
}

has_required_checks(workflow) {
    # This would check for required status checks in branch protection
    # Simplified for now
    true
}

is_deployment_job(job) {
    deployment_keywords := ["deploy", "release", "publish"]
    keyword := deployment_keywords[_]
    contains(lower(job.name), keyword)
}

has_cache_step(job) {
    step := job.steps[_]
    contains(step.uses, "actions/cache")
}
