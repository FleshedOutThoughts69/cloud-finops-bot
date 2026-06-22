// cmd/main.go

package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-lambda-go/lambdacontext"
    "github.com/aws/aws-sdk-go-v2/config"
    awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"

    appconfig "cloud-finops-bot/internal/config"
    "cloud-finops-bot/internal/dynamodb"
    "cloud-finops-bot/internal/ec2"
    "cloud-finops-bot/internal/s3"
    "cloud-finops-bot/internal/secrets"
    "cloud-finops-bot/internal/slack"
    "cloud-finops-bot/internal/utils"
    "cloud-finops-bot/pkg/logger"
)

// Version information injected at build time
var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

// Handler is the Lambda entry point.
func Handler(ctx context.Context, event events.CloudWatchEvent) error {
    // 1. Generate correlation ID
    correlationID := utils.GenerateCorrelationID()
    ctx = logger.WithCorrelationID(ctx, correlationID)

    // 2. Extract IAM principal and IP
    if lc, ok := lambdacontext.FromContext(ctx); ok {
        ctx = logger.WithWho(ctx, lc.InvokedFunctionArn)
        if lc.ClientContext.Custom != nil {
            if ip, ok := lc.ClientContext.Custom["source_ip"]; ok {
                ctx = logger.WithWhere(ctx, ip)
            }
        }
    }

    // 3. Initialize logger
    logLevel := os.Getenv("LOG_LEVEL")
    if logLevel == "" {
        logLevel = "info"
    }
    logger.Init(logLevel)

    // 4. Load configuration
    cfg, err := appconfig.LoadConfig(ctx)
    if err != nil {
        logger.Error(ctx, "Failed to load configuration", "error", err)
        return err
    }

    // 5. Fetch Slack webhook (optional)
    if cfg.SlackWebhookURL == "" && cfg.SecretsManagerARN != "" {
        awsCfg, err := config.LoadDefaultConfig(ctx)
        if err != nil {
            logger.Warn(ctx, "Failed to load AWS config for Secrets Manager", "error", err)
        } else {
            smClient := secretsmanager.NewFromConfig(awsCfg)
            webhook, err := secrets.GetSlackWebhook(ctx, smClient, cfg.SecretsManagerARN)
            if err != nil {
                logger.Warn(ctx, "Failed to fetch Slack webhook from Secrets Manager", "error", err)
            } else {
                cfg.SlackWebhookURL = webhook
            }
        }
    }

    // 6. Log config
    logger.Info(ctx, "Configuration loaded",
        "dry_run", cfg.DryRun,
        "environment", cfg.Environment,
        "regions", cfg.Regions,
        "quarantine_days", cfg.QuarantineDays,
    )

    // 7. Log start
    logger.Info(ctx, "FinOps Bot started",
        "version", Version,
        "build_time", BuildTime,
        "git_commit", GitCommit,
    )

    // 8. DynamoDB client
    tableName := os.Getenv("DYNAMODB_TABLE")
    if tableName == "" {
        tableName = "FinOps-State-dev"
    }

    var dynamoClient *awsdynamodb.Client
    if len(cfg.Regions) > 0 {
        dynamoClient, err = dynamodb.NewClient(ctx, cfg.Regions[0])
        if err != nil {
            logger.Error(ctx, "Failed to create DynamoDB client", "error", err)
            return err
        }
    } else {
        logger.Warn(ctx, "No regions configured, skipping DynamoDB client")
    }

    // 9. EC2 Discovery (enabled if ENABLE_EC2_DISCOVERY=true)
    if os.Getenv("ENABLE_EC2_DISCOVERY") == "true" {
        logger.Info(ctx, "EC2 discovery enabled")
        for _, region := range cfg.Regions {
            logger.Info(ctx, "Scanning region", "region", region)

            ec2Client, err := ec2.NewClient(ctx, region)
            if err != nil {
                logger.Error(ctx, "Failed to create EC2 client", "region", region, "error", err)
                continue
            }

            // Discover volumes
            volumes, err := ec2.DiscoverVolumes(ctx, ec2Client, region, cfg.QuarantineDays)
            if err != nil {
                logger.Error(ctx, "Failed to discover volumes", "region", region, "error", err)
            } else {
                logger.Info(ctx, "Discovered volumes", "region", region, "count", len(volumes))
                for _, vol := range volumes {
                    // Check DynamoDB state
                    if dynamoClient != nil {
                        state, err := dynamodb.GetState(ctx, dynamoClient, tableName, vol.VolumeID, vol.AccountID)
                        if err != nil {
                            logger.Error(ctx, "Failed to get state for volume", "volume_id", vol.VolumeID, "error", err)
                            continue
                        }
                        if state != nil && (state.ActionTaken == "DELETED" || state.ActionTaken == "SKIPPED") {
                            logger.Debug(ctx, "Volume already processed, skipping", "volume_id", vol.VolumeID)
                            continue
                        }
                    }

                    // Check if volume has AutoPurge tag
                    hasAutoPurge := false
                    if val, ok := vol.Tags["FinOps"]; ok && val == "AutoPurge" {
                        hasAutoPurge = true
                    }

                    // If not quarantined and no AutoPurge, quarantine
                    if !hasAutoPurge {
                        // Apply quarantine tag
                        if err := ec2.ApplyQuarantineTag(ctx, ec2Client, vol.VolumeID, "EBS_VOLUME", cfg.QuarantineDays); err != nil {
                            logger.Error(ctx, "Failed to apply quarantine tag", "volume_id", vol.VolumeID, "error", err)
                            continue
                        }

                        // Store state in DynamoDB
                        if dynamoClient != nil {
                            now := time.Now().Unix()
                            expiry := now + int64(cfg.QuarantineDays*24*60*60)
                            state := dynamodb.ResourceState{
                                ResourceID:          vol.VolumeID,
                                AccountID:           vol.AccountID,
                                Region:              vol.Region,
                                ActionTaken:         "QUARANTINED",
                                ResourceType:        "EBS_VOLUME",
                                ActionedBy:          logger.GetWho(ctx),
                                SourceIP:            logger.GetWhere(ctx),
                                CorrelationID:       correlationID,
                                ActionReason:        "Resource lacks FinOps: AutoPurge tag",
                                SizeGB:              &vol.SizeGB,
                                QuarantineExpiry:    &expiry,
                                ExpirationTimestamp: now + 90*24*60*60,
                                Version:             1,
                                RetryCount:          0,
                                DeleteProtection:    false,
                                Tags:                vol.Tags,
                            }
                            if err := dynamodb.PutState(ctx, dynamoClient, tableName, state); err != nil {
                                logger.Error(ctx, "Failed to store quarantine state", "volume_id", vol.VolumeID, "error", err)
                            }
                        }
                    } else {
                        // Volume has AutoPurge tag – check if quarantine expired
                        if dynamoClient != nil {
                            state, err := dynamodb.GetState(ctx, dynamoClient, tableName, vol.VolumeID, vol.AccountID)
                            if err != nil {
                                logger.Error(ctx, "Failed to get state for volume", "volume_id", vol.VolumeID, "error", err)
                                continue
                            }
                            if state != nil && state.ActionTaken == "QUARANTINED" && state.QuarantineExpiry != nil {
                                if *state.QuarantineExpiry < time.Now().Unix() {
                                    // Quarantine expired – delete
                                    if !cfg.DryRun {
                                        if err := ec2.DeleteVolume(ctx, ec2Client, vol.VolumeID); err != nil {
                                            logger.Error(ctx, "Failed to delete volume", "volume_id", vol.VolumeID, "error", err)
                                            // Update state with error
                                            state.ActionTaken = "DELETION_FAILED"
                                            state.RetryCount++
                                            state.ActionReason = "DeleteVolume failed: " + err.Error()
                                            _ = dynamodb.UpdateState(ctx, dynamoClient, tableName, *state)
                                            continue
                                        }
                                        // Update state to DELETED
                                        now := time.Now().Unix()
                                        state.ActionTaken = "DELETED"
                                        state.DeletionTimestamp = &now
                                        state.ActionReason = "Quarantine expired with AutoPurge tag"
                                        state.Version++
                                        if err := dynamodb.UpdateState(ctx, dynamoClient, tableName, *state); err != nil {
                                            logger.Error(ctx, "Failed to update state after deletion", "volume_id", vol.VolumeID, "error", err)
                                        }
                                        logger.Info(ctx, "Volume deleted successfully", "volume_id", vol.VolumeID)
                                    } else {
                                        logger.Info(ctx, "Dry-run: would delete volume", "volume_id", vol.VolumeID)
                                    }
                                }
                            }
                        }
                    }
                }
            }

            // Discover EIPs (placeholder)
            eips, err := ec2.DiscoverElasticIPs(ctx, ec2Client, region)
            if err != nil {
                logger.Error(ctx, "Failed to discover EIPs", "region", region, "error", err)
            } else {
                logger.Info(ctx, "Discovered EIPs", "region", region, "count", len(eips))
                // TODO: Add quarantine/release logic for EIPs
            }

            // Discover Snapshots (placeholder)
            snapshots, err := ec2.DiscoverSnapshots(ctx, ec2Client, region, cfg.SnapshotRetentionDays, cfg.SnapshotsToKeep)
            if err != nil {
                logger.Error(ctx, "Failed to discover snapshots", "region", region, "error", err)
            } else {
                logger.Info(ctx, "Discovered snapshots", "region", region, "count", len(snapshots))
                // TODO: Add quarantine/delete logic for snapshots
            }
        }
    } else {
        logger.Info(ctx, "EC2 discovery disabled. Set ENABLE_EC2_DISCOVERY=true to enable.")
    }

    // 10. Slack notification (conditional)
    if os.Getenv("ENABLE_SLACK") != "false" {
        if cfg.SlackWebhookURL != "" {
            msg := "✅ FinOps Bot started successfully (test notification)"
            if err := slack.Send(cfg.SlackWebhookURL, cfg.SlackChannel, msg); err != nil {
                logger.Error(ctx, "Failed to send Slack notification", "error", err)
            } else {
                logger.Info(ctx, "Slack notification sent")
            }
        } else {
            logger.Warn(ctx, "Slack webhook URL not configured, skipping notification")
        }
    } else {
        logger.Info(ctx, "Slack notifications disabled (ENABLE_SLACK=false)")
    }

    // 11. S3 audit upload (conditional)
    if os.Getenv("ENABLE_S3") != "false" {
        if cfg.S3ReportBucket != "" && len(cfg.Regions) > 0 {
            // Create S3 client
            awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Regions[0]))
            if err != nil {
                logger.Error(ctx, "Failed to load AWS config for S3", "error", err)
            } else {
                s3Client := awss3.NewFromConfig(awsCfg)
                testData := []byte(`{"test": "FinOps Bot started", "timestamp": "` + time.Now().UTC().Format(time.RFC3339) + `"}`)
                uri, err := s3.UploadAuditReport(ctx, s3Client, cfg.S3ReportBucket, cfg.S3ReportPrefix, testData)
                if err != nil {
                    logger.Error(ctx, "Failed to upload audit report", "error", err)
                } else {
                    logger.Info(ctx, "Audit report uploaded", "uri", uri)
                }
            }
        } else {
            logger.Warn(ctx, "S3 bucket or region not configured, skipping upload")
        }
    } else {
        logger.Info(ctx, "S3 upload disabled (ENABLE_S3=false)")
    }

    // 12. Dashboard generation (conditional)
    if os.Getenv("ENABLE_DASHBOARD") != "false" {
        if cfg.S3ReportBucket != "" && len(cfg.Regions) > 0 && dynamoClient != nil {
            // Query deleted resources from the last 30 days
            startTime := time.Now().AddDate(0, -1, 0).Unix()
            deletedRecords, err := dynamodb.QueryDeletedResources(ctx, dynamoClient, tableName, startTime)
            if err != nil {
                logger.Error(ctx, "Failed to query deleted resources for dashboard", "error", err)
            } else {
                // Calculate total savings
                totalSavings := 0.0
                for _, r := range deletedRecords {
                    if r.EstimatedSavings != nil {
                        totalSavings += *r.EstimatedSavings
                    }
                }

                // Health status (simple: healthy if records exist, else warning)
                healthStatus := "warning"
                if len(deletedRecords) > 0 {
                    healthStatus = "healthy"
                }

                // Create S3 client for dashboard upload
                awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Regions[0]))
                if err != nil {
                    logger.Error(ctx, "Failed to load AWS config for S3 dashboard", "error", err)
                } else {
                    s3Client := awss3.NewFromConfig(awsCfg)
                    lastUpdated := time.Now()
                    uri, err := s3.UploadDashboard(ctx, s3Client, cfg.S3ReportBucket, cfg.S3ReportPrefix, deletedRecords, totalSavings, lastUpdated, healthStatus)
                    if err != nil {
                        logger.Error(ctx, "Failed to upload dashboard", "error", err)
                    } else {
                        logger.Info(ctx, "Dashboard uploaded", "uri", uri)
                    }
                }
            }
        } else {
            logger.Warn(ctx, "S3 bucket or region or DynamoDB not configured, skipping dashboard upload")
        }
    } else {
        logger.Info(ctx, "Dashboard generation disabled (ENABLE_DASHBOARD=false)")
    }

    // 13. Log completion
    logger.Info(ctx, "FinOps Bot completed successfully")

    return nil
}

func main() {
    if os.Getenv("AWS_EXECUTION_ENV") != "" || os.Getenv("LAMBDA_TASK_ROOT") != "" {
        lambda.Start(Handler)
    } else {
        ctx := context.Background()
        if err := Handler(ctx, events.CloudWatchEvent{}); err != nil {
            log.Fatalf("Handler failed: %v", err)
        }
    }
}