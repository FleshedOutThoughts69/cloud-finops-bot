// tests/unit/config_test.go

package unit

import (
    "context"
    "os"
    "testing"

    "cloud-finops-bot/internal/config"
    "github.com/stretchr/testify/assert"
)

func TestLoadConfigFromEnv(t *testing.T) {
    // Set environment variables for test
    os.Setenv("DRY_RUN", "false")
    os.Setenv("ENVIRONMENT", "test")
    os.Setenv("REGIONS", "us-west-2,eu-west-1")
    os.Setenv("QUARANTINE_DAYS", "14")
    os.Setenv("S3_REPORT_BUCKET", "test-bucket")
    os.Setenv("COST_PER_GB", "0.10")

    defer func() {
        os.Unsetenv("DRY_RUN")
        os.Unsetenv("ENVIRONMENT")
        os.Unsetenv("REGIONS")
        os.Unsetenv("QUARANTINE_DAYS")
        os.Unsetenv("S3_REPORT_BUCKET")
        os.Unsetenv("COST_PER_GB")
    }()

    cfg, err := config.LoadConfig(context.Background())
    assert.NoError(t, err)
    assert.NotNil(t, cfg)
    assert.Equal(t, false, cfg.DryRun)
    assert.Equal(t, "test", cfg.Environment)
    assert.Equal(t, []string{"us-west-2", "eu-west-1"}, cfg.Regions)
    assert.Equal(t, 14, cfg.QuarantineDays)
    assert.Equal(t, "test-bucket", cfg.S3ReportBucket)
    assert.Equal(t, 0.10, cfg.CostPerGB)
}

func TestLoadConfigDefault(t *testing.T) {
    // Set required environment variable for this test
    os.Setenv("S3_REPORT_BUCKET", "default-bucket")
    defer os.Unsetenv("S3_REPORT_BUCKET")

    cfg, err := config.LoadConfig(context.Background())
    assert.NoError(t, err)
    assert.NotNil(t, cfg)
    assert.Equal(t, true, cfg.DryRun)
    assert.Equal(t, "dev", cfg.Environment)
    assert.Equal(t, []string{"us-east-1", "us-west-2", "eu-west-1"}, cfg.Regions)
    assert.Equal(t, 7, cfg.QuarantineDays)
    assert.Equal(t, "default-bucket", cfg.S3ReportBucket)
}

func TestValidateConfig(t *testing.T) {
    cfg := &config.Config{
        S3ReportBucket:   "test-bucket",
        QuarantineDays:   7,
        SnapshotRetentionDays: 30,
        SnapshotsToKeep:  3,
        RDSStopAgeDays:   7,
        Regions:          []string{"us-east-1"},
        CostPerGB:        0.08,
        Environment:      "dev",
        LogLevel:         "info",
    }
    err := cfg.Validate()
    assert.NoError(t, err)
}

func TestValidateConfigMissingBucket(t *testing.T) {
    cfg := &config.Config{
        QuarantineDays:   7,
        SnapshotRetentionDays: 30,
        SnapshotsToKeep:  3,
        RDSStopAgeDays:   7,
        Regions:          []string{"us-east-1"},
        CostPerGB:        0.08,
        Environment:      "dev",
        LogLevel:         "info",
    }
    err := cfg.Validate()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "S3_REPORT_BUCKET is required")
}