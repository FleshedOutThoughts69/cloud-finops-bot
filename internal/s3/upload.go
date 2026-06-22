// internal/s3/upload.go

package s3

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"

    "cloud-finops-bot/pkg/logger"
)

// UploadAuditReport uploads a JSON audit report to S3.
func UploadAuditReport(ctx context.Context, client *s3.Client, bucket, prefix string, data []byte) (string, error) {
    if bucket == "" {
        return "", fmt.Errorf("bucket name is empty")
    }

    timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
    key := fmt.Sprintf("%saudit-%s.json", prefix, timestamp)

    logger.Debug(ctx, "Uploading audit report", "bucket", bucket, "key", key)

    input := &s3.PutObjectInput{
        Bucket:      aws.String(bucket),
        Key:         aws.String(key),
        Body:        strings.NewReader(string(data)),
        ContentType: aws.String("application/json"),
    }

    _, err := client.PutObject(ctx, input)
    if err != nil {
        return "", fmt.Errorf("failed to upload audit report: %w", err)
    }

    s3URI := fmt.Sprintf("s3://%s/%s", bucket, key)
    logger.Info(ctx, "Audit report uploaded", "uri", s3URI)
    return s3URI, nil
}