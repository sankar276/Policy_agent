package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"policy-agent/pkg/types"
)

// ClaudeClient implements the Client interface using Anthropic's Claude API
type ClaudeClient struct {
	client      *anthropic.Client
	model       string
	cache       *ResponseCache
	rateLimiter *RateLimiter
}

// ClaudeConfig holds configuration for the Claude client
type ClaudeConfig struct {
	APIKey              string
	Model               string
	RequestsPerMinute   int
	EnableCache         bool
	CacheTTL            time.Duration
}

// NewClaudeClient creates a new Claude AI client
func NewClaudeClient(config *ClaudeConfig) (*ClaudeClient, error) {
	if config.APIKey == "" {
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	if config.Model == "" {
		config.Model = "claude-sonnet-4-5-20250929"
	}

	client := anthropic.NewClient(
		option.WithAPIKey(config.APIKey),
	)

	var cache *ResponseCache
	if config.EnableCache {
		cache = NewResponseCache(config.CacheTTL)
	}

	rateLimiter := NewRateLimiter(config.RequestsPerMinute)

	return &ClaudeClient{
		client:      client,
		model:       config.Model,
		cache:       cache,
		rateLimiter: rateLimiter,
	}, nil
}

// Generate creates a policy-compliant configuration from natural language requirements
func (c *ClaudeClient) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// Wait for rate limiter
	if c.rateLimiter != nil {
		c.rateLimiter.Wait(ctx)
	}

	// Check cache
	cacheKey := fmt.Sprintf("generate:%s:%s", req.Domain, req.Requirements)
	if c.cache != nil {
		if cached, ok := c.cache.Get(cacheKey); ok {
			return cached.(*GenerateResponse), nil
		}
	}

	// Build prompt
	prompt := c.buildGeneratePrompt(req)

	// Call Claude API
	response, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(c.model),
		MaxTokens: anthropic.F(int64(4096)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})

	if err != nil {
		return nil, fmt.Errorf("claude API error: %w", err)
	}

	// Extract generated configuration
	content := extractTextContent(response)
	config, explanation := c.parseGeneratedConfig(content, req.Domain)

	result := &GenerateResponse{
		Configuration: config,
		Format:        types.FormatYAML,
		Explanation:   explanation,
		PoliciesMet:   req.Policies,
	}

	// Cache result
	if c.cache != nil {
		c.cache.Set(cacheKey, result)
	}

	return result, nil
}

// Remediate suggests fixes for policy violations
func (c *ClaudeClient) Remediate(ctx context.Context, req *RemediateRequest) (*RemediateResponse, error) {
	// Wait for rate limiter
	if c.rateLimiter != nil {
		c.rateLimiter.Wait(ctx)
	}

	// Check cache
	cacheKey := fmt.Sprintf("remediate:%s:%d", req.Domain, len(req.Violations))
	if c.cache != nil {
		if cached, ok := c.cache.Get(cacheKey); ok {
			return cached.(*RemediateResponse), nil
		}
	}

	// Build prompt
	prompt := c.buildRemediatePrompt(req)

	// Call Claude API
	response, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(c.model),
		MaxTokens: anthropic.F(int64(4096)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})

	if err != nil {
		return nil, fmt.Errorf("claude API error: %w", err)
	}

	// Parse remediation response
	content := extractTextContent(response)
	result := c.parseRemediationResponse(content, req)

	// Cache result
	if c.cache != nil {
		c.cache.Set(cacheKey, result)
	}

	return result, nil
}

// Explain provides explanations for policies or violations
func (c *ClaudeClient) Explain(ctx context.Context, req *ExplainRequest) (*ExplainResponse, error) {
	// Wait for rate limiter
	if c.rateLimiter != nil {
		c.rateLimiter.Wait(ctx)
	}

	// Build prompt
	prompt := c.buildExplainPrompt(req)

	// Call Claude API
	response, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(c.model),
		MaxTokens: anthropic.F(int64(2048)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})

	if err != nil {
		return nil, fmt.Errorf("claude API error: %w", err)
	}

	content := extractTextContent(response)

	return &ExplainResponse{
		Explanation: content,
		Examples:    []string{}, // TODO: Extract examples from response
	}, nil
}

// buildGeneratePrompt creates a prompt for configuration generation
func (c *ClaudeClient) buildGeneratePrompt(req *GenerateRequest) string {
	template := GetPromptTemplate(req.Domain, "generate")

	prompt := fmt.Sprintf(`You are an expert in %s configuration and policy compliance.

TASK: Generate a production-ready, policy-compliant configuration.

REQUIREMENTS:
%s

POLICIES TO FOLLOW:
%s

OUTPUT FORMAT:
- Provide valid YAML configuration
- Include inline comments explaining key decisions
- Ensure all required policies are satisfied
- Follow %s best practices

CONSTRAINTS:
- Must pass all specified policies
- Prioritize security and reliability
- Use industry-standard configurations

Generate the configuration now:`,
		req.Domain,
		req.Requirements,
		formatPolicies(req.Policies),
		req.Domain,
	)

	if template != "" {
		prompt = template + "\n\n" + prompt
	}

	return prompt
}

