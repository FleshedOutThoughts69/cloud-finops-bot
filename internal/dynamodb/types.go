// internal/dynamodb/types.go

package dynamodb

// ResourceState represents the state of a resource in DynamoDB.
type ResourceState struct {
    ResourceID          string            `json:"resource_id"`
    AccountID           string            `json:"account_id"`
    Region              string            `json:"region"`
    ActionTaken         string            `json:"action_taken"` // QUARANTINED, DELETED, SKIPPED, STOPPED, DELETION_FAILED
    ResourceType        string            `json:"resource_type"` // EBS_VOLUME, EIP, SNAPSHOT, RDS_INSTANCE
    ActionedBy          string            `json:"actioned_by"`
    SourceIP            string            `json:"source_ip"`
    CorrelationID       string            `json:"correlation_id"`
    ActionReason        string            `json:"action_reason"`
    SizeGB              *int              `json:"size_gb,omitempty"`
    EstimatedSavings    *float64          `json:"estimated_savings,omitempty"`
    QuarantineExpiry    *int64            `json:"quarantine_expiry,omitempty"`
    DeletionTimestamp   *int64            `json:"deletion_timestamp,omitempty"`
    ExpirationTimestamp int64             `json:"expiration_timestamp"`
    Version             int               `json:"version"`
    RetryCount          int               `json:"retry_count"`
    DeleteProtection    bool              `json:"delete_protection"`
    Tags                map[string]string `json:"tags,omitempty"`
}