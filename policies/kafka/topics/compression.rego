package kafka.topics.compression

import future.keywords.if
import future.keywords.in

# Configuration
default require_compression := true
default allowed_compression_types := ["lz4", "snappy", "zstd", "gzip"]
default recommended_compression := "lz4"

# Deny if compression is required but not configured
deny[msg] {
    input.kind == "KafkaTopic"
    require_compression
    not input.spec.config["compression.type"]
    msg := sprintf(
        "Topic '%s' missing compression configuration",
        [input.metadata.name]
    )
}

# Deny if compression type is 'none' or 'uncompressed'
deny[msg] {
    input.kind == "KafkaTopic"
    require_compression
    compression := input.spec.config["compression.type"]
    compression in ["none", "uncompressed"]
    msg := sprintf(
        "Topic '%s' has compression disabled (compression is required)",
        [input.metadata.name]
    )
}

# Deny if compression type is not in allowed list
deny[msg] {
    input.kind == "KafkaTopic"
    compression := input.spec.config["compression.type"]
    not compression in allowed_compression_types
    msg := sprintf(
        "Topic '%s' uses unsupported compression '%s' (allowed: %v)",
        [input.metadata.name, compression, allowed_compression_types]
    )
}

# Warning for suboptimal compression choice
warn[msg] {
    input.kind == "KafkaTopic"
    compression := input.spec.config["compression.type"]
    compression == "gzip"
    msg := sprintf(
        "Topic '%s' uses 'gzip' compression which has higher CPU overhead (consider 'lz4' or 'zstd')",
        [input.metadata.name]
    )
}

# Recommendation to enable compression
recommend[suggestion] {
    input.kind == "KafkaTopic"
    not input.spec.config["compression.type"]
    suggestion := {
        "field": "spec.config.compression.type",
        "current": null,
        "recommended": recommended_compression,
        "reason": "Reduces network bandwidth and storage costs with minimal CPU overhead"
    }
}

# Recommendation for better compression algorithm
recommend[suggestion] {
    input.kind == "KafkaTopic"
    compression := input.spec.config["compression.type"]
    compression == "gzip"
    suggestion := {
        "field": "spec.config.compression.type",
        "current": compression,
        "recommended": "lz4",
        "reason": "LZ4 offers better compression speed and lower CPU usage than gzip"
    }
}