// buildRemediatePrompt creates a prompt for fixing violations
func (c *ClaudeClient) buildRemediatePrompt(req *RemediateRequest) string {
	violationsText := formatViolations(req.Violations)

	return fmt.Sprintf(`You are an expert in %s configuration and policy remediation.

TASK: Fix the following policy violations in the configuration.

ORIGINAL CONFIGURATION:
%s

POLICY VIOLATIONS:
%s

INSTRUCTIONS:
1. Analyze each violation
2. Provide specific fixes for each issue
3. Explain why each change is needed
4. Return the complete fixed configuration
5. Ensure no new violations are introduced

OUTPUT FORMAT:
Provide the fixed configuration in YAML format, followed by an explanation of changes made.

Generate the fixed configuration now:`,
		req.Domain,
		req.OriginalConfig,
		violationsText,
	)
}

// buildExplainPrompt creates a prompt for explaining policies
func (c *ClaudeClient) buildExplainPrompt(req *ExplainRequest) string {
	if req.Violation != nil {
		return fmt.Sprintf(`Explain the following policy violation in simple terms:

Policy: %s
Message: %s
Field: %s
Current Value: %v
Expected Value: %v

Provide:
1. What the policy checks for and why it matters
2. What went wrong in this case
3. How to fix it
4. Best practices related to this policy`,
			req.Violation.Policy,
			req.Violation.Message,
			req.Violation.Field,
			req.Violation.CurrentValue,
			req.Violation.ExpectedValue,
		)
	}

	return fmt.Sprintf(`Explain the following policy in simple terms:

Policy: %s

Provide:
1. What this policy checks for
2. Why this policy is important
3. Common violations and how to avoid them
4. Best practices and examples`,
		req.Policy,
	)
}

// parseGeneratedConfig extracts configuration and explanation from Claude's response
func (c *ClaudeClient) parseGeneratedConfig(content, domain string) (config, explanation string) {
	// Try to extract YAML block
	yamlStart := strings.Index(content, "```yaml")
	if yamlStart == -1 {
		yamlStart = strings.Index(content, "```")
	}

	if yamlStart != -1 {
		yamlStart = strings.Index(content[yamlStart:], "\n") + yamlStart + 1
		yamlEnd := strings.Index(content[yamlStart:], "```")
		if yamlEnd != -1 {
			config = strings.TrimSpace(content[yamlStart : yamlStart+yamlEnd])
			explanation = strings.TrimSpace(content[:yamlStart-4] + content[yamlStart+yamlEnd+3:])
		}
	}

	if config == "" {
		// No code block found, use entire response as config
		config = content
		explanation = "Generated configuration based on requirements"
	}

	return config, explanation
}

// parseRemediationResponse extracts fixed config and suggestions from Claude's response
func (c *ClaudeClient) parseRemediationResponse(content string, req *RemediateRequest) *RemediateResponse {
	config, explanation := c.parseGeneratedConfig(content, req.Domain)

	// Create suggestions for each violation
	suggestions := make([]Suggestion, len(req.Violations))
	for i, v := range req.Violations {
		suggestions[i] = Suggestion{
			Field:        v.Field,
			CurrentValue: v.CurrentValue,
			NewValue:     v.ExpectedValue,
			Suggestion:   fmt.Sprintf("Fix %s violation", v.Policy),
			Explanation:  explanation,
			AutoFixable:  true,
		}
	}

	return &RemediateResponse{
		FixedConfig:  config,
		Suggestions:  suggestions,
		Explanation:  explanation,
	}
}

// Helper functions

func extractTextContent(response *anthropic.Message) string {
	var content strings.Builder
	for _, block := range response.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}
	return content.String()
}

func formatPolicies(policies []string) string {
	if len(policies) == 0 {
		return "Follow all applicable best practices"
	}
	return strings.Join(policies, "\n- ")
}

func formatViolations(violations []types.Violation) string {
	var result strings.Builder
	for i, v := range violations {
		result.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, v.Severity, v.Policy))
		result.WriteString(fmt.Sprintf("   Message: %s\n", v.Message))
		if v.Field != "" {
			result.WriteString(fmt.Sprintf("   Field: %s\n", v.Field))
			result.WriteString(fmt.Sprintf("   Current: %v\n", v.CurrentValue))
			result.WriteString(fmt.Sprintf("   Expected: %v\n", v.ExpectedValue))
		}
		result.WriteString("\n")
	}
	return result.String()
}

// ResponseCache provides caching for AI responses
type ResponseCache struct {
	cache map[string]*cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func NewResponseCache(ttl time.Duration) *ResponseCache {
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	return &ResponseCache{
		cache: make(map[string]*cacheEntry),
		ttl:   ttl,
	}
}

func (c *ResponseCache) Get(key string) (interface{}, bool) {
	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.cache, key)
		return nil, false
	}
	return entry.value, true
}

func (c *ResponseCache) Set(key string, value interface{}) {
	c.cache[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// RateLimiter implements simple rate limiting
type RateLimiter struct {
	requestsPerMinute int
	lastRequest       time.Time
	minInterval       time.Duration
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 50
	}
	return &RateLimiter{
		requestsPerMinute: requestsPerMinute,
		minInterval:       time.Minute / time.Duration(requestsPerMinute),
	}
}

func (r *RateLimiter) Wait(ctx context.Context) error {
	if r.lastRequest.IsZero() {
		r.lastRequest = time.Now()
		return nil
	}

	elapsed := time.Since(r.lastRequest)
	if elapsed < r.minInterval {
		waitTime := r.minInterval - elapsed
		select {
		case <-time.After(waitTime):
			r.lastRequest = time.Now()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	r.lastRequest = time.Now()
	return nil
}

// Serialize/deserialize helpers for caching
func serializeToJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func deserializeFromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}
