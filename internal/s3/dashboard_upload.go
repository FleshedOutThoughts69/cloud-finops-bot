// internal/s3/dashboard_upload.go

package s3

import (
    "context"
    "fmt"
    "strings"
    "time" // <-- ADDED

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"

    "cloud-finops-bot/internal/dashboard"
    "cloud-finops-bot/internal/dynamodb"
    "cloud-finops-bot/internal/utils"
    "cloud-finops-bot/pkg/logger"
)

// UploadDashboard generates and uploads the HTML dashboard to S3.
func UploadDashboard(ctx context.Context, client *s3.Client, bucket, prefix string, records []dynamodb.ResourceState, totalSavings float64, lastUpdated time.Time, healthStatus string) (string, error) {
    if bucket == "" {
        return "", fmt.Errorf("bucket name is empty")
    }

    html := dashboard.GenerateDashboardHTML(records, totalSavings, lastUpdated, healthStatus)

    // Use a versioned filename for cache busting with correlation ID suffix
    correlationID := utils.GenerateCorrelationID()
    version := lastUpdated.Format("20060102-150405") + "-" + correlationID[:8]
    versionedKey := fmt.Sprintf("%sdashboard-%s.html", prefix, version)

    // Upload versioned file (no-cache)
    _, err := client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:       aws.String(bucket),
        Key:          aws.String(versionedKey),
        Body:         strings.NewReader(html),
        ContentType:  aws.String("text/html"),
        CacheControl: aws.String("no-cache"),
    })
    if err != nil {
        return "", fmt.Errorf("failed to upload versioned dashboard: %w", err)
    }

    // Also upload to index.html (cache for 1 hour)
    _, err = client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:       aws.String(bucket),
        Key:          aws.String(prefix + "index.html"),
        Body:         strings.NewReader(html),
        ContentType:  aws.String("text/html"),
        CacheControl: aws.String("max-age=3600"),
    })
    if err != nil {
        return "", fmt.Errorf("failed to upload index dashboard: %w", err)
    }

    logger.Info(ctx, "Dashboard uploaded",
        "bucket", bucket,
        "versioned_key", versionedKey,
    )

    return fmt.Sprintf("s3://%s/%s", bucket, versionedKey), nil
}