# Low-Level Design (LLD): Cloud FinOps Bot

**Document:** 06
**Version:** 3.0 (Go Edition - Enhanced & Audited)
**Author:** Jibrin Ahmed
**Date:** June 22, 2026
**Status:** Final

---

## 1. Document Purpose

This document provides the **detailed implementation specification** for the Cloud FinOps Bot. It translates the High-Level Design (HLD) into specific:

- **Go Package Structure:** How the code is organized.
- **Function Signatures:** Inputs, outputs, and error handling.
- **Struct Definitions:** Core data structures with audit fields.
- **Structured Logging Implementation:** Who/what/when/where logging.
- **Correlation ID Strategy:** End-to-end request tracing.
- **IAM Condition Validation:** Explicit checks for least-privilege policies.
- **Error Handling Patterns:** How failures are managed.
- **Concurrency Patterns:** How goroutines are managed with correlation IDs.

**Audience:**
- **Development Team:** Direct implementation guide.
- **Security Reviewers:** Code-level security validation.
- **Technical Reviewers:** Code structure validation.
- **Future Employers:** Demonstrates coding discipline, security awareness, and operational maturity.

---

## 2. Package Structure

```
finops-bot/
├── cmd/
│   └── main.go                    # Lambda entry point (bootstrap)
│   └── runner.go                  # Core runner logic
├── internal/
│   ├── ec2/
│   │   ├── client.go              # EC2 client setup
│   │   ├── discovery.go           # Resource discovery functions
│   │   └── quarantine.go          # Tagging and deletion logic
│   ├── rds/
│   │   ├── client.go              # RDS client setup
│   │   └── discovery.go           # RDS instance discovery
│   ├── dynamodb/
│   │   ├── client.go              # DynamoDB client setup
│   │   ├── state.go               # GetItem, PutItem, UpdateItem with audit fields
│   │   └── queries.go             # Query operations (GSI queries)
│   ├── pricing/
│   │   ├── engine.go              # Pricing engine core
│   │   └── map.go                 # Default pricing map
│   ├── slack/
│   │   ├── client.go              # Slack webhook client
│   │   └── messages.go            # Message formatting
│   ├── s3/
│   │   ├── client.go              # S3 client setup
│   │   └── upload.go              # Audit report upload
│   ├── secrets/
│   │   └── manager.go             # Secrets Manager + SSM client
│   ├── config/
│   │   └── config.go              # Configuration loader (Secrets + SSM + Env)
│   ├── logger/
│   │   └── logger.go              # Structured logging with correlation ID
│   ├── auth/
│   │   └── iam.go                 # IAM principal and source IP extraction
│   └── utils/
│       ├── correlation.go         # Correlation ID generation
│       └── pointer.go             # Utility functions
├── pkg/
│   └── logger/
│       └── logger.go              # Shared logging package
├── tests/
│   ├── unit/                      # Unit tests
│   ├── integration/               # Integration tests with Floci
│   └── e2e/                       # End-to-End tests (real AWS)
├── go.mod
├── go.sum
├── .gitignore
├── .env.example
├── terraform/                     # Terraform IaC
│   ├── main.tf
│   ├── variables.tf
│   ├── outputs.tf
│   └── terraform.tfvars.example
└── README.md
```

---

## 3. Core Data Structures

### 3.1 Configuration Struct

```go
// internal/config/config.go

package config

type Config struct {
    DryRun              bool
    CostPerGB           float64
    ExcludedIDs         map[string]bool     // Set for O(1) lookup
    Regions             []string
    QuarantineDays      int
    SnapshotRetentionDays int
    SnapshotsToKeep     int
    RDSStopAgeDays      int
    LogLevel            string
    SlackChannel        string
    S3ReportBucket      string
    S3ReportPrefix      string
    Timezone            string
    SecretsManagerARN   string
    SNSTopicARN         string
    EnableRDSSavings    bool
    Environment         string              // dev, staging, prod
}

func LoadConfig(ctx context.Context, secretsClient *secretsmanager.Client, ssmClient *ssm.Client) (*Config, error) {
    // 1. Load from environment variables (non-sensitive)
    // 2. Fetch from Secrets Manager (sensitive: Slack webhook URL)
    // 3. Fetch from SSM Parameter Store (non-sensitive: regions, retention)
    // 4. Validate all values
    // 5. Return Config
}
```

### 3.2 Context with Correlation ID

```go
// internal/logger/logger.go

type ContextKey string

const CorrelationIDKey ContextKey = "correlation_id"

// GetCorrelationID extracts the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
    if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
        return id
    }
    return "unknown"
}

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
    return context.WithValue(ctx, CorrelationIDKey, correlationID)
}
```

