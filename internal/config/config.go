package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Security SecurityConfig `json:"security"`
	RateLimit RateLimitConfig `json:"rate_limit"`
	Tracing  TracingConfig  `json:"tracing"`
	Features FeaturesConfig `json:"features"`
	Cache    CacheConfig    `json:"cache"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port     string `json:"port"`
	Host     string `json:"host"`
	EnableTLS bool  `json:"enable_tls"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

// DatabaseConfig holds database-related configuration.
type DatabaseConfig struct {
	Path string `json:"path"`
}

// SecurityConfig holds security-related configuration.
type SecurityConfig struct {
	// Max request body size in bytes (default: 10MB)
	MaxRequestBodySize int64 `json:"max_request_body_size"`
	// Allowed CORS origins (comma-separated)
	AllowedOrigins string `json:"allowed_origins"`
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Enabled bool `json:"enabled"`
	Rate    int  `json:"rate"`
	Window  int  `json:"window"` // in seconds
}

// TracingConfig holds distributed tracing configuration.
type TracingConfig struct {
	Enabled     bool   `json:"enabled"`
	Endpoint    string `json:"endpoint"`     // Jaeger endpoint
	ServiceName string `json:"service_name"` // Service name for traces
	Environment string `json:"environment"`  // Deployment environment
}

// FeaturesConfig holds feature flags configuration.
type FeaturesConfig struct {
	CacheEnabled          bool `json:"cache_enabled"`
	EventHooksEnabled     bool `json:"event_hooks_enabled"`
	AdvancedEligibility   bool `json:"advanced_eligibility"`
	BatchProcessing       bool `json:"batch_processing"`
}

// CacheConfig holds cache configuration.
type CacheConfig struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"`      // "redis" or "memory"
	Addr     string `json:"addr"`      // Redis address (e.g., "localhost:6379")
	Password string `json:"password"`  // Redis password
	DB       int    `json:"db"`        // Redis database number
	TTL      int    `json:"ttl"`       // Default TTL in seconds
}

// LoadConfig loads configuration from environment variables and/or config file.
// Environment variables take precedence over config file values.
func LoadConfig(configFile string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:     getEnv("SERVER_PORT", "8080"),
			Host:     getEnv("SERVER_HOST", ""),
			EnableTLS: getEnvBool("SERVER_ENABLE_TLS", false),
			CertFile: getEnv("SERVER_CERT_FILE", ""),
			KeyFile:  getEnv("SERVER_KEY_FILE", ""),
		},
		Database: DatabaseConfig{
			Path: getEnv("DATABASE_PATH", "./offer_eligibility.db"),
		},
		Security: SecurityConfig{
			MaxRequestBodySize: getEnvInt64("MAX_REQUEST_BODY_SIZE", 10<<20), // 10MB default
			AllowedOrigins:     getEnv("ALLOWED_ORIGINS", "*"),
		},
		RateLimit: RateLimitConfig{
			Enabled: getEnvBool("RATE_LIMIT_ENABLED", true),
			Rate:    getEnvInt("RATE_LIMIT_RATE", 100),
			Window:  getEnvInt("RATE_LIMIT_WINDOW", 60),
		},
		Tracing: TracingConfig{
			Enabled:     getEnvBool("TRACING_ENABLED", false),
			Endpoint:    getEnv("TRACING_ENDPOINT", "http://localhost:14268/api/traces"),
			ServiceName: getEnv("TRACING_SERVICE_NAME", "offer-eligibility-api"),
			Environment: getEnv("TRACING_ENVIRONMENT", "development"),
		},
		Features: FeaturesConfig{
			CacheEnabled:        getEnvBool("FEATURE_CACHE_ENABLED", false),
			EventHooksEnabled:    getEnvBool("FEATURE_EVENT_HOOKS_ENABLED", false),
			AdvancedEligibility: getEnvBool("FEATURE_ADVANCED_ELIGIBILITY", false),
			BatchProcessing:      getEnvBool("FEATURE_BATCH_PROCESSING", false),
		},
		Cache: CacheConfig{
			Enabled:  getEnvBool("CACHE_ENABLED", false),
			Type:     getEnv("CACHE_TYPE", "memory"),
			Addr:     getEnv("CACHE_ADDR", "localhost:6379"),
			Password: getEnv("CACHE_PASSWORD", ""),
			DB:       getEnvInt("CACHE_DB", 0),
			TTL:      getEnvInt("CACHE_TTL", 300), // 5 minutes default
		},
	}

	// Load from config file if provided
	if configFile != "" {
		if err := loadFromFile(configFile, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Override with environment variables (they take precedence)
	overrideFromEnv(cfg)

	return cfg, nil
}

// loadFromFile loads configuration from a JSON file.
func loadFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, cfg)
}

