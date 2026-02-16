package ai

import "fmt"

// PromptTemplate represents a template for AI prompts
type PromptTemplate struct {
	Domain   string
	Action   string // "generate", "remediate", "explain"
	Template string
}

// Prompt templates for different domains and actions
var promptTemplates = map[string]map[string]string{
	"kafka": {
		"generate": `You are a Kafka/Confluent for Kubernetes (CFK) expert specializing in production-ready topic configurations.

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
    # Additional configs...`,

		"remediate": `You are a Kafka expert specializing in fixing policy violations and improving topic configurations.

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
- Use inline comments to explain fixes`,

		"explain": `You are a Kafka educator explaining topics and policies in simple, practical terms.

EXPLANATION STYLE:
- Use clear, non-technical language when possible
- Provide real-world analogies and examples
- Explain both the "what" and the "why"
- Include practical implications (cost, performance, reliability)
- Suggest actionable next steps`,
	},

	"kubernetes": {
		"generate": `You are a Kubernetes expert specializing in production-ready manifest configurations.

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
4. Readiness/Liveness: Health checks should be configured`,

		"remediate": `You are a Kubernetes expert fixing policy violations in resource manifests.

REMEDIATION APPROACH:
1. Add missing required labels
2. Set appropriate resource limits based on workload type
3. Remove security violations (privileged, hostNetwork, etc.)
4. Add health checks if missing`,
	},

	"iac": {
		"generate": `You are a Terraform/IaC expert specializing in secure, compliant infrastructure code.

EXPERTISE:
- Terraform best practices and modules
- Cloud provider resources (AWS, Azure, GCP)
- State management and backend configuration
- Security and compliance requirements

KEY POLICY REQUIREMENTS:
1. Provider versions must be pinned
2. Remote state with encryption
3. Resource tagging and naming conventions
4. Security groups and IAM policies`,
	},
}

// GetPromptTemplate retrieves a prompt template for a domain and action
func GetPromptTemplate(domain, action string) string {
	if templates, ok := promptTemplates[domain]; ok {
		if template, ok := templates[action]; ok {
			return template
		}
	}
	return ""
}

// BuildKafkaGeneratePrompt creates a detailed prompt for Kafka topic generation
func BuildKafkaGeneratePrompt(requirements string, policies []string) string {
	baseTemplate := GetPromptTemplate("kafka", "generate")

	return fmt.Sprintf(`%s

USER REQUIREMENTS:
%s

POLICIES TO SATISFY:
%s

TASK:
Generate a production-ready Kafka topic configuration that:
1. Meets all the user's requirements
2. Passes all specified policies
3. Follows Kafka best practices
4. Includes helpful inline comments

OUTPUT:
Provide the complete YAML configuration with explanations.`,
		baseTemplate,
		requirements,
		formatPolicyList(policies),
	)
}

// BuildKafkaRemediatePrompt creates a prompt for fixing Kafka topic violations
func BuildKafkaRemediatePrompt(originalConfig string, violations string) string {
	baseTemplate := GetPromptTemplate("kafka", "remediate")

	return fmt.Sprintf(`%s

ORIGINAL CONFIGURATION:
'''yaml
%s
'''

POLICY VIOLATIONS DETECTED:
%s

TASK:
Fix the violations while preserving all compliant settings. Provide:
1. The complete fixed YAML configuration
2. A summary of changes made
3. Explanation of why each change was necessary

OUTPUT:
Return the fixed configuration in YAML format.`,
		baseTemplate,
		originalConfig,
		violations,
	)
}

// BuildExplainPrompt creates a prompt for explaining policies or violations
func BuildExplainPrompt(domain, policyName, violationMsg string) string {
	baseTemplate := GetPromptTemplate(domain, "explain")

	if violationMsg != "" {
		return fmt.Sprintf(`%s

POLICY: %s
VIOLATION: %s

Please explain:
1. What this policy checks for
2. Why this violation occurred
3. How to fix it
4. Best practices to prevent this in the future`,
			baseTemplate,
			policyName,
			violationMsg,
		)
	}

	return fmt.Sprintf(`%s

POLICY: %s

Please explain:
1. What this policy enforces
2. Why this policy is important
3. Common scenarios where it applies
4. Examples of good and bad configurations`,
		baseTemplate,
		policyName,
	)
}

// Helper functions

func formatPolicyList(policies []string) string {
	if len(policies) == 0 {
		return "- All applicable Kafka policies\n- kafka.topics.replication\n- kafka.topics.compression\n- kafka.topics.retention"
	}

	result := ""
	for _, p := range policies {
		result += fmt.Sprintf("- %s\n", p)
	}
	return result
}

// DomainPrompts provides domain-specific prompting assistance
type DomainPrompts struct {
	Domain string
}

// NewDomainPrompts creates a new domain prompts helper
func NewDomainPrompts(domain string) *DomainPrompts {
	return &DomainPrompts{Domain: domain}
}

// GetGenerateSystemPrompt returns the system prompt for generation
func (dp *DomainPrompts) GetGenerateSystemPrompt() string {
	switch dp.Domain {
	case "kafka":
		return "You are an expert Kafka/CFK engineer creating production-ready topic configurations."
	case "kubernetes":
		return "You are an expert Kubernetes engineer creating production-ready resource manifests."
	case "iac":
		return "You are an expert Infrastructure-as-Code engineer creating secure, compliant Terraform configurations."
	default:
		return fmt.Sprintf("You are an expert %s engineer creating production-ready configurations.", dp.Domain)
	}
}

// GetRemediateSystemPrompt returns the system prompt for remediation
func (dp *DomainPrompts) GetRemediateSystemPrompt() string {
	switch dp.Domain {
	case "kafka":
		return "You are an expert at fixing Kafka topic configuration issues and policy violations."
	case "kubernetes":
		return "You are an expert at fixing Kubernetes resource manifest issues and policy violations."
	default:
		return fmt.Sprintf("You are an expert at fixing %s configuration issues and policy violations.", dp.Domain)
	}
}

// EnrichRequirements adds domain-specific context to user requirements
func (dp *DomainPrompts) EnrichRequirements(requirements string) string {
	switch dp.Domain {
	case "kafka":
		return fmt.Sprintf(`%s

Consider:
- Appropriate partition count based on throughput needs
- Retention period based on use case (events, logs, CDC, etc.)
- Cleanup policy (delete vs compact)
- Segment size for efficient storage
- Any special configurations needed for the use case`, requirements)
	default:
		return requirements
	}
}

// ExampleConfigurations provides example configurations for each domain
var ExampleConfigurations = map[string]string{
	"kafka": `# Example: High-throughput event topic
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
    segment.ms: "3600000"  # 1 hour`,

	"kubernetes": `# Example: Production deployment
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
            port: 8080`,
}

// GetExampleConfiguration returns an example configuration for a domain
func GetExampleConfiguration(domain string) string {
	if example, ok := ExampleConfigurations[domain]; ok {
		return example
	}
	return ""
}