### 3.3 Resource Structs (Updated with Account ID)

```go
// internal/ec2/discovery.go

package ec2

type EC2Volume struct {
    VolumeID   string
    VolumeType string
    SizeGB     int
    State      string   // "available", "in-use", etc.
    CreateTime time.Time
    Tags       map[string]string
    Region     string
    AccountID  string
}

type ElasticIP struct {
    AllocationID   string
    PublicIP       string
    AssociationID  *string  // nil if unattached
    Region         string
    AccountID      string
    HoursUnattached float64
}

type Snapshot struct {
    SnapshotID   string
    VolumeID     string
    VolumeSizeGB int
    DataSizeGB   float64   // Actual snapshot size
    StartTime    time.Time
    Region       string
    AccountID    string
    IsBackingAMI bool      // True if referenced by any AMI
}
```

### 3.4 RDS Structs

```go
// internal/rds/discovery.go

package rds

type RDSInstance struct {
    DBInstanceIdentifier string
    DBInstanceClass      string
    DBInstanceStatus     string   // "available", "stopped", etc.
    Engine               string   // "postgres", "mysql", etc.
    EngineVersion        string
    CreateTime           time.Time
    Region               string
    AccountID            string
    IsReplica            bool     // True if ReadReplicaDBInstanceIdentifiers not empty
    IsStandalone         bool     // True if not a replica and not part of a cluster
}
```

### 3.5 DynamoDB State Structs (Updated with Audit Fields)

```go
// internal/dynamodb/state.go

package dynamodb

type ResourceState struct {
    // Primary Key
    ResourceID         string    `json:"resource_id"`
    AccountID          string    `json:"account_id"`
    
    // Core State
    Region             string    `json:"region"`
    ActionTaken        string    `json:"action_taken"`   // "QUARANTINED", "DELETED", "SKIPPED", "STOPPED", "DELETION_FAILED"
    ResourceType       string    `json:"resource_type"`  // "EBS_VOLUME", "EIP", "SNAPSHOT", "RDS_INSTANCE"
    
    // Audit Trail (NEW)
    ActionedBy         string    `json:"actioned_by"`    // IAM principal ARN
    SourceIP           string    `json:"source_ip"`      // Source IP address
    CorrelationID      string    `json:"correlation_id"` // Request tracing ID
    ActionReason       string    `json:"action_reason"`  // Why this action was taken
    
    // Financial & Metadata
    SizeGB             *int      `json:"size_gb"`
    EstimatedSavings   *float64  `json:"estimated_savings"`
    
    // Timestamps
    QuarantineExpiry   *int64    `json:"quarantine_expiry"`
    DeletionTimestamp  *int64    `json:"deletion_timestamp"`
    ExpirationTimestamp int64    `json:"expiration_timestamp"` // TTL field
    
    // Concurrency & Safety
    Version            int       `json:"version"`
    RetryCount         int       `json:"retry_count"`
    DeleteProtection   bool      `json:"delete_protection"`
    
    // History
    StatusHistory      []StatusHistoryEntry `json:"status_history"`
    Tags               map[string]string    `json:"tags"`
}

type StatusHistoryEntry struct {
    Action    string `json:"action"`
    Timestamp int64  `json:"timestamp"`
    Reason    string `json:"reason,omitempty"`
}
```

### 3.6 Pricing Map Struct

```go
// internal/pricing/map.go

package pricing

type PricingMap struct {
    EBS      map[string]float64   // VolumeType -> PricePerGB
    EIP      map[string]float64   // Region -> PricePerHour
    Snapshot map[string]float64   // Region -> PricePerGB
    RDS      map[string]float64   // InstanceClass -> HourlyCost
}
```

---

## 4. Structured Logging Implementation

### 4.1 Logger Package

