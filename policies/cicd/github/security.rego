package cicd.github.security

import future.keywords.if
import future.keywords.in

# Configuration
default require_security_scanning := true
default require_secret_scanning := true
default allowed_actions := ["actions/*", "github/*"]
default require_approval := true

# Deny if workflow runs on pull_request without approval
deny[msg] {
    require_approval
    workflow := input
    trigger := workflow.on.pull_request
    trigger

    not has_approval_job(workflow)

    msg := "Workflow triggered by pull_request should require approval for external contributors"
}

# Deny if using pull_request_target without proper security
deny[msg] {
    workflow := input
    workflow.on.pull_request_target

    not has_permission_restrictions(workflow)

    msg := "Workflow using pull_request_target must restrict permissions - security risk for PR from forks"
}

# Deny if workflow has write permissions without justification
deny[msg] {
    job := input.jobs[job_name]
    permissions := job.permissions

    permissions.contents == "write"
    permissions.pull_requests == "write"

    msg := sprintf(
        "Job '%s' has broad write permissions - use minimal required permissions",
        [job_name]
    )
}

# Deny if using third-party actions without version pinning
deny[msg] {
    job := input.jobs[job_name]
    step := job.steps[_]

    action := step.uses
    action

    # Not an official action
    not starts_with_allowed_action(action)

    # Not pinned to SHA
    not regex.match(`@[a-f0-9]{40}$`, action)

    msg := sprintf(
        "Job '%s' uses unpinned third-party action '%s' - pin to commit SHA for security",
        [job_name, action]
    )
}

# Deny if secrets are exposed in logs
deny[msg] {
    job := input.jobs[job_name]
    step := job.steps[_]

    # Check for potential secret exposure
    run := step.run
    contains(run, "echo ${{")
    contains(run, "secrets.")

    msg := sprintf(
        "Job '%s' may expose secrets in logs - avoid echoing secret values",
        [job_name]
    )
}

# Deny if using self-hosted runners without security controls
deny[msg] {
    job := input.jobs[job_name]
    runs_on := job["runs-on"]

    is_string(runs_on)
    runs_on == "self-hosted"

    not has_security_controls(job)

    msg := sprintf(
        "Job '%s' uses self-hosted runner without security controls",
        [job_name]
    )
}

# Warn if no security scanning step
warn[msg] {
    require_security_scanning
    workflow := input

    not has_security_scanning(workflow)

    msg := "Workflow should include security scanning (CodeQL, dependency scanning, or SAST)"
}

# Warn if artifacts uploaded without retention limit
warn[msg] {
    job := input.jobs[job_name]
    step := job.steps[_]

    step.uses
    contains(step.uses, "actions/upload-artifact")

    not step.with["retention-days"]

    msg := sprintf(
        "Job '%s' uploads artifacts without retention limit - set retention-days to manage costs",
        [job_name]
    )
}

# Helper functions
has_approval_job(workflow) {
    job := workflow.jobs[_]
    job.environment
}

has_permission_restrictions(workflow) {
    workflow.permissions
    workflow.permissions.contents == "read"
}

starts_with_allowed_action(action) {
    allowed := allowed_actions[_]
    prefix := trim_suffix(allowed, "*")
    startswith(action, prefix)
}

has_security_controls(job) {
    job.environment
}

has_security_scanning(workflow) {
    job := workflow.jobs[_]
    step := job.steps[_]

    security_actions := [
        "github/codeql-action",
        "aquasecurity/trivy-action",
        "snyk/actions",
    ]

    action := security_actions[_]
    contains(step.uses, action)
}
