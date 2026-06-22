
# DynamoDB Schema: Cloud FinOps Bot
**Document:** 03
**Version:** 3.0 (Go Edition - Enhanced & Audited)
**Author:** Jibrin Ahmed
**Date:** June 22, 2026
**Status:** Final

---

## 1. Document Purpose

This document defines the **exact structure** of the DynamoDB table used by the FinOps Bot to track resource states. It serves as the single source of truth for:

- **Idempotency:** Preventing duplicate deletions.
- **Audit Trail:** Recording every action taken by the bot with who/what/when/where.
- **Cost Reporting:** Providing historical data for the static HTML dashboard.
- **Operational Safety:** Enabling manual queries to inspect resource history.
- **Security Observability:** Tracking IAM principals and correlation IDs for incident response.
- **Incident Response:** Supporting forensic analysis during security incidents.

---

## 2. Table Overview

| Attribute | Value |
| :--- | :--- |
| **Table Name** | `FinOps-State` |
| **Billing Mode** | `PAY_PER_REQUEST` (On-Demand) – Cost-effective for low-traffic, daily batch jobs. |
| **Partition Key** | `ResourceId` (String) – Unique AWS resource identifier (e.g., `vol-12345`, `eipalloc-67890`). |
| **Sort Key** | `AccountId` (String) – AWS account ID (e.g., `123456789012`). For v1, this will always be a single value (e.g., `"default"`). |
| **TTL (Time-to-Live)** | Enabled on `ExpirationTimestamp` – Auto-deletes records after 90 days to control storage costs. |

---

## 3. Attribute Definitions

### 3.1 Primary Key Attributes

| Attribute Name | Data Type | Required? | Description |
| :--- | :--- | :--- | :--- |
| `ResourceId` | String (Partition Key) | **Yes** | Unique AWS resource ID. Example: `vol-0abc123def456`. |
| `AccountId` | String (Sort Key) | **Yes** | AWS account ID. Example: `123456789012`. |

### 3.2 Core State Attributes

| Attribute Name | Data Type | Required? | Description |
| :--- | :--- | :--- | :--- |
| `Region` | String | **Yes** | AWS region where the resource resides. Example: `us-east-1`. |
| `ActionTaken` | String | **Yes** | Action performed. Values: `QUARANTINED`, `DELETED`, `SKIPPED`, `STOPPED` (for RDS), `DELETION_FAILED`. |
| `ResourceType` | String | **Yes** | Type of AWS resource. Values: `EBS_VOLUME`, `EIP`, `SNAPSHOT`, `RDS_INSTANCE`. |

### 3.3 Audit Trail Attributes (NEW - Who/What/When/Where)

| Attribute Name | Data Type | Required? | Description |
| :--- | :--- | :--- | :--- |
| `ActionedBy` | String | **Yes** | IAM principal ARN that performed the action. Example: `arn:aws:sts::123456789012:assumed-role/finops-lambda-role-dev/finops-cleaner`. |
| `SourceIP` | String | **Optional** | Source IP address of the request (if available). Example: `203.0.113.1`. |
| `CorrelationID` | String | **Yes** | Unique identifier for the Lambda invocation. Enables end-to-end request tracing. Example: `abc-123-def-456`. |
| `ActionReason` | String | **Optional** | Reason for the action. Example: `Quarantine expired with AutoPurge tag` or `Resource lacks FinOps: AutoPurge tag`. |

### 3.4 Financial & Metadata Attributes

| Attribute Name | Data Type | Required? | Description |
| :--- | :--- | :--- | :--- |
| `SizeGB` | Number | **Conditional** | Size of the resource in GB (for EBS volumes and snapshots). |
| `EstimatedSavings` | Float | **Conditional** | Estimated monthly savings in USD. Populated for `DELETED` and `STOPPED` actions. |

### 3.5 Timestamp Attributes

| Attribute Name | Data Type | Required? | Description |
| :--- | :--- | :--- | :--- |
| `DeletionTimestamp` | Number | **Conditional** | Unix epoch (seconds) when the resource was deleted/stopped. Populated only when `ActionTaken` is `DELETED` or `STOPPED`. |
| `ExpirationTimestamp` | Number (TTL Field) | **Yes** | Unix epoch (seconds) when the record should be auto-deleted. Set to `DeletionTimestamp + 90 days`. If `ActionTaken` is `QUARANTINED`, this remains null or set to 7 days. |
| `QuarantineExpiry` | Number | **Conditional** | Unix epoch when the 7-day quarantine expires. Populated only when `ActionTaken` is `QUARANTINED`. |

### 3.6 Concurrency & Safety Attributes