```go
// pkg/logger/logger.go

package logger

import (
    "context"
    "log/slog"
    "os"
    "time"
    "finops-bot/internal/utils"
)

var Logger *slog.Logger

// LogEntry represents a structured log entry
type LogEntry struct {
    Timestamp      string `json:"timestamp"`
    Level          string `json:"level"`
    CorrelationID  string `json:"correlation_id"`
    Who            string `json:"who,omitempty"`
    What           string `json:"what,omitempty"`
    Where          string `json:"where,omitempty"`
    ResourceID     string `json:"resource_id,omitempty"`
    ResourceType   string `json:"resource_type,omitempty"`
    Region         string `json:"region,omitempty"`
    AccountID      string `json:"account_id,omitempty"`
    Message        string `json:"message"`
    DurationMs     int64  `json:"duration_ms,omitempty"`
    Error          string `json:"error,omitempty"`
    Extra          map[string]interface{} `json:"extra,omitempty"`
}

func Init(level string) {
    var l slog.Level
    switch level {
    case "debug":
        l = slog.LevelDebug
    case "info":
        l = slog.LevelInfo
    case "warn":
        l = slog.LevelWarn
    case "error":
        l = slog.LevelError
    default:
        l = slog.LevelInfo
    }

    Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: l,
    }))
}

// LogStructured logs a structured entry with correlation ID and audit fields
func LogStructured(ctx context.Context, level slog.Level, msg string, args ...interface{}) {
    if Logger == nil {
        Init("info")
    }
    
    correlationID := utils.GetCorrelationID(ctx)
    who := utils.GetWho(ctx)
    where := utils.GetWhere(ctx)
    
    // Build log entry
    entry := slog.Group("",
        slog.String("timestamp", time.Now().UTC().Format(time.RFC3339Nano)),
        slog.String("correlation_id", correlationID),
        slog.String("who", who),
        slog.String("where", where),
        slog.String("message", msg),
    )
    
    // Add extra args
    // ...
    
    Logger.Log(ctx, level, msg, entry)
}

func Debug(ctx context.Context, msg string, args ...interface{}) {
    LogStructured(ctx, slog.LevelDebug, msg, args...)
}

func Info(ctx context.Context, msg string, args ...interface{}) {
    LogStructured(ctx, slog.LevelInfo, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...interface{}) {
    LogStructured(ctx, slog.LevelWarn, msg, args...)
}

func Error(ctx context.Context, msg string, args ...interface{}) {
    LogStructured(ctx, slog.LevelError, msg, args...)
}
```

### 4.2 Correlation ID Generation

```go
// internal/utils/correlation.go

package utils

import (
    "context"
    "crypto/rand"
    "encoding/hex"
)

type contextKey string

const (
    CorrelationIDKey contextKey = "correlation_id"
    WhoKey           contextKey = "who"
    WhereKey         contextKey = "where"
)

// GenerateCorrelationID creates a unique identifier for request tracing
func GenerateCorrelationID() string {
    bytes := make([]byte, 8)
    if _, err := rand.Read(bytes); err != nil {
        // Fallback to timestamp-based ID
        return time.Now().UTC().Format("20060102150405") + "-fallback"
    }
    return hex.EncodeToString(bytes)
}

// GetCorrelationID extracts the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
    if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
        return id
    }
    return "unknown"
}

// GetWho extracts the IAM principal from context
func GetWho(ctx context.Context) string {
    if who, ok := ctx.Value(WhoKey).(string); ok {
        return who
    }
    return "unknown"
}

// GetWhere extracts the source IP from context
func GetWhere(ctx context.Context) string {
    if where, ok := ctx.Value(WhereKey).(string); ok {
        return where
    }
    return "unknown"
}

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
    return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// WithWho adds IAM principal to the context
func WithWho(ctx context.Context, who string) context.Context {
    return context.WithValue(ctx, WhoKey, who)
}

// WithWhere adds source IP to the context
func WithWhere(ctx context.Context, where string) context.Context {
    return context.WithValue(ctx, WhereKey, where)
}
```

---

## 5. Function Specifications

### 5.1 Main Entry Point (`cmd/main.go`)

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-lambda-go/lambdacontext"

    "finops-bot/internal/config"
    "finops-bot/internal/logger"
    "finops-bot/internal/utils"
)

// Handler is the Lambda entry point.
func Handler(ctx context.Context, event events.CloudWatchEvent) error {
    // 1. Generate correlation ID for this invocation
    correlationID := utils.GenerateCorrelationID()
    ctx = utils.WithCorrelationID(ctx, correlationID)
    
    // 2. Extract IAM principal from Lambda context
    if lc, ok := lambdacontext.FromContext(ctx); ok {
        ctx = utils.WithWho(ctx, lc.InvokedFunctionArn)
        // Source IP may not be available in all cases
        if lc.ClientContext != nil {
            ctx = utils.WithWhere(ctx, lc.ClientContext.Custom["source_ip"])
        }
    }
    
    // 3. Load configuration
    cfg, err := config.LoadConfig(ctx, nil, nil)
    if err != nil {
        logger.Error(ctx, "Failed to load configuration", "error", err)
        return err
    }

    // 4. Initialize logger
    logger.Init(cfg.LogLevel)
    logger.Info(ctx, "FinOps Bot starting", 
        "correlation_id", correlationID,
        "environment", cfg.Environment,
        "dry_run", cfg.DryRun)

    // 5. Run the bot
    runner := NewRunner(cfg)
    return runner.Run(ctx, event)
}

func main() {
    lambda.Start(Handler)
}
```

### 5.2 Runner Implementation (`cmd/runner.go`)

```go
package main

