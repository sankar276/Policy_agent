package gitops.flux.gitrepository

import future.keywords.if
import future.keywords.in

# Configuration
default require_verification := true
default require_secret_ref := true
default allowed_git_providers := ["github.com", "gitlab.com", "bitbucket.org"]
default max_interval := 600  # 10 minutes
default min_interval := 60   # 1 minute

# Deny if GitRepository doesn't specify URL
deny[msg] {
    repo := input

    not repo.spec.url

    msg := "GitRepository must specify URL"
}

# Deny if using unverified Git provider
deny[msg] {
    repo := input
    url := repo.spec.url

    provider := extract_git_provider(url)
    provider != ""

    not provider in allowed_git_providers

    msg := sprintf(
        "GitRepository uses unverified provider '%s' - use approved providers only",
        [provider]
    )
}

# Deny if using HTTP instead of HTTPS or SSH
deny[msg] {
    repo := input
    url := repo.spec.url

    startswith(url, "http://")

    msg := sprintf(
        "GitRepository URL '%s' uses insecure HTTP - use HTTPS or SSH",
        [url]
    )
}

# Deny if no secret reference for private repos
deny[msg] {
    require_secret_ref
    repo := input

    is_private_repo(repo)
    not repo.spec.secretRef

    msg := "Private GitRepository should specify secretRef for authentication"
}

# Deny if verification not configured
deny[msg] {
    require_verification
    repo := input

    not repo.spec.verify

    msg := "GitRepository should enable verification to validate commit signatures"
}

# Deny if interval too frequent
deny[msg] {
    repo := input
    interval := repo.spec.interval

    is_string(interval)
    interval_seconds := parse_duration_to_seconds(interval)
    interval_seconds < min_interval

    msg := sprintf(
        "GitRepository interval '%s' is too frequent - minimum %d seconds",
        [interval, min_interval]
    )
}

# Warn if interval too long
warn[msg] {
    repo := input
    interval := repo.spec.interval

    is_string(interval)
    interval_seconds := parse_duration_to_seconds(interval)
    interval_seconds > max_interval

    msg := sprintf(
        "GitRepository interval '%s' is very long - changes won't sync quickly",
        [interval]
    )
}

# Deny if using default branch without explicit ref
warn[msg] {
    repo := input

    not repo.spec.ref

    msg := "GitRepository should specify explicit ref (branch, tag, or commit) for predictability"
}

# Deny if using mutable tag
warn[msg] {
    repo := input
    ref := repo.spec.ref

    ref.tag
    is_mutable_tag(ref.tag)

    msg := sprintf(
        "GitRepository tag '%s' appears mutable - use immutable tags or commit SHAs",
        [ref.tag]
    )
}

# Warn if no timeout configured
warn[msg] {
    repo := input

    not repo.spec.timeout

    msg := "GitRepository should set timeout to prevent hanging operations"
}

# Deny if ignore configuration is too broad
deny[msg] {
    repo := input
    ignore := repo.spec.ignore

    is_string(ignore)
    ignore == "*"

    msg := "GitRepository ignore pattern '*' excludes all files - too broad"
}

# Recommend using shallow clones for large repos
recommend[suggestion] {
    repo := input

    not repo.spec.gitImplementation

    suggestion := {
        "field": "spec.gitImplementation",
        "recommended": "go-git",
        "reason": "go-git implementation supports shallow clones for faster syncing"
    }
}

# Recommend using include for specific paths
recommend[suggestion] {
    repo := input

    not repo.spec.include
    not repo.spec.ignore

    suggestion := {
        "field": "spec.include",
        "recommended": "Specify paths to include",
        "reason": "Reduces sync scope and improves performance"
    }
}

# Helper functions
extract_git_provider(url) = provider {
    # Extract provider from URLs like https://github.com/org/repo
    contains(url, "github.com")
    provider := "github.com"
}

extract_git_provider(url) = provider {
    contains(url, "gitlab.com")
    provider := "gitlab.com"
}

extract_git_provider(url) = provider {
    contains(url, "bitbucket.org")
    provider := "bitbucket.org"
}

extract_git_provider(url) = "" {
    not contains(url, "github.com")
    not contains(url, "gitlab.com")
    not contains(url, "bitbucket.org")
}

is_private_repo(repo) {
    # Heuristic: private repos usually need secretRef
    url := repo.spec.url
    not contains(url, "public")
}

is_mutable_tag(tag) {
    # Tags like "latest", "main", "master" are mutable
    mutable_tags := ["latest", "main", "master", "develop", "dev"]
    tag in mutable_tags
}

parse_duration_to_seconds(duration) = seconds {
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