| Attribute Name | Data Type | Required? | Description |
| :--- | :--- | :--- | :--- |
| `Version` | Number | **Yes** | Optimistic locking version. Incremented on every write. Prevents concurrent modifications. |
| `RetryCount` | Number | **Optional** | Number of consecutive failed deletion attempts. Incremented on failure; reset on success. Triggers alert if > 3. |
| `DeleteProtection` | Boolean | **Optional** | If `true`, the bot will skip this resource unconditionally. Allows manual override without restarting the Lambda. |

### 3.7 History & Tags Attributes

| Attribute Name | Data Type | Required? | Description |
| :--- | :--- | :--- | :--- |
| `StatusHistory` | List of Maps | **Optional** | Tracks status changes over time. Example: `[{"Action": "QUARANTINED", "Timestamp": 1750874400, "Reason": "No AutoPurge tag"}, {"Action": "DELETED", "Timestamp": 1750874401, "Reason": "Quarantine expired"}]` |
| `Tags` | Map (String -> String) | **Optional** | Snapshot of the resource's tags at the time of action (useful for debugging). |

---

## 4. Global Secondary Indexes (GSIs)

| GSI Name | Partition Key | Sort Key | Purpose |
| :--- | :--- | :--- | :--- |
| `ActionTaken-Index` | `ActionTaken` | `DeletionTimestamp` (Number) | Allows efficient queries for "all DELETED resources" or "all QUARANTINED resources" for reporting and dashboard generation. |
| `Region-Index` | `Region` | `ResourceId` | Enables region-specific queries (e.g., "show me all resources processed in us-east-1"). |
| `QuarantineExpiry-Index` | `ActionTaken` | `QuarantineExpiry` | Enables efficient queries for "all QUARANTINED resources where QuarantineExpiry < now()" for deletion processing. |
| `DeleteProtection-Index` | `DeleteProtection` | `ResourceId` | Enables queries for "all resources with DeleteProtection = true" for manual auditing. |
| **`ActionedBy-Index` (NEW)** | `ActionedBy` | `DeletionTimestamp` | Enables forensic queries for "all actions performed by a specific IAM principal" during incident response. |

---

## 5. Data Access Patterns (Query Scenarios)

The Lambda function will perform the following queries against the DynamoDB table:

| Pattern | Operation | Purpose |
| :--- | :--- | :--- |
| **Check Idempotency** | `GetItem` with `ResourceId` + `AccountId` | Before acting on a resource, check if it already exists in the table with `ActionTaken` = `DELETED` or `SKIPPED`. If so, skip processing. |
| **Quarantine Cleanup** | `PutItem` (or `UpdateItem`) | When a resource is identified as a zombie and lacks the `FinOps: AutoPurge` tag, write/update the record with `ActionTaken` = `QUARANTINED` and set `QuarantineExpiry` to `now + 7 days`. Include audit fields (`ActionedBy`, `SourceIP`, `CorrelationID`, `ActionReason`). |
| **Record Deletion** | `PutItem` (or `UpdateItem`) with Conditional Write | After successfully deleting/stopping a resource, update the record with `ActionTaken` = `DELETED`/`STOPPED`, set `DeletionTimestamp`, and calculate `EstimatedSavings`. Use `Version` for optimistic locking. Include all audit fields. |
| **Generate Dashboard Report** | `Query` on `ActionTaken-Index` | At the end of each run, query all `DELETED` resources from the last 30 days to generate the static HTML dashboard with savings trends. |
| **Find Expired Quarantines** | `Query` on `QuarantineExpiry-Index` | Query for all `QUARANTINED` resources where `QuarantineExpiry < now()` to determine which resources are ready for deletion. |
| **Handle NotFound Errors** | `PutItem` with `ActionTaken` = `SKIPPED` | If a resource is already manually deleted (AWS API returns `NotFound`), write a `SKIPPED` record to prevent repeated error logs. Include audit fields. |
| **Check DeleteProtection** | `GetItem` | Before attempting deletion, check if `DeleteProtection` is `true`. If so, skip unconditionally. |
| **Forensic Audit (NEW)** | `Query` on `ActionedBy-Index` | During incident response, query all actions performed by a specific IAM principal to investigate potential privilege escalation or unauthorized access. |

---

## 6. Example Items

### 6.1 Quarantined Resource (with Audit Fields)

