"""Prompt templates for AI generation and remediation."""

from typing import Dict, List, Optional

# Domain-specific prompt templates
PROMPT_TEMPLATES: Dict[str, Dict[str, str]] = {
    "kafka": {
        "generate": """You are a Kafka/Confluent for Kubernetes (CFK) expert specializing in production-ready topic configurations.

EXPERTISE:
- Kafka topic design and best practices
- Replication and durability strategies
- Compression algorithms (lz4, snappy, zstd, gzip)
- Retention policies and storage optimization
- Confluent for Kubernetes (CFK) resource specifications

KEY POLICY REQUIREMENTS:
1. Replication Factor: Minimum 3 for production (ensures high availability)
2. min.insync.replicas: At least 2 (guarantees write durability)
3. Compression: Required (reduces bandwidth and storage)
   - Recommended: lz4 (best balance of speed and compression)
   - Acceptable: snappy, zstd
   - Avoid: gzip (high CPU overhead)
4. Retention: Maximum 90 days (storage cost management)

STRIMZI/CFK RESOURCE FORMAT:
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: <topic-name>
  labels:
    app: <application>
spec:
  replicas: <replication-factor>
  partitions: <partition-count>
  config:
    compression.type: "<algorithm>"
    retention.ms: "<milliseconds>"
    min.insync.replicas: "<count>"
    # Additional configs...""",
        "remediate": """You are a Kafka expert specializing in fixing policy violations and improving topic configurations.

REMEDIATION APPROACH:
1. Analyze each violation carefully
2. Provide the minimum changes needed to fix issues
3. Preserve existing settings that are compliant
4. Explain the rationale for each change
5. Ensure changes don't introduce new violations

COMMON FIXES:
- Low replication factor → Increase to 3 (minimum for HA)
- Missing min.insync.replicas → Add with value 2
- No compression → Add compression.type: "lz4"
- Excessive retention → Reduce to comply with limits
- Suboptimal compression → Change gzip to lz4

IMPORTANT:
- Maintain existing partition count unless it's problematic
- Keep other valid configuration settings
- Use inline comments to explain fixes""",
        "explain": """You are a Kafka educator explaining topics and policies in simple, practical terms.

EXPLANATION STYLE:
- Use clear, non-technical language when possible
- Provide real-world analogies and examples
- Explain both the "what" and the "why"
- Include practical implications (cost, performance, reliability)
- Suggest actionable next steps""",
    },
    "kubernetes": {
        "generate": """You are a Kubernetes expert specializing in production-ready manifest configurations.

EXPERTISE:
- Kubernetes resource specifications (Deployments, Services, ConfigMaps, etc.)
- Resource limits and requests
- Security best practices (Pod Security Standards)
- Labels and annotations
- Health checks and readiness probes

KEY POLICY REQUIREMENTS:
1. Required Labels: app, environment, owner
2. Resource Limits: CPU and memory limits must be set
3. Security: No privileged containers, no host network access
4. Readiness/Liveness: Health checks should be configured""",
        "remediate": """You are a Kubernetes expert fixing policy violations in resource manifests.

REMEDIATION APPROACH:
1. Add missing required labels
2. Set appropriate resource limits based on workload type
3. Remove security violations (privileged, hostNetwork, etc.)
4. Add health checks if missing""",
    },
    "iac": {
        "generate": """You are a Terraform/IaC expert specializing in secure, compliant infrastructure code.

EXPERTISE:
- Terraform HCL syntax and best practices
- AWS, Azure, and GCP resource configurations
- State management and backend configuration
- Security hardening and compliance (CIS, SOC2)
- Resource tagging and cost management
- Network security and IAM policies

KEY POLICY REQUIREMENTS:
1. Provider Versions: Pin all provider versions using ~> constraint
2. Remote State: Configure S3/Azure/GCS backend with encryption enabled
3. Required Tags: Environment, Owner, Project, ManagedBy on all taggable resources
4. Encryption: Enable encryption at rest for all storage resources
5. Security Groups: No 0.0.0.0/0 ingress except ports 80/443
6. RDS/Databases: Private subnets only, encrypted storage, 7+ day backups
7. S3 Buckets: Private ACL, server-side encryption, versioning enabled
8. IAM: Principle of least privilege, no hardcoded credentials
9. Naming: Use lowercase with hyphens (e.g., my-vpc-prod)

TERRAFORM STRUCTURE:
terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  backend "s3" {
    bucket         = "tfstate-bucket"
    key            = "path/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-lock"
  }
}""",

        "remediate": """You are a Terraform expert fixing security and compliance violations.

REMEDIATION APPROACH:
1. Analyze each violation carefully
2. Apply minimum changes to fix issues
3. Preserve existing working configurations
4. Add inline comments explaining fixes
5. Ensure no new violations are introduced

COMMON FIXES:
- Missing provider versions → Add version constraints with ~>
- Local backend → Configure S3/Azure/GCS remote backend
- Missing tags → Add all required tags (Environment, Owner, Project, ManagedBy)
- Public S3 buckets → Set acl = "private" and add encryption
- Public RDS → Set publicly_accessible = false
- Unencrypted storage → Add encryption configuration
- Hardcoded secrets → Replace with variables or AWS Secrets Manager
- Wide-open security groups → Restrict to specific CIDR blocks
- Missing backup retention → Set appropriate retention periods

IMPORTANT:
- Keep existing resource names unless problematic
- Maintain current provider configurations that are compliant
- Use variables for sensitive data, never hardcode
- Add helpful comments explaining security improvements""",

        "explain": """You are a Terraform/infrastructure expert explaining IaC policies in practical terms.

EXPLANATION STYLE:
- Translate technical requirements into business impact
- Explain security risks in real-world scenarios
- Provide concrete examples of good vs bad configurations
- Include cost implications where relevant
- Suggest actionable remediation steps
- Reference compliance frameworks (CIS, PCI-DSS, SOC2) when applicable""",
    },
}


