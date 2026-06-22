// internal/config/config.go

package config

import (
    "context"
    "fmt"
    "os"
    "strconv"
    "strings"
)

// Config holds all configuration for the FinOps Bot.
type Config struct {
    // Core settings
    DryRun           bool
    Environment      string
    LogLevel         string
    Timezone         string

    // AWS resources
    Regions          []string
    S3ReportBucket   string
    S3ReportPrefix   string
    SecretsManagerARN string
    SNSTopicARN      string

    // Pricing
    CostPerGB        float64

    // Resource cleanup rules
    ExcludedIDs      map[string]bool
    QuarantineDays   int
    SnapshotRetentionDays int
    SnapshotsToKeep  int
    RDSStopAgeDays   int

    // Slack
    SlackChannel     string
    SlackWebhookURL  string

    // Features
    EnableRDSSavings bool
}

// LoadConfig loads configuration from environment variables, SSM Parameter Store,
// and Secrets Manager. It returns a validated Config.
func LoadConfig(ctx context.Context) (*Config, error) {
    cfg := &Config{}

    // 1. Load from environment variables
    loadFromEnv(cfg)

    // 2. Load from SSM Parameter Store (stub for now)
    if err := loadFromSSM(ctx, cfg); err != nil {
        // In production, this could be fatal; for local dev we continue.
        // We'll just log but not fail.
        // We'll assume caller handles logging.
    }

    // 3. Load secrets from Secrets Manager (stub for now)
    if err := loadFromSecretsManager(ctx, cfg); err != nil {
        // Continue on error for local development.
    }

    // 4. Validate
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    return cfg, nil
}

// loadFromEnv populates config from environment variables.
func loadFromEnv(cfg *Config) {
    cfg.DryRun = getEnvBool("DRY_RUN", true)
    cfg.Environment = getEnvString("ENVIRONMENT", "dev")
    cfg.LogLevel = getEnvString("LOG_LEVEL", "info")
    cfg.Timezone = getEnvString("TZ", "UTC")

    cfg.Regions = getEnvSlice("REGIONS", []string{"us-east-1", "us-west-2", "eu-west-1"})
    cfg.S3ReportBucket = getEnvString("S3_REPORT_BUCKET", "")
    cfg.S3ReportPrefix = getEnvString("S3_REPORT_PREFIX", "audit/")
    cfg.SecretsManagerARN = getEnvString("SECRETS_MANAGER_ARN", "")
    cfg.SNSTopicARN = getEnvString("SNS_TOPIC_ARN", "")

    cfg.CostPerGB = getEnvFloat("COST_PER_GB", 0.08)

    cfg.ExcludedIDs = getEnvMap("EXCLUDED_IDS")
    cfg.QuarantineDays = getEnvInt("QUARANTINE_DAYS", 7)
    cfg.SnapshotRetentionDays = getEnvInt("SNAPSHOT_RETENTION_DAYS", 30)
    cfg.SnapshotsToKeep = getEnvInt("SNAPSHOTS_TO_KEEP", 3)
    cfg.RDSStopAgeDays = getEnvInt("RDS_STOP_AGE_DAYS", 7)

    cfg.SlackChannel = getEnvString("SLACK_CHANNEL", "")
    cfg.SlackWebhookURL = getEnvString("SLACK_WEBHOOK_URL", "")

    cfg.EnableRDSSavings = getEnvBool("ENABLE_RDS_SAVINGS", true)
}

// loadFromSSM loads non-sensitive configuration from SSM Parameter Store.
// For now, this is a stub that does nothing.
func loadFromSSM(ctx context.Context, cfg *Config) error {
    // TODO: Implement SSM loading in Phase 3.
    // For now, return nil.
    return nil
}

// loadFromSecretsManager loads secrets from AWS Secrets Manager.
// For now, this is a stub that does nothing.
func loadFromSecretsManager(ctx context.Context, cfg *Config) error {
    // TODO: Implement Secrets Manager loading in Phase 3.
    // For now, return nil.
    return nil
}

// Validate checks that required fields are set and values are reasonable.
func (c *Config) Validate() error {
    var errs []string

    if c.S3ReportBucket == "" {
        errs = append(errs, "S3_REPORT_BUCKET is required")
    }
    if c.SecretsManagerARN == "" && c.SlackWebhookURL == "" {
        // Not failing for local dev, but we could add a warning.
        // We'll just allow it.
    }
    if c.QuarantineDays < 1 {
        errs = append(errs, "QUARANTINE_DAYS must be >= 1")
    }
    if c.SnapshotRetentionDays < 1 {
        errs = append(errs, "SNAPSHOT_RETENTION_DAYS must be >= 1")
    }
    if c.SnapshotsToKeep < 0 {
        errs = append(errs, "SNAPSHOTS_TO_KEEP must be >= 0")
    }
    if c.RDSStopAgeDays < 1 {
        errs = append(errs, "RDS_STOP_AGE_DAYS must be >= 1")
    }
    if c.CostPerGB <= 0 {
        errs = append(errs, "COST_PER_GB must be > 0")
    }
    if len(c.Regions) == 0 {
        errs = append(errs, "REGIONS must have at least one region")
    }

    if len(errs) > 0 {
        return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
    }

    return nil
}

// Helper functions for environment variable parsing.

func getEnvString(key, defaultVal string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
    if v := os.Getenv(key); v != "" {
        b, err := strconv.ParseBool(v)
        if err == nil {
            return b
        }
    }
    return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
    if v := os.Getenv(key); v != "" {
        i, err := strconv.Atoi(v)
        if err == nil {
            return i
        }
    }
    return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
    if v := os.Getenv(key); v != "" {
        f, err := strconv.ParseFloat(v, 64)
        if err == nil {
            return f
        }
    }
    return defaultVal
}

func getEnvSlice(key string, defaultVal []string) []string {
    if v := os.Getenv(key); v != "" {
        parts := strings.Split(v, ",")
        result := make([]string, 0, len(parts))
        for _, p := range parts {
            p = strings.TrimSpace(p)
            if p != "" {
                result = append(result, p)
            }
        }
        if len(result) > 0 {
            return result
        }
    }
    return defaultVal
}

func getEnvMap(key string) map[string]bool {
    result := make(map[string]bool)
    if v := os.Getenv(key); v != "" {
        parts := strings.Split(v, ",")
        for _, p := range parts {
            p = strings.TrimSpace(p)
            if p != "" {
                result[p] = true
            }
        }
    }
    return result
}