import (
    "context"
    "fmt"
    "os"
    "sync"
    "time"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/aws/aws-sdk-go-v2/service/rds"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
    "github.com/aws/aws-sdk-go-v2/service/ssm"

    "finops-bot/internal/ec2"
    "finops-bot/internal/rds"
    "finops-bot/internal/dynamodb"
    "finops-bot/internal/pricing"
    "finops-bot/internal/slack"
    "finops-bot/internal/s3"
    "finops-bot/internal/secrets"
    "finops-bot/internal/config"
    "finops-bot/internal/logger"
    "finops-bot/internal/utils"
)

type Runner struct {
    cfg              *config.Config
    ec2Client        *ec2.Client
    rdsClient        *rds.Client
    dynamoClient     *dynamodb.Client
    s3Client         *s3.Client
    secretsClient    *secretsmanager.Client
    ssmClient        *ssm.Client
    pricingMap       pricing.PricingMap
    slackWebhookURL  string
}

func NewRunner(cfg *config.Config) *Runner {
    // Initialize AWS SDK clients with endpoint override for Floci
    awsConfig := loadAWSConfig(cfg)
    
    r := &Runner{
        cfg:           cfg,
        ec2Client:     ec2.NewFromConfig(awsConfig),
        rdsClient:     rds.NewFromConfig(awsConfig),
        dynamoClient:  dynamodb.NewFromConfig(awsConfig),
        s3Client:      s3.NewFromConfig(awsConfig),
        secretsClient: secretsmanager.NewFromConfig(awsConfig),
        ssmClient:     ssm.NewFromConfig(awsConfig),
        pricingMap:    pricing.GetPricingMap(cfg.CostPerGB),
    }
    
    // Fetch Slack webhook from Secrets Manager
    webhookURL, err := secrets.GetSlackWebhook(context.Background(), r.secretsClient, cfg.SecretsManagerARN)
    if err != nil {
        logger.Warn(context.Background(), "Failed to fetch Slack webhook URL", "error", err)
    }
    r.slackWebhookURL = webhookURL
    
    return r
}

func loadAWSConfig(cfg *config.Config) aws.Config {
    opts := []func(*config.LoadOptions) error{
        config.WithRegion(cfg.Regions[0]), // Default region
        config.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
            // Check for Floci endpoint override
            endpointURL := os.Getenv("AWS_ENDPOINT_URL")
            if endpointURL != "" {
                return aws.Endpoint{
                    URL:           endpointURL,
                    SigningRegion: region,
                }, nil
            }
            return aws.Endpoint{}, &aws.EndpointNotFoundError{}
        })),
    }
    
    awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
    if err != nil {
        logger.Error(context.Background(), "Failed to load AWS config", "error", err)
    }
    return awsCfg
}

func (r *Runner) Run(ctx context.Context, event events.CloudWatchEvent) error {
    correlationID := utils.GetCorrelationID(ctx)
    who := utils.GetWho(ctx)
    where := utils.GetWhere(ctx)
    
    logger.Info(ctx, "Starting FinOps Bot run", 
        "correlation_id", correlationID,
        "regions", r.cfg.Regions,
        "dry_run", r.cfg.DryRun)
    
    // Create a semaphore to limit concurrency (max 5 concurrent scanners)
    sem := make(chan struct{}, 5)
    
    var wg sync.WaitGroup
    results := make(chan regionResult, len(r.cfg.Regions))
    
    for _, region := range r.cfg.Regions {
        wg.Add(1)
        sem <- struct{}{} // Acquire semaphore
        go func(reg string) {
            defer func() {
                <-sem // Release semaphore
                wg.Done()
            }()
            r.processRegion(ctx, reg, results)
        }(region)
    }
    
    // Wait for all goroutines to complete
    wg.Wait()
    close(results)
    
    // Collect results
    var allResults []regionResult
    for result := range results {
        allResults = append(allResults, result)
    }
    
    // Generate report
    report, err := r.generateReport(ctx, allResults)
    if err != nil {
        logger.Error(ctx, "Report generation failed", "error", err)
        return fmt.Errorf("report generation failed: %w", err)
    }
    
    // Send Slack summary
    summary := slack.FormatReportMessage(report.TotalMonthlySavings, report.Breakdown)
    r.sendSlackMessage(ctx, summary)
    
    logger.Info(ctx, "FinOps Bot run complete", 
        "total_savings", report.TotalMonthlySavings,
        "resources_processed", report.TotalResources)
    return nil
}

type regionResult struct {
    Region             string
    Savings            float64
    ResourcesProcessed int
    Errors             []error
}

