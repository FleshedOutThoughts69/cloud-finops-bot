// cmd/health_check.go

package main

import (
    "context"
    "os"
    "time"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
    "github.com/aws/aws-sdk-go-v2/service/ssm"

    "cloud-finops-bot/internal/utils"
    "cloud-finops-bot/pkg/logger"
)

// Version information
var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

// HealthStatus represents the overall health check result.
type HealthStatus struct {
    Healthy       bool
    Checks        []CheckResult
    CorrelationID string
}

// CheckResult represents a single health check.
type CheckResult struct {
    Name    string
    Healthy bool
    Error   string
    Duration time.Duration
}

// Handler is the Lambda entry point for health checks.
func Handler(ctx context.Context, event events.CloudWatchEvent) error {
    // 1. Generate correlation ID
    correlationID := utils.GenerateCorrelationID()
    ctx = logger.WithCorrelationID(ctx, correlationID)

    // 2. Initialize logger
    logLevel := os.Getenv("LOG_LEVEL")
    if logLevel == "" {
        logLevel = "info"
    }
    logger.Init(logLevel)

    // 3. Log start
    logger.Info(ctx, "Health check started",
        "version", Version,
        "build_time", BuildTime,
        "git_commit", GitCommit,
    )

    // 4. Load AWS config
    region := os.Getenv("AWS_DEFAULT_REGION")
    if region == "" {
        region = "us-east-1"
    }
    awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
    if err != nil {
        logger.Error(ctx, "Failed to load AWS config", "error", err)
        // Emit unhealthy metric and return
        emitHealthMetric(ctx, false)
        return err
    }

    // 5. Run health checks
    status := runChecks(ctx, awsCfg)

    // 6. Emit CloudWatch metric
    emitHealthMetric(ctx, status.Healthy)

    // 7. Log summary
    if status.Healthy {
        logger.Info(ctx, "Health check passed", "correlation_id", correlationID)
    } else {
        logger.Warn(ctx, "Health check failed", "correlation_id", correlationID)
        for _, check := range status.Checks {
            if !check.Healthy {
                logger.Error(ctx, "Check failed",
                    "check", check.Name,
                    "error", check.Error,
                    "duration_ms", check.Duration.Milliseconds(),
                )
            }
        }
    }

    return nil
}

// runChecks executes all health checks.
func runChecks(ctx context.Context, awsCfg config.Config) HealthStatus {
    status := HealthStatus{
        Healthy: true,
        Checks:  []CheckResult{},
    }

    // 1. DynamoDB check
    status.Checks = append(status.Checks, checkDynamoDB(ctx, awsCfg))

    // 2. S3 check
    status.Checks = append(status.Checks, checkS3(ctx, awsCfg))

    // 3. Secrets Manager check
    status.Checks = append(status.Checks, checkSecretsManager(ctx, awsCfg))

    // 4. SSM check
    status.Checks = append(status.Checks, checkSSM(ctx, awsCfg))

    // Determine overall health
    for _, check := range status.Checks {
        if !check.Healthy {
            status.Healthy = false
            break
        }
    }

    return status
}

// checkDynamoDB verifies connectivity to DynamoDB.
func checkDynamoDB(ctx context.Context, awsCfg config.Config) CheckResult {
    start := time.Now()
    result := CheckResult{Name: "DynamoDB"}

    client := dynamodb.NewFromConfig(awsCfg)
    tableName := os.Getenv("DYNAMODB_TABLE")
    if tableName == "" {
        tableName = "FinOps-State-dev"
    }

    input := &dynamodb.DescribeTableInput{
        TableName: &tableName,
    }

    _, err := client.DescribeTable(ctx, input)
    result.Duration = time.Since(start)
    if err != nil {
        result.Healthy = false
        result.Error = err.Error()
    } else {
        result.Healthy = true
    }

    return result
}

// checkS3 verifies connectivity to S3.
func checkS3(ctx context.Context, awsCfg config.Config) CheckResult {
    start := time.Now()
    result := CheckResult{Name: "S3"}

    client := s3.NewFromConfig(awsCfg)
    bucket := os.Getenv("S3_REPORT_BUCKET")
    if bucket == "" {
        bucket = "finops-audit-local" // fallback
    }

    input := &s3.HeadBucketInput{
        Bucket: &bucket,
    }

    _, err := client.HeadBucket(ctx, input)
    result.Duration = time.Since(start)
    if err != nil {
        result.Healthy = false
        result.Error = err.Error()
    } else {
        result.Healthy = true
    }

    return result
}

// checkSecretsManager verifies connectivity to Secrets Manager.
func checkSecretsManager(ctx context.Context, awsCfg config.Config) CheckResult {
    start := time.Now()
    result := CheckResult{Name: "SecretsManager"}

    client := secretsmanager.NewFromConfig(awsCfg)
    secretARN := os.Getenv("SECRETS_MANAGER_ARN")
    if secretARN == "" {
        // If no secret ARN is configured, skip the check (pass)
        result.Healthy = true
        result.Duration = time.Since(start)
        return result
    }

    input := &secretsmanager.DescribeSecretInput{
        SecretId: &secretARN,
    }

    _, err := client.DescribeSecret(ctx, input)
    result.Duration = time.Since(start)
    if err != nil {
        result.Healthy = false
        result.Error = err.Error()
    } else {
        result.Healthy = true
    }

    return result
}

// checkSSM verifies connectivity to SSM Parameter Store.
func checkSSM(ctx context.Context, awsCfg config.Config) CheckResult {
    start := time.Now()
    result := CheckResult{Name: "SSM"}

    client := ssm.NewFromConfig(awsCfg)
    // Use a known parameter path (or a dummy)
    paramName := os.Getenv("SSM_PARAM_PATH")
    if paramName == "" {
        // If no SSM parameter is configured, skip the check (pass)
        result.Healthy = true
        result.Duration = time.Since(start)
        return result
    }

    input := &ssm.GetParameterInput{
        Name: &paramName,
    }

    _, err := client.GetParameter(ctx, input)
    result.Duration = time.Since(start)
    if err != nil {
        result.Healthy = false
        result.Error = err.Error()
    } else {
        result.Healthy = true
    }

    return result
}

// emitHealthMetric sends a CloudWatch metric for health status.
func emitHealthMetric(ctx context.Context, healthy bool) {
    // We'll use the CloudWatch API to put a metric.
    // For simplicity, we'll log that we would emit it.
    // In a real implementation, we'd call cloudwatch.PutMetricData.
    value := 0
    if !healthy {
        value = 1
    }
    logger.Info(ctx, "Health metric would be emitted",
        "metric", "HealthCheckStatus",
        "value", value,
    )
}

func main() {
    lambda.Start(Handler)
}