```json
{
  "ResourceId": "vol-0abc123def456",
  "AccountId": "123456789012",
  "Region": "us-east-1",
  "ActionTaken": "QUARANTINED",
  "ResourceType": "EBS_VOLUME",
  "ActionedBy": "arn:aws:sts::123456789012:assumed-role/finops-lambda-role-dev/finops-cleaner",
  "SourceIP": "203.0.113.1",
  "CorrelationID": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "ActionReason": "Resource lacks FinOps: AutoPurge tag",
  "SizeGB": 50,
  "QuarantineExpiry": 1750874400,
  "EstimatedSavings": null,
  "DeletionTimestamp": null,
  "ExpirationTimestamp": null,
  "Version": 1,
  "DeleteProtection": false,
  "RetryCount": 0,
  "StatusHistory": [
    {"Action": "QUARANTINED", "Timestamp": 1750269600, "Reason": "Resource lacks FinOps: AutoPurge tag"}
  ],
  "Tags": {
    "Environment": "dev",
    "CreatedBy": "jibrin"
  }
}
```

### 6.2 Deleted Resource (with Audit Fields)

```json
{
  "ResourceId": "vol-0abc123def456",
  "AccountId": "123456789012",
  "Region": "us-east-1",
  "ActionTaken": "DELETED",
  "ResourceType": "EBS_VOLUME",
  "ActionedBy": "arn:aws:sts::123456789012:assumed-role/finops-lambda-role-dev/finops-cleaner",
  "SourceIP": "203.0.113.1",
  "CorrelationID": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "ActionReason": "Quarantine expired with AutoPurge tag",
  "SizeGB": 50,
  "DeletionTimestamp": 1750874400,
  "EstimatedSavings": 4.00,
  "ExpirationTimestamp": 1753466400,
  "QuarantineExpiry": null,
  "Version": 2,
  "DeleteProtection": false,
  "RetryCount": 0,
  "StatusHistory": [
    {"Action": "QUARANTINED", "Timestamp": 1750269600, "Reason": "Resource lacks FinOps: AutoPurge tag"},
    {"Action": "DELETED", "Timestamp": 1750874400, "Reason": "Quarantine expired with AutoPurge tag"}
  ],
  "Tags": {
    "Environment": "dev",
    "CreatedBy": "jibrin",
    "FinOps": "AutoPurge"
  }
}
```

### 6.3 Skipped Resource (Already Manually Deleted)

```json
{
  "ResourceId": "vol-0xyz789ghi012",
  "AccountId": "123456789012",
  "Region": "us-west-2",
  "ActionTaken": "SKIPPED",
  "ResourceType": "EBS_VOLUME",
  "ActionedBy": "arn:aws:sts::123456789012:assumed-role/finops-lambda-role-dev/finops-cleaner",
  "SourceIP": "203.0.113.1",
  "CorrelationID": "a1b2c3d4-e5f6-7890-abcd-ef1234567891",
  "ActionReason": "Resource already manually deleted (NotFound)",
  "SizeGB": null,
  "DeletionTimestamp": 1750874400,
  "EstimatedSavings": null,
  "ExpirationTimestamp": 1753466400,
  "QuarantineExpiry": null,
  "Version": 1,
  "DeleteProtection": false,
  "RetryCount": 0,
  "StatusHistory": [
    {"Action": "SKIPPED", "Timestamp": 1750874400, "Reason": "Resource already manually deleted (NotFound)"}
  ],
  "Tags": {}
}
```

### 6.4 Deletion Failed Resource (with Audit Fields)

```json
{
  "ResourceId": "vol-0def456ghi789",
  "AccountId": "123456789012",
  "Region": "us-east-1",
  "ActionTaken": "DELETION_FAILED",
  "ResourceType": "EBS_VOLUME",
  "ActionedBy": "arn:aws:sts::123456789012:assumed-role/finops-lambda-role-dev/finops-cleaner",
  "SourceIP": "203.0.113.1",
  "CorrelationID": "a1b2c3d4-e5f6-7890-abcd-ef1234567892",
  "ActionReason": "DeleteVolume API call failed: AccessDenied",
  "SizeGB": 20,
  "DeletionTimestamp": null,
  "EstimatedSavings": null,
  "ExpirationTimestamp": null,
  "QuarantineExpiry": 1750874400,
  "Version": 2,
  "DeleteProtection": false,
  "RetryCount": 1,
  "StatusHistory": [
    {"Action": "QUARANTINED", "Timestamp": 1750269600, "Reason": "Resource lacks FinOps: AutoPurge tag"},
    {"Action": "DELETION_FAILED", "Timestamp": 1750874400, "Reason": "DeleteVolume API call failed: AccessDenied"}
  ],
  "Tags": {
    "Environment": "dev",
    "CreatedBy": "jibrin"
  }
}
```

---

## 7. Billing & Cost Considerations