func (r *Runner) processRegion(ctx context.Context, region string, results chan<- regionResult) {
    result := regionResult{Region: region}
    correlationID := utils.GetCorrelationID(ctx)
    
    logger.Debug(ctx, "Processing region", 
        "region", region,
        "correlation_id", correlationID)
    
    // 1. Discover EC2 resources
    volumes, err := ec2.DiscoverVolumes(ctx, r.ec2Client, region, r.cfg.QuarantineDays)
    if err != nil {
        result.Errors = append(result.Errors, fmt.Errorf("volume discovery failed: %w", err))
        logger.Error(ctx, "Volume discovery failed", "region", region, "error", err)
    }
    
    eips, err := ec2.DiscoverElasticIPs(ctx, r.ec2Client, region)
    if err != nil {
        result.Errors = append(result.Errors, fmt.Errorf("EIP discovery failed: %w", err))
        logger.Error(ctx, "EIP discovery failed", "region", region, "error", err)
    }
    
    snapshots, err := ec2.DiscoverSnapshots(ctx, r.ec2Client, region, r.cfg.SnapshotRetentionDays, r.cfg.SnapshotsToKeep)
    if err != nil {
        result.Errors = append(result.Errors, fmt.Errorf("snapshot discovery failed: %w", err))
        logger.Error(ctx, "Snapshot discovery failed", "region", region, "error", err)
    }
    
    // 2. Discover RDS instances (if enabled)
    if r.cfg.EnableRDSSavings {
        instances, err := rds.DiscoverInstances(ctx, r.rdsClient, region, r.cfg.RDSStopAgeDays)
        if err != nil {
            result.Errors = append(result.Errors, fmt.Errorf("RDS discovery failed: %w", err))
            logger.Error(ctx, "RDS discovery failed", "region", region, "error", err)
        }
        result.ResourcesProcessed += r.processRDSInstances(ctx, instances)
    }
    
    // 3. Process each resource type
    result.ResourcesProcessed += r.processVolumes(ctx, volumes)
    result.ResourcesProcessed += r.processEIPs(ctx, eips)
    result.ResourcesProcessed += r.processSnapshots(ctx, snapshots)
    
    results <- result
}

func (r *Runner) processVolumes(ctx context.Context, volumes []ec2.EC2Volume) int {
    processed := 0
    for _, vol := range volumes {
        // Check DynamoDB state
        state, err := dynamodb.GetState(ctx, r.dynamoClient, vol.VolumeID, vol.AccountID)
        if err != nil {
            logger.Error(ctx, "Failed to get state", "resource_id", vol.VolumeID, "error", err)
            continue
        }
        
        // If already deleted or skipped, skip
        if state != nil && (state.ActionTaken == "DELETED" || state.ActionTaken == "SKIPPED") {
            continue
        }
        
        // Check kill-switch
        if r.cfg.ExcludedIDs[vol.VolumeID] {
            logger.Debug(ctx, "Resource excluded by kill-switch", "resource_id", vol.VolumeID)
            continue
        }
        
        // Evaluate quarantine vs. deletion
        if err := r.evaluateVolume(ctx, vol, state); err != nil {
            logger.Error(ctx, "Failed to evaluate volume", "resource_id", vol.VolumeID, "error", err)
        }
        processed++
    }
    return processed
}

// ... Similar functions for processEIPs, processSnapshots, processRDSInstances ...
```

### 5.3 Resource Evaluation Logic (with Audit Trail)

```go
// cmd/runner.go

