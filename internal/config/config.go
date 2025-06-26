package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration values for AWX deployment
type Config struct {
	// Kubernetes settings
	KubeconfigPath string
	Namespace      string

	// AWX settings
	AWXName       string
	AWXHostname   string
	AdminUser     string
	AdminPassword string

	// Storage settings
	StorageClass    string
	PostgresStorage string
	ProjectsStorage string

	// PostgreSQL settings
	PostgresHost     string
	PostgresPort     int
	PostgresDatabase string
	PostgresUsername string
	PostgresPassword string

	// Ingress settings
	IngressClassName string
	TLSSecretName    string
	CertIssuer       string

	// Operator settings
	OperatorVersion string
	OperatorTimeout int // in minutes
}

// NewConfigFromEnv creates a new Config from environment variables with defaults
func NewConfigFromEnv() (*Config, error) {
	cfg := &Config{
		// Kubernetes settings
		KubeconfigPath: getEnvOrDefault("KUBECONFIG", "/kubeconfig"),
		Namespace:      getEnvOrDefault("AWX_NAMESPACE", "awx"),

		// AWX settings
		AWXName:       getEnvOrDefault("AWX_NAME", "awx-instance"),
		AWXHostname:   getEnvOrDefault("AWX_HOSTNAME", "awx.sin.padminisys.com"),
		AdminUser:     getEnvOrDefault("AWX_ADMIN_USER", "admin"),
		AdminPassword: getEnvOrDefault("AWX_ADMIN_PASSWORD", "admin123!@#"),

		// Storage settings
		StorageClass:    getEnvOrDefault("AWX_STORAGE_CLASS", "hostpath"),
		PostgresStorage: getEnvOrDefault("AWX_POSTGRES_STORAGE", "8Gi"),
		ProjectsStorage: getEnvOrDefault("AWX_PROJECTS_STORAGE", "8Gi"),

		// PostgreSQL settings
		PostgresHost:     getEnvOrDefault("AWX_POSTGRES_HOST", "awx-instance-postgres-13"),
		PostgresDatabase: getEnvOrDefault("AWX_POSTGRES_DATABASE", "awx"),
		PostgresUsername: getEnvOrDefault("AWX_POSTGRES_USERNAME", "awx"),
		PostgresPassword: getEnvOrDefault("AWX_POSTGRES_PASSWORD", "awxpassword"),

		// Ingress settings
		IngressClassName: getEnvOrDefault("AWX_INGRESS_CLASS", "nginx"),
		TLSSecretName:    getEnvOrDefault("AWX_TLS_SECRET", "awx-tls"),
		CertIssuer:       getEnvOrDefault("AWX_CERT_ISSUER", "letsencrypt-prod"),

		// Operator settings
		OperatorVersion: getEnvOrDefault("AWX_OPERATOR_VERSION", "2.19.1"),
	}

	// Parse integer values
	var err error
	cfg.PostgresPort, err = strconv.Atoi(getEnvOrDefault("AWX_POSTGRES_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid AWX_POSTGRES_PORT: %v", err)
	}

	cfg.OperatorTimeout, err = strconv.Atoi(getEnvOrDefault("AWX_OPERATOR_TIMEOUT", "15"))
	if err != nil {
		return nil, fmt.Errorf("invalid AWX_OPERATOR_TIMEOUT: %v", err)
	}

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	return cfg, nil
}

// validate checks that all required configuration is present
func (c *Config) validate() error {
	if c.KubeconfigPath == "" {
		return fmt.Errorf("KUBECONFIG is required")
	}
	if c.AWXHostname == "" {
		return fmt.Errorf("AWX_HOSTNAME is required")
	}
	if c.AdminPassword == "" {
		return fmt.Errorf("AWX_ADMIN_PASSWORD is required")
	}
	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
