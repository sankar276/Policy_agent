package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Agent        AgentConfig        `mapstructure:"agent"`
	Policy       PolicyConfig       `mapstructure:"policy"`
	AI           AIConfig           `mapstructure:"ai"`
	Domains      DomainsConfig      `mapstructure:"domains"`
	Integrations IntegrationsConfig `mapstructure:"integrations"`
	Storage      StorageConfig      `mapstructure:"storage"`
	Telemetry    TelemetryConfig    `mapstructure:"telemetry"`
}

type AgentConfig struct {
	Name     string `mapstructure:"name"`
	LogLevel string `mapstructure:"log_level"`
}

type PolicyConfig struct {
	Engine          string   `mapstructure:"engine"`
	PolicyPath      string   `mapstructure:"policy_path"`
	EnabledDomains  []string `mapstructure:"enabled_domains"`
	EnforcementMode string   `mapstructure:"enforcement_mode"`
}

type AIConfig struct {
	Provider      string        `mapstructure:"provider"`
	APIKeyEnv     string        `mapstructure:"api_key_env"`
	Model         string        `mapstructure:"model"`
	Features      AIFeatures    `mapstructure:"features"`
	RateLimiting  RateLimiting  `mapstructure:"rate_limiting"`
	Cache         CacheConfig   `mapstructure:"cache"`
}

type AIFeatures struct {
	AutoRemediation bool `mapstructure:"auto_remediation"`
	Generation      bool `mapstructure:"generation"`
	Explanation     bool `mapstructure:"explanation"`
}

type RateLimiting struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
	Burst             int `mapstructure:"burst"`
}

type CacheConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	TTL     string `mapstructure:"ttl"`
}