| Aspect | Configuration | Rationale |
| :--- | :--- | :--- |
| **Billing Mode** | `PAY_PER_REQUEST` (On-Demand) | The bot runs once per day. On-Demand is cheaper than Provisioned for low-throughput workloads. |
| **TTL (Time-to-Live)** | Enabled on `ExpirationTimestamp` | Automatically deletes records 90 days after deletion. Prevents storage costs from growing indefinitely. |
| **Read/Write Capacity** | N/A (On-Demand) | No manual capacity planning required. AWS automatically scales. |
| **Expected Monthly Cost** | < $0.50 | Based on 1 Lambda invocation per day (~30 writes, ~30 reads, minimal storage). |
| **GSI Cost** | Additional GSIs add cost | The `ActionedBy-Index` adds minimal cost as it's only queried during incident response (rare). |

---

## 8. Go SDK Integration Notes

In your Go Lambda, you will interact with the DynamoDB table using the AWS SDK for Go v2 (`aws-sdk-go-v2/service/dynamodb`).

### 8.1 Key Code Patterns

| Operation | Go SDK Method | Input |
| :--- | :--- | :--- |
| Check Idempotency | `GetItem` | Primary Key: `ResourceId` + `AccountId` |
| Add/Update Record | `UpdateItem` with Conditional Expression | Use `Version` for optimistic locking. Include all audit fields. |
| Query Expired Quarantines | `Query` on `QuarantineExpiry-Index` | `ActionTaken = 'QUARANTINED' AND QuarantineExpiry < now()` |
| Query for Dashboard | `Query` on `ActionTaken-Index` | `ActionTaken = 'DELETED' AND DeletionTimestamp > <date>` |
| Forensic Audit (NEW) | `Query` on `ActionedBy-Index` | `ActionedBy = '<IAM_ARN>' AND DeletionTimestamp > <date>` |

### 8.2 Audit Field Population

```go
// internal/dynamodb/state.go

func PutState(ctx context.Context, client *dynamodb.Client, state ResourceState) error {
    // Get audit fields from context
    correlationID := utils.GetCorrelationID(ctx)
    who := utils.GetWho(ctx)
    where := utils.GetWhere(ctx)
    
    // Populate audit fields
    state.CorrelationID = correlationID
    state.ActionedBy = who
    state.SourceIP = where
    
    // ... rest of PutState logic ...
}
```

### 8.3 Best Practices

- Always use `context.Context` to respect Lambda timeouts:
  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 250*time.Second)
  defer cancel()
  
  result, err := dynamodbClient.GetItem(ctx, &dynamodb.GetItemInput{...})
  ```

- Use `Version` for optimistic locking to prevent concurrent modifications:
  ```go
  ConditionExpression: aws.String("attribute_not_exists(ResourceId) OR Version = :version")
  ```

- Include all audit fields for every write operation to ensure full traceability.

---

## 9. Future Extensibility (v2 Considerations)

| Potential Future Feature | Schema Impact |
| :--- | :--- | :--- |
| **Multi-Account Support** | The `AccountId` Sort Key already supports this out of the box. |
| **Cost Explorer Integration** | Add `ActualCostSaved` (Float) to store real billing data. |
| **Retention Policy Customization** | Add `RetentionDays` (Number) as a configurable attribute (default 90). |
| **Detailed Audit Trail** | Expand `StatusHistory` to include more fields (e.g., `SourceIP`, `CorrelationID`). |
| **Compliance Reporting** | Add `ComplianceStatus` (String) for regulatory reporting. |

---

## 10. Incident Response Queries

During an incident, the following queries are useful:

```sql
-- 1. Find all actions performed by a specific IAM principal
SELECT * FROM "FinOps-State"
WHERE ActionedBy = 'arn:aws:sts::123456789012:assumed-role/finops-lambda-role-dev/finops-cleaner'
AND DeletionTimestamp > unix_timestamp(now() - interval 7 day)

-- 2. Find all DELETE actions in the last 24 hours
SELECT * FROM "FinOps-State"
WHERE ActionTaken = 'DELETED'
AND DeletionTimestamp > unix_timestamp(now() - interval 1 day)

-- 3. Find all DELETION_FAILED resources
SELECT * FROM "FinOps-State"
WHERE ActionTaken = 'DELETION_FAILED'
AND RetryCount > 3

-- 4. Find all resources with DeleteProtection = true
SELECT * FROM "FinOps-State"
WHERE DeleteProtection = true

-- 5. Correlation ID tracing
SELECT * FROM "FinOps-State"
WHERE CorrelationID = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'
```

---

## 11. Sign-Off

| Role | Name | Date | Signature |
| :--- | :--- | :--- | :--- |
| **Project Lead / Architect** | Jibrin Ahmed | June 22, 2026 | JA |
| **Security Reviewer** | [Security Team / Imaginary CISO] | [Date] | [Initials] |

---
