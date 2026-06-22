// internal/ec2/discovery.go

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

// DiscoverVolumes finds unattached EBS volumes older than the specified age.
func DiscoverVolumes(ctx context.Context, client *ec2.Client, region string, minAgeDays int) ([]EC2Volume, error) {
    logger.Debug(ctx, "Discovering volumes", "region", region, "min_age_days", minAgeDays)

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
            return nil, fmt.Errorf("DescribeVolumes failed: %w", err)
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

                volume := EC2Volume{
                    VolumeID:   aws.ToString(v.VolumeId),
                    VolumeType: string(v.VolumeType),
                    SizeGB:     int(aws.ToInt32(v.Size)),
                    State:      string(v.State),
                    CreateTime: *v.CreateTime,
                    Tags:       tags,
                    Region:     region,
                    AccountID:  "", // OwnerId not available in DescribeVolumes response
                }
                volumes = append(volumes, volume)

                logger.Debug(ctx, "Found candidate volume",
                    "volume_id", volume.VolumeID,
                    "size_gb", volume.SizeGB,
                    "state", volume.State,
                    "age_days", int(time.Since(volume.CreateTime).Hours()/24),
                )
            }
        }

        nextToken = resp.NextToken
        if nextToken == nil {
            break
        }
    }

    logger.Info(ctx, "Volumes discovered",
        "region", region,
        "count", len(volumes),
        "min_age_days", minAgeDays,
    )
    return volumes, nil
}

// DiscoverElasticIPs finds unassociated Elastic IPs.
func DiscoverElasticIPs(ctx context.Context, client *ec2.Client, region string) ([]ElasticIP, error) {
    logger.Debug(ctx, "Discovering Elastic IPs", "region", region)

    var eips []ElasticIP

    input := &ec2.DescribeAddressesInput{}

    resp, err := client.DescribeAddresses(ctx, input)
    if err != nil {
        logger.Error(ctx, "DescribeAddresses failed", "region", region, "error", err)
        return nil, fmt.Errorf("DescribeAddresses failed: %w", err)
    }

    for _, addr := range resp.Addresses {
        // Check if EIP is unassociated (no AssociationId or NetworkInterfaceId)
        isAssociated := addr.AssociationId != nil || addr.NetworkInterfaceId != nil

        if !isAssociated {
            eip := ElasticIP{
                AllocationID:   aws.ToString(addr.AllocationId),
                PublicIP:       aws.ToString(addr.PublicIp),
                AssociationID:  nil,
                Region:         region,
                AccountID:      aws.ToString(addr.NetworkInterfaceOwnerId), // Use NetworkInterfaceOwnerId as fallback
                HoursUnattached: 0,
            }

            eips = append(eips, eip)

            logger.Debug(ctx, "Found candidate EIP",
                "allocation_id", eip.AllocationID,
                "public_ip", eip.PublicIP,
            )
        }
    }

    logger.Info(ctx, "Elastic IPs discovered",
        "region", region,
        "count", len(eips),
    )
    return eips, nil
}

