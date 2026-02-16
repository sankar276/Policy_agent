package kafka.topics.retention

import future.keywords.if

# Configuration (in milliseconds for calculations)
default max_retention_days := 90
default warn_retention_days := 60
default min_retention_days := 1
default ms_per_day := 86400000  # 1000 * 60 * 60 * 24

# Deny if retention exceeds maximum allowed
deny[msg] {
    input.kind == "KafkaTopic"
    retention_ms := to_number(input.spec.config["retention.ms"])
    retention_days := retention_ms / ms_per_day
    retention_days > max_retention_days
    msg := sprintf(
        "Topic '%s' retention %.1f days exceeds maximum %d days",
        [input.metadata.name, retention_days, max_retention_days]
    )
}

# Deny if retention is too short (data loss risk)
deny[msg] {
    input.kind == "KafkaTopic"
    retention_ms := to_number(input.spec.config["retention.ms"])
    retention_days := retention_ms / ms_per_day
    retention_days < min_retention_days
    msg := sprintf(
        "Topic '%s' retention %.1f days is too short (minimum: %d days)",
        [input.metadata.name, retention_days, min_retention_days]
    )
}

# Deny if retention.ms is missing
deny[msg] {
    input.kind == "KafkaTopic"
    not input.spec.config["retention.ms"]
    msg := sprintf(
        "Topic '%s' missing retention.ms configuration",
        [input.metadata.name]
    )
}

# Warning for high retention
warn[msg] {
    input.kind == "KafkaTopic"
    retention_ms := to_number(input.spec.config["retention.ms"])
    retention_days := retention_ms / ms_per_day
    retention_days > warn_retention_days
    retention_days <= max_retention_days
    msg := sprintf(
        "Topic '%s' retention %.1f days is high (consider reviewing storage costs)",
        [input.metadata.name, retention_days]
    )
}

# Warning if both retention.ms and retention.bytes are very high
warn[msg] {
    input.kind == "KafkaTopic"
    retention_ms := to_number(input.spec.config["retention.ms"])
    retention_bytes := to_number(input.spec.config["retention.bytes"])
    retention_days := retention_ms / ms_per_day
    retention_gb := retention_bytes / (1024 * 1024 * 1024)
    retention_days > warn_retention_days
    retention_gb > 100
    msg := sprintf(
        "Topic '%s' has both high time retention (%.1f days) and size retention (%.1f GB)",
        [input.metadata.name, retention_days, retention_gb]
    )
}

# Recommendations
recommend[suggestion] {
    input.kind == "KafkaTopic"
    not input.spec.config["retention.ms"]
    default_retention_days := 7
    suggestion := {
        "field": "spec.config.retention.ms",
        "current": null,
        "recommended": sprintf("%d", [default_retention_days * ms_per_day]),
        "reason": "Explicitly set retention period to avoid relying on broker defaults"
    }
}

recommend[suggestion] {
    input.kind == "KafkaTopic"
    retention_ms := to_number(input.spec.config["retention.ms"])
    retention_days := retention_ms / ms_per_day
    retention_days > max_retention_days
    suggestion := {
        "field": "spec.config.retention.ms",
        "current": retention_ms,
        "recommended": sprintf("%d", [max_retention_days * ms_per_day]),
        "reason": sprintf("Reduce retention to maximum allowed %d days to optimize storage", [max_retention_days])
    }
}

# Recommend setting retention.bytes as a safety measure
recommend[suggestion] {
    input.kind == "KafkaTopic"
    not input.spec.config["retention.bytes"]
    msg := "Consider setting retention.bytes as a safety limit to prevent unbounded storage growth"
    suggestion := {
        "field": "spec.config.retention.bytes",
        "current": null,
        "recommended": "-1",
        "reason": msg
    }
}