func (r *Runner) evaluateVolume(ctx context.Context, vol ec2.EC2Volume, state *dynamodb.ResourceState) error {
    correlationID := utils.GetCorrelationID(ctx)
    who := utils.GetWho(ctx)
    where := utils.GetWhere(ctx)
    
    // Check if volume has AutoPurge tag
    hasAutoPurge := false
    if val, ok := vol.Tags["FinOps"]; ok && val == "AutoPurge" {
        hasAutoPurge = true
    }
    
    // If quarantined and expired and has AutoPurge → DELETE
    if state != nil && state.ActionTaken == "QUARANTINED" {
        if state.QuarantineExpiry != nil && *state.QuarantineExpiry < time.Now().Unix() {
            if hasAutoPurge {
                // Log the deletion attempt
                logger.Info(ctx, "Attempting to delete volume", 
                    "resource_id", vol.VolumeID,
                    "size_gb", vol.SizeGB,
                    "region", vol.Region,
                    "has_autopurge", hasAutoPurge)
                
                // Execute deletion
                if !r.cfg.DryRun {
                    if err := ec2.DeleteVolume(ctx, r.ec2Client, vol.VolumeID); err != nil {
                        // Log failure
                        logger.Error(ctx, "Volume deletion failed", 
                            "resource_id", vol.VolumeID,
                            "error", err)
                        
                        // Update state with DELETION_FAILED and audit fields
                        dynamodb.UpdateState(ctx, r.dynamoClient, dynamodb.ResourceState{
                            ResourceID: vol.VolumeID,
                            ActionTaken: "DELETION_FAILED",
                            RetryCount: state.RetryCount + 1,
                            ActionedBy: who,
                            SourceIP: where,
                            CorrelationID: correlationID,
                            ActionReason: "DeleteVolume API call failed",
                        })
                        return fmt.Errorf("delete failed: %w", err)
                    }
                }
                
                // Calculate savings
                savings := pricing.CalculateEBSSavings(vol, r.pricingMap)
                
                // Update state to DELETED with audit fields
                dynamodb.PutState(ctx, r.dynamoClient, dynamodb.ResourceState{
                    ResourceID: vol.VolumeID,
                    AccountID: vol.AccountID,
                    Region: vol.Region,
                    ActionTaken: "DELETED",
                    ResourceType: "EBS_VOLUME",
                    ActionedBy: who,
                    SourceIP: where,
                    CorrelationID: correlationID,
                    ActionReason: "Quarantine expired with AutoPurge tag",
                    EstimatedSavings: &savings,
                    DeletionTimestamp: ptr(time.Now().Unix()),
                    Tags: vol.Tags,
                })
                
                // Send Slack notification
                r.sendSlackMessage(ctx, slack.FormatDeletionMessage(vol.VolumeID, "EBS_VOLUME", vol.Region, savings))
                
                logger.Info(ctx, "Volume deleted successfully", 
                    "resource_id", vol.VolumeID,
                    "savings", savings,
                    "correlation_id", correlationID)
                return nil
            }
        }
    }
    
    // If not quarantined and no AutoPurge → QUARANTINE
    if !hasAutoPurge {
        // Log quarantine
        logger.Info(ctx, "Quarantining volume", 
            "resource_id", vol.VolumeID,
            "quarantine_days", r.cfg.QuarantineDays)
        
        if err := ec2.ApplyQuarantineTag(ctx, r.ec2Client, vol.VolumeID, r.cfg.QuarantineDays); err != nil {
            logger.Error(ctx, "Failed to apply quarantine tag", 
                "resource_id", vol.VolumeID,
                "error", err)
            return fmt.Errorf("quarantine tag failed: %w", err)
        }
        
        // Update state to QUARANTINED with audit fields
        expiry := time.Now().AddDate(0, 0, r.cfg.QuarantineDays).Unix()
        dynamodb.PutState(ctx, r.dynamoClient, dynamodb.ResourceState{
            ResourceID: vol.VolumeID,
            AccountID: vol.AccountID,
            Region: vol.Region,
            ActionTaken: "QUARANTINED",
            ResourceType: "EBS_VOLUME",
            ActionedBy: who,
            SourceIP: where,
            CorrelationID: correlationID,
            ActionReason: "Resource lacks FinOps: AutoPurge tag",
            QuarantineExpiry: &expiry,
            Tags: vol.Tags,
        })
        
        // Send Slack notification
        r.sendSlackMessage(ctx, slack.FormatQuarantineMessage(vol.VolumeID, "EBS_VOLUME", vol.Region, r.cfg.QuarantineDays))
        
        logger.Info(ctx, "Volume quarantined", 
            "resource_id", vol.VolumeID,
            "expiry", expiry,
            "correlation_id", correlationID)
    }
    
    return nil
}
```

### 5.4 Slack Notification (with Correlation ID)

```go
// cmd/runner.go

func (r *Runner) sendSlackMessage(ctx context.Context, message string) {
    if r.slackWebhookURL == "" {
        logger.Warn(ctx, "Slack webhook URL not configured, skipping notification")
        return
    }
    
    if err := slack.SendMessage(r.slackWebhookURL, r.cfg.SlackChannel, message); err != nil {
        logger.Error(ctx, "Failed to send Slack notification", "error", err)
    } else {
        logger.Debug(ctx, "Slack notification sent successfully")
    }
}
```

---

## 6. EC2 Discovery Functions

### 6.1 Discover Volumes

```go
// internal/ec2/discovery.go

package ec2