// DiscoverSnapshots finds snapshots older than the specified retention period,
// excluding those that back an AMI and preserving the latest N per volume.
func DiscoverSnapshots(ctx context.Context, client *ec2.Client, region string, retentionDays int, keepCount int) ([]Snapshot, error) {
    logger.Debug(ctx, "Discovering snapshots",
        "region", region,
        "retention_days", retentionDays,
        "keep_count", keepCount,
    )

    var snapshots []Snapshot
    var nextToken *string

    for {
        input := &ec2.DescribeSnapshotsInput{
            OwnerIds: []string{"self"},
            NextToken: nextToken,
        }

        resp, err := client.DescribeSnapshots(ctx, input)
        if err != nil {
            logger.Error(ctx, "DescribeSnapshots failed", "region", region, "error", err)
            return nil, fmt.Errorf("DescribeSnapshots failed: %w", err)
        }

        for _, s := range resp.Snapshots {
            // Check if snapshot is older than retentionDays
            if s.StartTime != nil && time.Since(*s.StartTime) > time.Duration(retentionDays)*24*time.Hour {
                // Use VolumeSize as the data size approximation (AWS doesn't return DataSize separately)
                var dataSizeGB float64
                if s.VolumeSize != nil {
                    dataSizeGB = float64(aws.ToInt32(s.VolumeSize))
                } else {
                    dataSizeGB = 0
                }

                snap := Snapshot{
                    SnapshotID:   aws.ToString(s.SnapshotId),
                    VolumeID:     aws.ToString(s.VolumeId),
                    VolumeSizeGB: int(aws.ToInt32(s.VolumeSize)),
                    DataSizeGB:   dataSizeGB,
                    StartTime:    *s.StartTime,
                    Region:       region,
                    AccountID:    aws.ToString(s.OwnerId),
                    IsBackingAMI: false,
                }

                // Check if snapshot is AMI-backed
                isBackingAMI, err := checkAMIBacking(ctx, client, snap.SnapshotID)
                if err != nil {
                    logger.Warn(ctx, "Failed to check AMI backing for snapshot",
                        "snapshot_id", snap.SnapshotID,
                        "error", err,
                    )
                    snap.IsBackingAMI = false
                } else {
                    snap.IsBackingAMI = isBackingAMI
                }

                if snap.IsBackingAMI {
                    logger.Debug(ctx, "Skipping AMI-backed snapshot",
                        "snapshot_id", snap.SnapshotID,
                    )
                    continue
                }

                snapshots = append(snapshots, snap)

                logger.Debug(ctx, "Found candidate snapshot",
                    "snapshot_id", snap.SnapshotID,
                    "volume_id", snap.VolumeID,
                    "size_gb", snap.VolumeSizeGB,
                    "data_size_gb", snap.DataSizeGB,
                    "age_days", int(time.Since(snap.StartTime).Hours()/24),
                    "is_ami_backed", snap.IsBackingAMI,
                )
            }
        }

        nextToken = resp.NextToken
        if nextToken == nil {
            break
        }
    }

    // Keep the latest N per volume
    snapshots = filterLatestSnapshotsPerVolume(snapshots, keepCount)

    logger.Info(ctx, "Snapshots discovered",
        "region", region,
        "total_candidates", len(snapshots),
        "retention_days", retentionDays,
        "keep_count", keepCount,
    )
    return snapshots, nil
}

// checkAMIBacking determines if a snapshot is referenced by any registered AMI.
func checkAMIBacking(ctx context.Context, client *ec2.Client, snapshotID string) (bool, error) {
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
        return false, fmt.Errorf("DescribeImages failed: %w", err)
    }

    return len(resp.Images) > 0, nil
}

// filterLatestSnapshotsPerVolume keeps only the latest N snapshots per volume.
func filterLatestSnapshotsPerVolume(snapshots []Snapshot, keepCount int) []Snapshot {
    if keepCount <= 0 {
        return snapshots
    }

    // Group snapshots by volume ID
    volumeSnapshots := make(map[string][]Snapshot)
    for _, s := range snapshots {
        volumeSnapshots[s.VolumeID] = append(volumeSnapshots[s.VolumeID], s)
    }

    // For each volume, sort by StartTime descending and keep top N
    var filtered []Snapshot
    for volumeID, snaps := range volumeSnapshots {
        if len(snaps) <= keepCount {
            filtered = append(filtered, snaps...)
            continue
        }

        // Sort by StartTime descending (newest first)
        for i := 0; i < len(snaps); i++ {
            for j := i + 1; j < len(snaps); j++ {
                if snaps[i].StartTime.Before(snaps[j].StartTime) {
                    snaps[i], snaps[j] = snaps[j], snaps[i]
                }
            }
        }

        // Keep only the first keepCount snapshots
        for i := 0; i < keepCount && i < len(snaps); i++ {
            filtered = append(filtered, snaps[i])
        }

        logger.Debug(context.Background(), "Filtered snapshots for volume",
            "volume_id", volumeID,
            "total", len(snaps),
            "kept", min(keepCount, len(snaps)),
            "removed", max(0, len(snaps)-keepCount),
        )
    }

    return filtered
}

// Helper functions
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}