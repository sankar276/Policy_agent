package kafka.topics.replication

import future.keywords.if
import future.keywords.in

# Configuration - these can be overridden via data
default min_replication_factor := 3
default require_min_insync_replicas := true
default min_insync_replicas_value := 2

# Deny if replication factor is too low
deny[msg] {
    input.kind == "KafkaTopic"
    rf := input.spec.replicas
    rf < min_replication_factor
    msg := sprintf(
        "Topic '%s' has insufficient replication factor %d (minimum: %d)",
        [input.metadata.name, rf, min_replication_factor]
    )
}

# Deny if min.insync.replicas is not configured
deny[msg] {
    input.kind == "KafkaTopic"
    require_min_insync_replicas
    not input.spec.config["min.insync.replicas"]
    msg := sprintf(
        "Topic '%s' missing min.insync.replicas configuration",
        [input.metadata.name]
    )
}

# Deny if min.insync.replicas is too low
deny[msg] {
    input.kind == "KafkaTopic"
    require_min_insync_replicas
    isr := to_number(input.spec.config["min.insync.replicas"])
    isr < min_insync_replicas_value
    msg := sprintf(
        "Topic '%s' has min.insync.replicas %d (minimum: %d)",
        [input.metadata.name, isr, min_insync_replicas_value]
    )
}

# Deny if min.insync.replicas >= replication factor (invalid configuration)
deny[msg] {
    input.kind == "KafkaTopic"
    rf := input.spec.replicas
    isr := to_number(input.spec.config["min.insync.replicas"])
    isr >= rf
    msg := sprintf(
        "Topic '%s' has min.insync.replicas (%d) >= replication factor (%d), which prevents writes",
        [input.metadata.name, isr, rf]
    )
}

# Warning for single replication (data loss risk)
warn[msg] {
    input.kind == "KafkaTopic"
    rf := input.spec.replicas
    rf == 1
    msg := sprintf(
        "Topic '%s' has no replication (RF=1), significant data loss risk",
        [input.metadata.name]
    )
}

# Recommendations for fixing replication issues
recommend[suggestion] {
    input.kind == "KafkaTopic"
    rf := input.spec.replicas
    rf < min_replication_factor
    suggestion := {
        "field": "spec.replicas",
        "current": rf,
        "recommended": min_replication_factor,
        "reason": "Ensures data durability and high availability across multiple brokers"
    }
}

recommend[suggestion] {
    input.kind == "KafkaTopic"
    not input.spec.config["min.insync.replicas"]
    suggestion := {
        "field": "spec.config.min.insync.replicas",
        "current": null,
        "recommended": sprintf("%d", [min_insync_replicas_value]),
        "reason": "Guarantees that writes are acknowledged by at least this many replicas"
    }
}
