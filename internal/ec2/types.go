// internal/ec2/types.go

package ec2

import "time"

// EC2Volume represents an EBS volume.
type EC2Volume struct {
    VolumeID   string
    VolumeType string
    SizeGB     int
    State      string            // "available", "in-use", etc.
    CreateTime time.Time
    Tags       map[string]string
    Region     string
    AccountID  string
}

// ElasticIP represents an Elastic IP address.
type ElasticIP struct {
    AllocationID   string
    PublicIP       string
    AssociationID  *string // nil if unattached
    Region         string
    AccountID      string
    HoursUnattached float64
}

// Snapshot represents an EBS snapshot.
type Snapshot struct {
    SnapshotID   string
    VolumeID     string
    VolumeSizeGB int
    DataSizeGB   float64 // Actual snapshot data size
    StartTime    time.Time
    Region       string
    AccountID    string
    IsBackingAMI bool // True if referenced by any AMI
}