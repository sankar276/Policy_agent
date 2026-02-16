package cicd.gitlab.security

import future.keywords.if
import future.keywords.in

# Configuration
default require_security_scanning := true
default require_sast := true
default require_dependency_scanning := true
default allow_docker_privileged := false

# Deny if using privileged Docker mode
deny[msg] {
    not allow_docker_privileged

    job := input[job_name]
    job.services

    service := job.services[_]
    service.command
    contains(service.command[_], "--privileged")

    msg := sprintf(
        "Job '%s' uses privileged Docker mode - security risk",
        [job_name]
    )
}

# Deny if variables contain potential secrets
deny[msg] {
    job := input[job_name]
    variables := job.variables

    var_name := variables[_]
    is_string(var_name)

    sensitive_keywords := ["PASSWORD", "TOKEN", "SECRET", "KEY"]
    keyword := sensitive_keywords[_]
    contains(var_name, keyword)

    not contains(var_name, "$")  # Not a reference

    msg := sprintf(
        "Job '%s' variable '%s' may contain hardcoded secrets - use CI/CD variables",
        [job_name, var_name]
    )
}

# Deny if script uses curl/wget without verification
deny[msg] {
    job := input[job_name]
    script := job.script[_]

    contains(script, "curl")
    contains(script, "| sh")
    not contains(script, "--fail")

    msg := sprintf(
        "Job '%s' pipes curl to shell without error checking - security risk",
        [job_name]
    )
}

deny[msg] {
    job := input[job_name]
    script := job.script[_]

    contains(script, "wget")
    contains(script, "| sh")

    msg := sprintf(
        "Job '%s' pipes wget to shell - security risk",
        [job_name]
    )
}

# Deny if using latest tag for Docker images
deny[msg] {
    job := input[job_name]
    image := job.image

    is_string(image)
    endswith(image, ":latest")

    msg := sprintf(
        "Job '%s' uses ':latest' Docker image tag - pin to specific version",
        [job_name]
    )
}

# Deny if artifacts exposed without expiration
deny[msg] {
    job := input[job_name]
    artifacts := job.artifacts

    artifacts.paths
    not artifacts.expire_in

    msg := sprintf(
        "Job '%s' artifacts should have expire_in set to manage storage",
        [job_name]
    )
}

# Warn if no SAST job
warn[msg] {
    require_sast
    pipeline := input

    not has_sast_job(pipeline)

    msg := "Pipeline should include SAST (Static Application Security Testing)"
}

# Warn if no dependency scanning
warn[msg] {
    require_dependency_scanning
    pipeline := input

    not has_dependency_scanning(pipeline)

    msg := "Pipeline should include dependency scanning for vulnerable packages"
}

# Deny if deploying to production without manual approval
deny[msg] {
    job := input[job_name]

    is_production_deploy(job)
    not job.when == "manual"

    msg := sprintf(
        "Production deployment job '%s' should require manual approval",
        [job_name]
    )
}

# Deny if secrets used in before_script
deny[msg] {
    job := input[job_name]
    before_script := job.before_script[_]

    contains(before_script, "echo")
    contains(before_script, "$CI_")
    contains(before_script, "TOKEN")

    msg := sprintf(
        "Job '%s' may expose secrets in before_script",
        [job_name]
    )
}

# Warn if cache not used for package managers
warn[msg] {
    job := input[job_name]
    script := job.script[_]

    package_manager_commands := ["npm install", "pip install", "bundle install"]
    command := package_manager_commands[_]
    contains(script, command)

    not job.cache

    msg := sprintf(
        "Job '%s' runs package manager but doesn't use cache - add caching to speed up builds",
        [job_name]
    )
}

# Helper functions
has_sast_job(pipeline) {
    job := pipeline[job_name]
    sast_indicators := ["sast", "security-scan", "code-quality"]
    indicator := sast_indicators[_]
    contains(lower(job_name), indicator)
}

has_dependency_scanning(pipeline) {
    job := pipeline[job_name]
    dependency_indicators := ["dependency", "gemnasium", "retire"]
    indicator := dependency_indicators[_]
    contains(lower(job_name), indicator)
}

is_production_deploy(job) {
    environment := job.environment
    is_string(environment)
    contains(lower(environment), "prod")
}

is_production_deploy(job) {
    environment := job.environment.name
    is_string(environment)
    contains(lower(environment), "prod")
}
