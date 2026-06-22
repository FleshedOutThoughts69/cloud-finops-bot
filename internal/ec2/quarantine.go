// internal/ec2/quarantine.go

package ec2

import (
    "context"
    "fmt"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/aws/aws-sdk-go-v2/service/ec2/types"

    "cloud-finops-bot/pkg/logger"
)

// ApplyQuarantineTag applies a Pending_Deletion tag to a resource.
func ApplyQuarantineTag(ctx context.Context, client *ec2.Client, resourceID, resourceType string, days int) error {
    expiry := time.Now().AddDate(0, 0, days).Unix()
    tagValue := fmt.Sprintf("%d", expiry)

    logger.Debug(ctx, "Applying quarantine tag",
        "resource_id", resourceID,
        "resource_type", resourceType,
        "expiry", expiry,
    )

    input := &ec2.CreateTagsInput{
        Resources: []string{resourceID},
        Tags: []types.Tag{
            {
                Key:   aws.String("Pending_Deletion"),
                Value: aws.String(tagValue),
            },
        },
    }

    _, err := client.CreateTags(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to apply quarantine tag: %w", err)
    }

    logger.Info(ctx, "Quarantine tag applied",
        "resource_id", resourceID,
        "resource_type", resourceType,
        "expiry_days", days,
    )
    return nil
}

// DeleteVolume deletes an EBS volume.
func DeleteVolume(ctx context.Context, client *ec2.Client, volumeID string) error {
    logger.Debug(ctx, "Deleting volume", "volume_id", volumeID)

    input := &ec2.DeleteVolumeInput{
        VolumeId: aws.String(volumeID),
    }

    _, err := client.DeleteVolume(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to delete volume: %w", err)
    }

    logger.Info(ctx, "Volume deleted", "volume_id", volumeID)
    return nil
}

// DeleteSnapshot deletes an EBS snapshot.
func DeleteSnapshot(ctx context.Context, client *ec2.Client, snapshotID string) error {
    logger.Debug(ctx, "Deleting snapshot", "snapshot_id", snapshotID)

    input := &ec2.DeleteSnapshotInput{
        SnapshotId: aws.String(snapshotID),
    }

    _, err := client.DeleteSnapshot(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to delete snapshot: %w", err)
    }

    logger.Info(ctx, "Snapshot deleted", "snapshot_id", snapshotID)
    return nil
}

// ReleaseElasticIP releases an Elastic IP.
func ReleaseElasticIP(ctx context.Context, client *ec2.Client, allocationID string) error {
    logger.Debug(ctx, "Releasing EIP", "allocation_id", allocationID)

    input := &ec2.ReleaseAddressInput{
        AllocationId: aws.String(allocationID),
    }

    _, err := client.ReleaseAddress(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to release EIP: %w", err)
    }

    logger.Info(ctx, "EIP released", "allocation_id", allocationID)
    return nil
}