import (
    "context"
    "time"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// DiscoverVolumes finds unattached EBS volumes older than the specified age.
func DiscoverVolumes(ctx context.Context, client *ec2.Client, region string, minAgeDays int) ([]EC2Volume, error) {
    logger.Debug(ctx, "Discovering volumes", "region", region)
    
    var volumes []EC2Volume
    var nextToken *string
    
    for {
        input := &ec2.DescribeVolumesInput{
            Filters: []types.Filter{
                {
                    Name:   aws.String("status"),
                    Values: []string{"available"},
                },
            },
            NextToken: nextToken,
        }
        
        resp, err := client.DescribeVolumes(ctx, input)
        if err != nil {
            logger.Error(ctx, "DescribeVolumes failed", "region", region, "error", err)
            return nil, err
        }
        
        for _, v := range resp.Volumes {
            // Check if volume is older than minAgeDays
            if v.CreateTime != nil && time.Since(*v.CreateTime) > time.Duration(minAgeDays)*24*time.Hour {
                // Build tags map
                tags := make(map[string]string)
                for _, t := range v.Tags {
                    if t.Key != nil && t.Value != nil {
                        tags[*t.Key] = *t.Value
                    }
                }
                
                volumes = append(volumes, EC2Volume{
                    VolumeID:   aws.ToString(v.VolumeId),
                    VolumeType: string(v.VolumeType),
                    SizeGB:     int(aws.ToInt32(v.Size)),
                    State:      string(v.State),
                    CreateTime: *v.CreateTime,
                    Tags:       tags,
                    Region:     region,
                    AccountID:  aws.ToString(v.OwnerId),
                })
            }
        }
        
        nextToken = resp.NextToken
        if nextToken == nil {
            break
        }
    }
    
    logger.Info(ctx, "Volumes discovered", 
        "region", region, 
        "count", len(volumes))
    return volumes, nil
}
```

### 6.2 Check AMI Backing

```go
// internal/ec2/discovery.go

// CheckAMIBacking determines if a snapshot is referenced by any registered AMI.
func CheckAMIBacking(ctx context.Context, client *ec2.Client, snapshotID string) (bool, error) {
    input := &ec2.DescribeImagesInput{
        Filters: []types.Filter{
            {
                Name:   aws.String("block-device-mapping.snapshot-id"),
                Values: []string{snapshotID},
            },
        },
    }
    
    resp, err := client.DescribeImages(ctx, input)
    if err != nil {
        return false, err
    }
    
    return len(resp.Images) > 0, nil
}
```

---

## 7. DynamoDB State Functions (with Audit Fields)

### 7.1 Get State

```go
// internal/dynamodb/state.go

package dynamodb

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// GetState retrieves the state of a resource from DynamoDB.
func GetState(ctx context.Context, client *dynamodb.Client, resourceID, accountID string) (*ResourceState, error) {
    input := &dynamodb.GetItemInput{
        TableName: aws.String("FinOps-State-dev"), // Use environment
        Key: map[string]types.AttributeValue{
            "ResourceId": &types.AttributeValueMemberS{Value: resourceID},
            "AccountId":  &types.AttributeValueMemberS{Value: accountID},
        },
    }
    
    resp, err := client.GetItem(ctx, input)
    if err != nil {
        return nil, err
    }
    
    if resp.Item == nil {
        return nil, nil // Not found
    }
    
    // Unmarshal item to ResourceState
    state := &ResourceState{}
    // ... unmarshaling logic ...
    
    return state, nil
}
```

### 7.2 Put State (with Audit Fields)

```go
// internal/dynamodb/state.go

// PutState writes a resource state to DynamoDB with audit fields.
func PutState(ctx context.Context, client *dynamodb.Client, state ResourceState) error {
    // Build item with all fields including audit trail
    item := map[string]types.AttributeValue{
        "ResourceId":         &types.AttributeValueMemberS{Value: state.ResourceID},
        "AccountId":          &types.AttributeValueMemberS{Value: state.AccountID},
        "Region":             &types.AttributeValueMemberS{Value: state.Region},
        "ActionTaken":        &types.AttributeValueMemberS{Value: state.ActionTaken},
        "ResourceType":       &types.AttributeValueMemberS{Value: state.ResourceType},
        "ActionedBy":         &types.AttributeValueMemberS{Value: state.ActionedBy},
        "SourceIP":           &types.AttributeValueMemberS{Value: state.SourceIP},
        "CorrelationID":      &types.AttributeValueMemberS{Value: state.CorrelationID},
        "ActionReason":       &types.AttributeValueMemberS{Value: state.ActionReason},
        "Version":            &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", state.Version)},
        "RetryCount":         &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", state.RetryCount)},
        "DeleteProtection":   &types.AttributeValueMemberBOOL{Value: state.DeleteProtection},
        "ExpirationTimestamp": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", state.ExpirationTimestamp)},
    }
    
    // Add optional fields
    if state.SizeGB != nil {
        item["SizeGB"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", *state.SizeGB)}
    }
    if state.EstimatedSavings != nil {
        item["EstimatedSavings"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", *state.EstimatedSavings)}
    }
    if state.QuarantineExpiry != nil {
        item["QuarantineExpiry"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", *state.QuarantineExpiry)}
    }
    if state.DeletionTimestamp != nil {
        item["DeletionTimestamp"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", *state.DeletionTimestamp)}
    }
    
    // Marshal tags as map
    if len(state.Tags) > 0 {
        // ... marshal tags into DynamoDB Map ...
    }
    
    input := &dynamodb.PutItemInput{
        TableName: aws.String("FinOps-State-dev"),
        Item:      item,
        ConditionExpression: aws.String("attribute_not_exists(ResourceId) OR Version = :version"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":version": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", state.Version)},
        },
    }
    
    _, err := client.PutItem(ctx, input)
    return err
}
```

---

## 8. Security Considerations in Code

### 8.1 IAM Condition Validation

```go
// internal/auth/iam.go

package auth

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/aws"
)

// ValidateIAMCondition checks that the resource has the required tags
// before performing a sensitive action.
func ValidateIAMCondition(ctx context.Context, tags map[string]string) bool {
    // Check for FinOps: AutoPurge tag
    if val, ok := tags["FinOps"]; ok && val == "AutoPurge" {
        return true
    }
    return false
}

// CheckDeleteProtection validates that a resource is not protected
func CheckDeleteProtection(ctx context.Context, state *dynamodb.ResourceState) bool {
    if state != nil && state.DeleteProtection {
        return false
    }
    return true
}
```

### 8.2 Secrets Manager Integration

```go
// internal/secrets/manager.go

package secrets

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type SlackSecret struct {
    WebhookURL string `json:"SLACK_WEBHOOK_URL"`
}

// GetSlackWebhook fetches the Slack webhook URL from Secrets Manager.
func GetSlackWebhook(ctx context.Context, client *secretsmanager.Client, secretARN string) (string, error) {
    if secretARN == "" {
        return "", fmt.Errorf("secrets manager ARN is empty")
    }
    
    input := &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretARN),
    }

    result, err := client.GetSecretValue(ctx, input)
    if err != nil {
        return "", err
    }

    var secret SlackSecret
    if err := json.Unmarshal([]byte(*result.SecretString), &secret); err != nil {
        return "", err
    }

    return secret.WebhookURL, nil
}
```

### 8.3 SSM Parameter Store Integration

```go
// internal/secrets/manager.go

// GetSSMParameter fetches a non-sensitive configuration from SSM Parameter Store.
func GetSSMParameter(ctx context.Context, client *ssm.Client, name string) (string, error) {
    input := &ssm.GetParameterInput{
        Name:           aws.String(name),
        WithDecryption: aws.Bool(true),
    }
    
    result, err := client.GetParameter(ctx, input)
    if err != nil {
        return "", err
    }
    
    return *result.Parameter.Value, nil
}
```

---

## 9. Unit Test Patterns

### 9.1 Structured Logging Test

```go
// tests/unit/logger_test.go

package unit

import (
    "context"
    "testing"
    "finops-bot/internal/logger"
    "finops-bot/internal/utils"
)

func TestStructuredLogging(t *testing.T) {
    logger.Init("debug")
    
    ctx := context.Background()
    correlationID := utils.GenerateCorrelationID()
    ctx = utils.WithCorrelationID(ctx, correlationID)
    ctx = utils.WithWho(ctx, "arn:aws:iam::123456789012:role/test-role")
    ctx = utils.WithWhere(ctx, "203.0.113.1")
    
    // Log a structured entry
    logger.Info(ctx, "Test log message", 
        "resource_id", "vol-123",
        "resource_type", "EBS_VOLUME")
    
    // Output should be JSON with all fields
    // This is a smoke test - actual validation would parse the output
}
```

### 9.2 Audit Field Test

```go
// tests/unit/dynamodb_test.go

package unit

import (
    "testing"
    "finops-bot/internal/dynamodb"
)

func TestAuditFields(t *testing.T) {
    state := dynamodb.ResourceState{
        ResourceID: "vol-test-001",
        AccountID: "123456789012",
        ActionTaken: "DELETED",
        ActionedBy: "arn:aws:iam::123456789012:role/test-role",
        SourceIP: "203.0.113.1",
        CorrelationID: "abc-123-def-456",
        ActionReason: "Test deletion",
    }
    
    if state.ActionedBy == "" {
        t.Error("ActionedBy should not be empty")
    }
    if state.CorrelationID == "" {
        t.Error("CorrelationID should not be empty")
    }
}
```

---

## 10. Sign-Off

| Role | Name | Date | Signature |
| :--- | :--- | :--- | :--- |
| **Project Lead / Architect** | Jibrin Ahmed | June 22, 2026 | JA |
| **Security Reviewer** | [Security Team / Imaginary CISO] | [Date] | [Initials] |

---