type DomainsConfig struct {
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	IAC        IACConfig        `mapstructure:"iac"`
	CICD       CICDConfig       `mapstructure:"cicd"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Postgres   PostgresConfig   `mapstructure:"postgres"`
	Flink      FlinkConfig      `mapstructure:"flink"`
	AppConfig  AppConfigConfig  `mapstructure:"appconfig"`
}

type KafkaConfig struct {
	MinReplicationFactor      int      `mapstructure:"min_replication_factor"`
	RequireMinInsyncReplicas  bool     `mapstructure:"require_min_insync_replicas"`
	MinInsyncReplicasValue    int      `mapstructure:"min_insync_replicas_value"`
	RequireCompression        bool     `mapstructure:"require_compression"`
	AllowedCompressionTypes   []string `mapstructure:"allowed_compression_types"`
	RecommendedCompression    string   `mapstructure:"recommended_compression"`
	MaxRetentionDays          int      `mapstructure:"max_retention_days"`
	WarnRetentionDays         int      `mapstructure:"warn_retention_days"`
	MinRetentionDays          int      `mapstructure:"min_retention_days"`
}

type KubernetesConfig struct {
	RequiredLabels         []string `mapstructure:"required_labels"`
	EnforceResourceLimits  bool     `mapstructure:"enforce_resource_limits"`
	DisallowPrivileged     bool     `mapstructure:"disallow_privileged"`
	DisallowHostNetwork    bool     `mapstructure:"disallow_host_network"`
}

type IACConfig struct {
	Terraform TerraformConfig `mapstructure:"terraform"`
}

type TerraformConfig struct {
	RequiredProvidersVersion bool `mapstructure:"required_providers_version"`
	StateEncryption          bool `mapstructure:"state_encryption"`
	RequireBackendConfig     bool `mapstructure:"require_backend_config"`
}

type CICDConfig struct {
	GitHubActions GitHubActionsConfig `mapstructure:"github_actions"`
}

type GitHubActionsConfig struct {
	DisallowSecretsInLogs  bool `mapstructure:"disallow_secrets_in_logs"`
	RequirePermissionsBlock bool `mapstructure:"require_permissions_block"`
}

type RedisConfig struct {
	RequirePassword bool `mapstructure:"require_password"`
	RequireTLS      bool `mapstructure:"require_tls"`
}

type PostgresConfig struct {
	RequireSSL              bool `mapstructure:"require_ssl"`
	EnforceConnectionLimits bool `mapstructure:"enforce_connection_limits"`
}

type FlinkConfig struct {
	RequireCheckpointing bool `mapstructure:"require_checkpointing"`
	MinParallelism       int  `mapstructure:"min_parallelism"`
}

type AppConfigConfig struct {
	DisallowPlainSecrets  bool `mapstructure:"disallow_plain_secrets"`
	RequireEnvValidation  bool `mapstructure:"require_env_validation"`
}

type IntegrationsConfig struct {
	GitHooks GitHooksConfig `mapstructure:"git_hooks"`
	CICD     CICDIntegrationConfig `mapstructure:"cicd"`
	Webhook  WebhookConfig  `mapstructure:"webhook"`
}

type GitHooksConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	Hooks            []string `mapstructure:"hooks"`
	FailOnViolation  bool     `mapstructure:"fail_on_violation"`
	FailOnWarning    bool     `mapstructure:"fail_on_warning"`
}

type CICDIntegrationConfig struct {
	GitHubActions GitHubActionsIntegration `mapstructure:"github_actions"`
	GitLabCI      GitLabCIIntegration      `mapstructure:"gitlab_ci"`
}

type GitHubActionsIntegration struct {
	Enabled         bool `mapstructure:"enabled"`
	FailOnViolation bool `mapstructure:"fail_on_violation"`
}

type GitLabCIIntegration struct {
	Enabled         bool `mapstructure:"enabled"`
	FailOnViolation bool `mapstructure:"fail_on_violation"`
}

type WebhookConfig struct {
	Enabled        bool      `mapstructure:"enabled"`
	Port           int       `mapstructure:"port"`
	TLS            TLSConfig `mapstructure:"tls"`
	TimeoutSeconds int       `mapstructure:"timeout_seconds"`
}

type TLSConfig struct {
	CertPath string `mapstructure:"cert_path"`
	KeyPath  string `mapstructure:"key_path"`
}

type StorageConfig struct {
	Backend  string         `mapstructure:"backend"`
	Path     string         `mapstructure:"path"`
	AuditLog AuditLogConfig `mapstructure:"audit_log"`
}

type AuditLogConfig struct {
	Enabled       bool `mapstructure:"enabled"`
	RetentionDays int  `mapstructure:"retention_days"`
}

type TelemetryConfig struct {
	Enabled bool           `mapstructure:"enabled"`
	Export  string         `mapstructure:"export"`
	Metrics MetricsConfig  `mapstructure:"metrics"`
	Logging LoggingConfig  `mapstructure:"logging"`
	Tracing TracingConfig  `mapstructure:"tracing"`
}

type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

type LoggingConfig struct {
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type TracingConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
}

// Load loads configuration from file
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Look for config in common locations
		v.SetConfigName("policy-agent")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME/.policy-agent")
		v.AddConfigPath("/etc/policy-agent")
	}

	// Read environment variables
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Name:     "policy-ai-agent",
			LogLevel: "info",
		},
		Policy: PolicyConfig{
			Engine:          "opa",
			PolicyPath:      "./policies",
			EnabledDomains:  []string{"kafka", "kubernetes"},
			EnforcementMode: "strict",
		},
		AI: AIConfig{
			Provider:  "anthropic",
			APIKeyEnv: "ANTHROPIC_API_KEY",
			Model:     "claude-sonnet-4-5-20250929",
			Features: AIFeatures{
				AutoRemediation: true,
				Generation:      true,
				Explanation:     true,
			},
		},
		Domains: DomainsConfig{
			Kafka: KafkaConfig{
				MinReplicationFactor:     3,
				RequireMinInsyncReplicas: true,
				MinInsyncReplicasValue:   2,
				RequireCompression:       true,
				AllowedCompressionTypes:  []string{"lz4", "snappy", "zstd"},
				MaxRetentionDays:         90,
				WarnRetentionDays:        60,
			},
		},
	}
}

// GetAPIKey retrieves the API key from environment
func (c *AIConfig) GetAPIKey() string {
	return os.Getenv(c.APIKeyEnv)
}