// overrideFromEnv overrides configuration with environment variables.
func overrideFromEnv(cfg *Config) {
	if port := os.Getenv("SERVER_PORT"); port != "" {
		cfg.Server.Port = port
	}
	if host := os.Getenv("SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if enableTLS := os.Getenv("SERVER_ENABLE_TLS"); enableTLS != "" {
		cfg.Server.EnableTLS = enableTLS == "true" || enableTLS == "1"
	}
	if certFile := os.Getenv("SERVER_CERT_FILE"); certFile != "" {
		cfg.Server.CertFile = certFile
	}
	if keyFile := os.Getenv("SERVER_KEY_FILE"); keyFile != "" {
		cfg.Server.KeyFile = keyFile
	}
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}
	if maxBodySize := os.Getenv("MAX_REQUEST_BODY_SIZE"); maxBodySize != "" {
		if size, err := strconv.ParseInt(maxBodySize, 10, 64); err == nil {
			cfg.Security.MaxRequestBodySize = size
		}
	}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		cfg.Security.AllowedOrigins = origins
	}
	if enabled := os.Getenv("RATE_LIMIT_ENABLED"); enabled != "" {
		cfg.RateLimit.Enabled = enabled == "true" || enabled == "1"
	}
	if rate := os.Getenv("RATE_LIMIT_RATE"); rate != "" {
		if r, err := strconv.Atoi(rate); err == nil {
			cfg.RateLimit.Rate = r
		}
	}
	if window := os.Getenv("RATE_LIMIT_WINDOW"); window != "" {
		if w, err := strconv.Atoi(window); err == nil {
			cfg.RateLimit.Window = w
		}
	}
	if enabled := os.Getenv("TRACING_ENABLED"); enabled != "" {
		cfg.Tracing.Enabled = enabled == "true" || enabled == "1"
	}
	if endpoint := os.Getenv("TRACING_ENDPOINT"); endpoint != "" {
		cfg.Tracing.Endpoint = endpoint
	}
	if serviceName := os.Getenv("TRACING_SERVICE_NAME"); serviceName != "" {
		cfg.Tracing.ServiceName = serviceName
	}
	if environment := os.Getenv("TRACING_ENVIRONMENT"); environment != "" {
		cfg.Tracing.Environment = environment
	}
	if enabled := os.Getenv("FEATURE_CACHE_ENABLED"); enabled != "" {
		cfg.Features.CacheEnabled = enabled == "true" || enabled == "1"
	}
	if enabled := os.Getenv("FEATURE_EVENT_HOOKS_ENABLED"); enabled != "" {
		cfg.Features.EventHooksEnabled = enabled == "true" || enabled == "1"
	}
	if enabled := os.Getenv("FEATURE_ADVANCED_ELIGIBILITY"); enabled != "" {
		cfg.Features.AdvancedEligibility = enabled == "true" || enabled == "1"
	}
	if enabled := os.Getenv("FEATURE_BATCH_PROCESSING"); enabled != "" {
		cfg.Features.BatchProcessing = enabled == "true" || enabled == "1"
	}
	if enabled := os.Getenv("CACHE_ENABLED"); enabled != "" {
		cfg.Cache.Enabled = enabled == "true" || enabled == "1"
	}
	if cacheType := os.Getenv("CACHE_TYPE"); cacheType != "" {
		cfg.Cache.Type = cacheType
	}
	if addr := os.Getenv("CACHE_ADDR"); addr != "" {
		cfg.Cache.Addr = addr
	}
	if password := os.Getenv("CACHE_PASSWORD"); password != "" {
		cfg.Cache.Password = password
	}
	if db := os.Getenv("CACHE_DB"); db != "" {
		if d, err := strconv.Atoi(db); err == nil {
			cfg.Cache.DB = d
		}
	}
	if ttl := os.Getenv("CACHE_TTL"); ttl != "" {
		if t, err := strconv.Atoi(ttl); err == nil {
			cfg.Cache.TTL = t
		}
	}
}

// getEnv gets an environment variable or returns the default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable or returns the default value.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable or returns the default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// getEnvInt64 gets an int64 environment variable or returns the default value.
func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

// Validate validates the configuration and returns any errors.
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}
	if c.Server.EnableTLS {
		if c.Server.CertFile == "" || c.Server.KeyFile == "" {
			// Self-signed cert will be generated, so this is OK
		}
	}
	if c.RateLimit.Enabled {
		if c.RateLimit.Rate <= 0 {
			return fmt.Errorf("rate limit rate must be positive")
		}
		if c.RateLimit.Window <= 0 {
			return fmt.Errorf("rate limit window must be positive")
		}
	}
	return nil
}