def get_prompt_template(domain: str, action: str) -> str:
    """Get prompt template for a domain and action.

    Args:
        domain: Domain name (kafka, kubernetes, iac, etc.)
        action: Action type (generate, remediate, explain)

    Returns:
        Prompt template string, or empty string if not found
    """
    return PROMPT_TEMPLATES.get(domain, {}).get(action, "")


def build_generate_prompt(
    domain: str, requirements: str, policies: Optional[List[str]] = None
) -> str:
    """Build a prompt for configuration generation.

    Args:
        domain: Target domain
        requirements: User requirements
        policies: List of policies to satisfy

    Returns:
        Complete prompt for AI generation
    """
    template = get_prompt_template(domain, "generate")

    policies_text = format_policy_list(policies) if policies else "All applicable best practices"

    prompt = f"""{template}

USER REQUIREMENTS:
{requirements}

POLICIES TO SATISFY:
{policies_text}

TASK:
Generate a production-ready {domain} configuration that:
1. Meets all the user's requirements
2. Passes all specified policies
3. Follows {domain} best practices
4. Includes helpful inline comments

OUTPUT:
Provide the complete YAML configuration with explanations."""

    return prompt


def build_remediate_prompt(domain: str, original_config: str, violations: str) -> str:
    """Build a prompt for fixing violations.

    Args:
        domain: Target domain
        original_config: Original configuration
        violations: Formatted violations string

    Returns:
        Complete prompt for AI remediation
    """
    template = get_prompt_template(domain, "remediate")

    prompt = f"""{template}

ORIGINAL CONFIGURATION:
```yaml
{original_config}
```

POLICY VIOLATIONS DETECTED:
{violations}

TASK:
Fix the violations while preserving all compliant settings. Provide:
1. The complete fixed YAML configuration
2. A summary of changes made
3. Explanation of why each change was necessary

OUTPUT:
Return the fixed configuration in YAML format."""

    return prompt


def build_explain_prompt(
    domain: str, policy_name: str, violation_msg: Optional[str] = None
) -> str:
    """Build a prompt for explaining policies.

    Args:
        domain: Target domain
        policy_name: Policy name to explain
        violation_msg: Optional violation message

    Returns:
        Complete prompt for policy explanation
    """
    template = get_prompt_template(domain, "explain")

    if violation_msg:
        prompt = f"""{template}

POLICY: {policy_name}
VIOLATION: {violation_msg}

Please explain:
1. What this policy checks for
2. Why this violation occurred
3. How to fix it
4. Best practices to prevent this in the future"""
    else:
        prompt = f"""{template}

POLICY: {policy_name}

Please explain:
1. What this policy enforces
2. Why this policy is important
3. Common scenarios where it applies
4. Examples of good and bad configurations"""

    return prompt


def format_policy_list(policies: List[str]) -> str:
    """Format policy list for prompts.

    Args:
        policies: List of policy names

    Returns:
        Formatted policy list
    """
    if not policies:
        return "- All applicable policies"

    return "\n".join(f"- {p}" for p in policies)


def format_violations(violations: List) -> str:
    """Format violations for prompts.

    Args:
        violations: List of Violation objects

    Returns:
        Formatted violations string
    """
    result = []
    for i, v in enumerate(violations, 1):
        result.append(f"{i}. [{v.severity.value}] {v.policy}")
        result.append(f"   Message: {v.message}")
        if v.field:
            result.append(f"   Field: {v.field}")
            result.append(f"   Current: {v.current_value}")
            result.append(f"   Expected: {v.expected_value}")
        result.append("")

    return "\n".join(result)


# Example configurations for each domain
EXAMPLE_CONFIGURATIONS = {
    "kafka": """# Example: High-throughput event topic
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: user-events
  labels:
    app: user-service
    environment: production
spec:
  replicas: 3
  partitions: 12
  config:
    compression.type: "lz4"
    retention.ms: "604800000"  # 7 days
    min.insync.replicas: "2"
    segment.ms: "3600000"  # 1 hour""",
    "kubernetes": """# Example: Production deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
  labels:
    app: api-server
    environment: production
    owner: platform-team
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: api
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "1000m"
            memory: "1Gi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080""",
}


def get_example_configuration(domain: str) -> Optional[str]:
    """Get example configuration for a domain.

    Args:
        domain: Domain name

    Returns:
        Example configuration or None
    """
    return EXAMPLE_CONFIGURATIONS.get(domain)
