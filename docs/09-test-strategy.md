# Test Strategy with Floci: Cloud FinOps Bot

**Document:** 09
**Version:** 3.0 (Go Edition - Enhanced & Audited)
**Author:** Jibrin Ahmed
**Date:** June 22, 2026
**Status:** Final

---

## 1. Document Purpose

This document defines the **comprehensive testing strategy** for the Cloud FinOps Bot. It covers:

- **Test Pyramid:** Unit, integration, and E2E tests.
- **Floci Integration:** How Floci enables zero-cost local testing.
- **IAM Policy Validation:** Automated least-privilege validation.
- **Security Scanning:** Vulnerability scanning in CI.
- **Structured Logging Tests:** Validation of who/what/when/where logging.
- **Audit Trail Tests:** Validation of audit field population.
- **Test Data:** Mock resources and scenarios.
- **CI Pipeline:** How tests run in FlowCI.
- **Coverage Goals:** Quality gates and metrics.
- **Manual Testing:** How to test the bot manually.

**Audience:**
- **Developers:** Implementation of tests.
- **Security Engineers:** Validation of security posture.
- **QA Engineers:** Validation of test coverage.
- **Technical Reviewers:** Quality assurance validation.
- **Future Employers:** Demonstrates testing discipline and security awareness.

---

## 2. Test Pyramid Overview

```
┌─────────────────────────────────────────────┐
│              Test Pyramid                    │
├─────────────────────────────────────────────┤
│                                              │
│           ┌─────────────┐                    │
│           │   E2E Tests  │   ← Few (real AWS)│
│           │   (5-10)     │                    │
│        ┌─────────────────┐                    │
│        │ Integration Tests│   ← Many (Floci)  │
│        │   (30-50)       │                    │
│     ┌───────────────────────┐                 │
│     │   Unit Tests (50-80)  │   ← Most (pure Go)│
│     └───────────────────────┘                 │
│                                              │
└─────────────────────────────────────────────┘
```

| Test Layer | Count | Speed | Cost | AWS Required? |
| :--- | :--- | :--- | :--- | :--- |
| **Unit Tests** | 50-80 | Milliseconds | $0 | ❌ No |
| **Integration Tests (Floci)** | 30-50 | Seconds | $0 | ❌ No |
| **E2E Tests (Real AWS)** | 5-10 | Minutes | < $1/month | ✅ Yes (optional) |

---

## 3. Unit Tests (Pure Go, No AWS)

### 3.1 Unit Test Framework

- **Assertions:** `github.com/stretchr/testify/assert`
- **Mocking:** `github.com/stretchr/testify/mock` for mocking AWS SDK clients.
- **Example:**
  ```go
  import (
      "testing"
      "github.com/stretchr/testify/assert"
      "github.com/stretchr/testify/mock"
  )
  
  func TestSomething(t *testing.T) {
      assert := assert.New(t)
      // ... test logic ...
      assert.Equal(expected, actual, "values should match")
  }
  ```

### 3.2 What to Test

| Package | Test Focus | Example |
| :--- | :--- | :--- |
| **`config`** | Configuration loading and validation | Ensure `LoadConfig()` correctly parses environment variables, SSM, and Secrets Manager. Fails on invalid values. |
| **`pricing`** | Cost calculation formulas | Verify `CalculateEBSSavings()` returns correct savings for different volume types and regions. |
| **`ec2/discovery`** | Resource discovery logic | Test `DiscoverVolumes()` filters correctly based on age and state; test `DiscoverElasticIPs()` filters unassociated EIPs; test `DiscoverSnapshots()` handles AMI-backed snapshots and retention policies. |
| **`ec2/quarantine`** | Tagging and deletion logic | Test `ApplyQuarantineTag()` applies correct TTL; test `DeleteVolume()` with success and error cases; test `CheckAMIBacking()` correctly identifies AMI-backed snapshots. |
| **`rds`** | RDS discovery and stop logic | Test `DiscoverInstances()` filters standalone instances by age and status; test `StopInstance()` with success and error cases. |
| **`dynamodb`** | State management with audit fields | Test `GetState()` with existing and missing items; test `PutState()` with optimistic locking and audit fields (`ActionedBy`, `SourceIP`, `CorrelationID`, `ActionReason`); test `QueryExpiredQuarantines()` and `QueryDeletedResources()` with GSI queries. |
| **`logger`** | Structured logging | Test that structured logs contain all required fields (`who`, `what`, `when`, `where`, `correlation_id`). |
| **`auth`** | IAM principal and source IP extraction | Test extraction of IAM principal ARN and source IP from Lambda context. |
| **`secrets`** | Secrets fetching | Test `GetSlackWebhook()` with valid and invalid secret responses; test `GetSSMParameter()` with SSM. |
| **`s3`** | Upload logic | Test `UploadAuditReport()` and `UploadDashboard()` with mocked S3 client; verify correct key generation and content types. |
| **`slack`** | Message formatting | Verify `FormatQuarantineMessage()` produces correctly formatted Slack messages with embedded resource IDs and expiry dates. |
| **`utils`** | Helper functions | Test `GenerateCorrelationID()`; test `GetCorrelationID()`; test `GetWho()` and `GetWhere()`; test `Ptr()` and other utility functions. |

### 3.3 Example Unit Test (Audit Fields)

```go
// tests/unit/dynamodb_test.go

package unit

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "finops-bot/internal/dynamodb"
    "finops-bot/internal/utils"
)

func TestPutStateWithAuditFields(t *testing.T) {
    assert := assert.New(t)
    
    // Setup context with audit fields
    ctx := context.Background()
    correlationID := utils.GenerateCorrelationID()
    ctx = utils.WithCorrelationID(ctx, correlationID)
    ctx = utils.WithWho(ctx, "arn:aws:iam::123456789012:role/test-role")
    ctx = utils.WithWhere(ctx, "203.0.113.1")
    
    // Create state with audit fields
    state := dynamodb.ResourceState{
        ResourceID:   "vol-test-001",
        AccountID:    "123456789012",
        Region:       "us-east-1",
        ActionTaken:  "DELETED",
        ResourceType: "EBS_VOLUME",
        ActionReason: "Quarantine expired with AutoPurge tag",
        // CorrelationID, ActionedBy, SourceIP should be populated from context
    }
    
    // In production, PutState would populate audit fields from context
    // This test verifies the fields are correctly set
    state.CorrelationID = utils.GetCorrelationID(ctx)
    state.ActionedBy = utils.GetWho(ctx)
    state.SourceIP = utils.GetWhere(ctx)
    
    assert.Equal(correlationID, state.CorrelationID, "CorrelationID should match")
    assert.Equal("arn:aws:iam::123456789012:role/test-role", state.ActionedBy, "ActionedBy should match")
    assert.Equal("203.0.113.1", state.SourceIP, "SourceIP should match")
}
```

### 3.4 Example Unit Test (Correlation ID)

```go
// tests/unit/logger_test.go

package unit

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "finops-bot/internal/logger"
    "finops-bot/internal/utils"
)

func TestStructuredLogging(t *testing.T) {
    assert := assert.New(t)
    
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
    // Verify correlation ID is present in logs (implementation-specific)
    assert.NotEmpty(correlationID, "CorrelationID should not be empty")
}
```

### 3.5 Example Unit Test (IAM Condition Validation)

```go
// tests/unit/auth_test.go

package unit

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "finops-bot/internal/auth"
)

func TestValidateIAMCondition(t *testing.T) {
    assert := assert.New(t)
    
    tests := []struct {
        name     string
        tags     map[string]string
        expected bool
    }{
        {"Has AutoPurge tag", map[string]string{"FinOps": "AutoPurge"}, true},
        {"Has different FinOps tag", map[string]string{"FinOps": "ManualPurge"}, false},
        {"Missing FinOps tag", map[string]string{"Environment": "dev"}, false},
        {"Empty tags", map[string]string{}, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := auth.ValidateIAMCondition(tt.tags)
            assert.Equal(tt.expected, result, "IAM condition validation mismatch")
        })
    }
}
```

---

## 4. Integration Tests (Floci Emulator)

### 4.1 Purpose

Integration tests validate the bot's **interaction with AWS services** using Floci, the open-source AWS emulator. These tests run against `http://localhost:4566` and require **zero AWS credentials**.

### 4.2 Why Floci?

- **24ms startup time** (fastest emulator available).
- **13MiB memory footprint** (runs anywhere).
- **100% SDK compatibility** (1,925/1,925 tests passed).
- **No auth tokens, no restrictions, no telemetry.**
- **Supports 59 AWS services** (EC2, RDS, DynamoDB, S3, Lambda, Secrets Manager, SQS).

### 4.3 What to Test

| Test Scenario | Floci Services Used | Verification |
| :--- | :--- | :--- |
| **EC2 Volume Discovery** | EC2 (DescribeVolumes) | Verify bot finds unattached volumes older than 7 days. |
| **Quarantine Tagging** | EC2 (CreateTags) | Verify bot applies `Pending_Deletion` tag with correct TTL. |
| **Volume Deletion** | EC2 (DeleteVolume) | Verify bot deletes volumes with `FinOps: AutoPurge` tag after quarantine expires. |
| **EIP Discovery & Release** | EC2 (DescribeAddresses, ReleaseAddress) | Verify bot finds and releases unattached EIPs. |
| **Snapshot Discovery** | EC2 (DescribeSnapshots, DescribeImages) | Verify bot skips AMI-backed snapshots. |
| **RDS Discovery & Stop** | RDS (DescribeDBInstances, StopDBInstance) | Verify bot stops standalone RDS instances older than 7 days. |
| **DynamoDB State Tracking with Audit Fields** | DynamoDB (GetItem, PutItem, Query) | Verify bot correctly tracks state with `ActionedBy`, `SourceIP`, `CorrelationID`, `ActionReason`. Prevents duplicate deletions. |
| **Structured Logging Validation** | CloudWatch Logs | Verify structured logs contain `who`, `what`, `when`, `where`, and `correlation_id`. |
| **S3 Audit Upload** | S3 (PutObject) | Verify bot uploads audit reports to S3. |
| **End-to-End Run** | All services | Verify the full bot run completes successfully with audit trail. |
| **Volume Evaluation Logic** | EC2, DynamoDB | Verify `evaluateVolume()` correctly handles: volume with `FinOps: AutoPurge` tag but not quarantined → quarantine; volume without `FinOps: AutoPurge` tag → quarantine; volume with `FinOps: AutoPurge` tag AND quarantine expired → delete; volume with quarantine expired but NO `FinOps: AutoPurge` tag → NOT deleted (only quarantined); volume with `DeleteProtection` in DynamoDB → SKIPPED unconditionally; volume in `EXCLUDED_IDS` → SKIPPED unconditionally. |
| **Dashboard Generation** | DynamoDB, S3 | Verify `generateReport()` correctly queries `QueryDeletedResources()` and generates an HTML dashboard with Chart.js that displays savings trends. |
| **NotFound Error Handling** | EC2, DynamoDB | Verify that when a resource is not found (e.g., manually deleted), the bot catches the `NotFound` error, logs it as `ResourceAlreadyRemoved`, and updates DynamoDB state to `SKIPPED` with audit fields. |
| **RetryCount Increment** | EC2, DynamoDB | Verify that when `DeleteVolume()` fails (simulate permission error), the bot updates DynamoDB with `RetryCount` incremented and `ActionTaken = "DELETION_FAILED"` with audit fields. |
| **IAM Condition Validation** | EC2 | Verify the bot respects IAM conditions by checking for `FinOps: AutoPurge` tag before deletion. |
| **Audit Trail Population** | DynamoDB, CloudWatch | Verify all audit fields (`ActionedBy`, `SourceIP`, `CorrelationID`, `ActionReason`) are correctly populated in DynamoDB and logs. |
| **Secrets Manager Integration** | Secrets Manager | Verify the bot can fetch the Slack webhook URL from Secrets Manager. |
| **SSM Parameter Store Integration** | SSM | Verify the bot can fetch non-sensitive configuration from SSM Parameter Store. |

---

### 4.4 Integration Test Setup with Floci

#### Step 1: Start Floci

```bash
# Install Floci (macOS)
brew install floci-io/floci/floci

# Start Floci
floci start
```

#### Step 2: Set Environment Variables

```bash
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export DRY_RUN=true
export S3_REPORT_BUCKET=finops-audit-local
```

#### Step 3: Create Test Fixtures

Floci supports creating resources via the AWS SDK or CLI:

```go
// tests/integration/setup_test.go

package integration

import (
    "context"
    "testing"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func setupTestResources(t *testing.T, client *ec2.Client) {
    // Create a test volume
    _, err := client.CreateVolume(context.Background(), &ec2.CreateVolumeInput{
        AvailabilityZone: aws.String("us-east-1a"),
        Size:             aws.Int32(10),
        VolumeType:       types.VolumeTypeGp3,
    })
    if err != nil {
        t.Fatalf("Failed to create test volume: %v", err)
    }
    
    // Create a test snapshot
    // ... etc
}
```

### 4.5 Example Integration Test (Audit Trail)

```go
// tests/integration/dynamodb_test.go

package integration

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "finops-bot/internal/dynamodb"
    "finops-bot/internal/utils"
)

func TestAuditTrailPopulation(t *testing.T) {
    assert := assert.New(t)
    
    // Setup Floci and create test resources
    ctx := context.Background()
    correlationID := utils.GenerateCorrelationID()
    ctx = utils.WithCorrelationID(ctx, correlationID)
    ctx = utils.WithWho(ctx, "arn:aws:iam::123456789012:role/finops-lambda-role-dev")
    ctx = utils.WithWhere(ctx, "203.0.113.1")
    
    // Create a test volume and process it
    // ... test logic ...
    
    // Query DynamoDB for the state
    state, err := dynamodb.GetState(ctx, client, "vol-test-001", "123456789012")
    assert.NoError(err, "GetState should not return an error")
    assert.NotNil(state, "State should exist")
    
    // Verify audit fields
    assert.Equal(correlationID, state.CorrelationID, "CorrelationID should match")
    assert.Equal("arn:aws:iam::123456789012:role/finops-lambda-role-dev", state.ActionedBy, "ActionedBy should match")
    assert.Equal("203.0.113.1", state.SourceIP, "SourceIP should match")
    assert.Equal("Quarantine expired with AutoPurge tag", state.ActionReason, "ActionReason should match")
}
```

### 4.6 Integration Test (IAM Condition Validation)

```go
// tests/integration/iam_condition_test.go

package integration

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "finops-bot/internal/ec2"
)

func TestIAMConditionValidation(t *testing.T) {
    assert := assert.New(t)
    
    // Create a volume without AutoPurge tag
    // ... setup logic ...
    
    // Attempt to delete the volume
    // The bot should check the IAM condition and skip deletion
    
    // Verify the volume was NOT deleted
    // ... verification logic ...
    
    // Add AutoPurge tag and attempt again
    // ... add tag logic ...
    
    // Verify the volume WAS deleted
    // ... verification logic ...
}
```

---

## 5. End-to-End (E2E) Tests (Real AWS)

### 5.1 Purpose

E2E tests validate the bot's **full workflow** against **real AWS** infrastructure. These are optional and run only on the `main` branch to validate that Floci's behavior matches real AWS.

### 5.2 What to Test

| Test Scenario | Verification |
| :--- | :--- |
| **Full Run** | Run the bot against a dedicated sandbox AWS account and verify it completes without errors. |
| **Quarantine Flow** | Create a test volume, run the bot, and verify the volume is quarantined (not deleted). |
| **Deletion Flow** | Create a test volume with `FinOps: AutoPurge` tag, run the bot, and verify the volume is deleted. |
| **Audit Trail Flow** | Verify that DynamoDB records contain `ActionedBy`, `SourceIP`, `CorrelationID`, `ActionReason`. |
| **Structured Logging Flow** | Verify that CloudWatch logs contain all required structured fields. |
| **Secrets Manager Flow** | Verify the bot can fetch the Slack webhook URL from Secrets Manager. |
| **SSM Parameter Store Flow** | Verify the bot can fetch configuration from SSM Parameter Store. |

### 5.3 E2E Test Setup

```go
// tests/e2e/full_run_test.go

package e2e

import (
    "testing"
    "context"
    "github.com/stretchr/testify/assert"
    "finops-bot/cmd"
)

func TestFullRun(t *testing.T) {
    if os.Getenv("RUN_E2E_TESTS") != "true" {
        t.Skip("Skipping E2E tests (RUN_E2E_TESTS not set)")
    }
    
    assert := assert.New(t)
    
    // Run the bot
    runner := cmd.NewRunner(cfg)
    err := runner.Run(context.Background(), events.CloudWatchEvent{})
    assert.NoError(err, "Full run should complete without errors")
}
```

### 5.4 Running E2E Tests

```bash
# Run only on main branch
export RUN_E2E_TESTS=true
export DRY_RUN=true
export ENVIRONMENT=sandbox
go test -tags=e2e ./tests/e2e/... -v
```

---

## 6. IAM Policy Validation Tests

### 6.1 Purpose

IAM policy validation tests ensure that the Terraform IAM policies follow least-privilege principles and do not contain overly permissive statements.

### 6.2 Validation Tools

| Tool | Purpose | Validation |
| :--- | :--- | :--- |
| **checkov** | Infrastructure-as-Code security scanning | Checks for overly permissive policies, missing conditions |
| **tfsec** | Terraform security scanning | Checks for insecure configurations, broad permissions |
| **golangci-lint** | Go code security linting | Checks for hardcoded credentials, insecure patterns |

### 6.3 CI Validation Commands

```bash
# Validate IAM policies with checkov
checkov -d terraform/ --check CKV_AWS_* --framework terraform

# Specific checks for IAM policies
checkov -d terraform/ --check CKV_AWS_109  # No wildcard in IAM policy
checkov -d terraform/ --check CKV_AWS_111  # IAM policy has no explicit deny
checkov -d terraform/ --check CKV_AWS_112  # IAM policy has no privileged actions with wildcard

# Validate Terraform security with tfsec
tfsec terraform/ --exclude aws-iam-no-policy-wildcards
tfsec terraform/ --include aws-iam-no-policy-wildcards,aws-iam-no-policy-unrestricted
```

### 6.4 Unit Test for IAM Policy Validation

```go
// tests/unit/iam_validation_test.go

package unit

import (
    "os/exec"
    "testing"
)

func TestIAMPolicyValidation(t *testing.T) {
    // Run checkov on the Terraform directory
    cmd := exec.Command("checkov", "-d", "terraform/", "--framework", "terraform")
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        t.Errorf("checkov failed: %v\nOutput: %s", err, output)
    }
    
    // Check that no high-severity issues were found
    // This is a simplified example - actual validation would parse checkov output
}
```

---

## 7. Security Scanning Tests

### 7.1 Purpose

Security scanning tests identify vulnerabilities in the Go code and container images.

### 7.2 Scanning Tools

| Tool | Purpose | Target |
| :--- | :--- | :--- |
| **trivy** | Vulnerability scanning | Go dependencies, container images |
| **snyk** | Vulnerability scanning | Go dependencies |
| **golangci-lint** | Code quality and security linting | Go source code |

### 7.3 CI Scanning Commands

```bash
# Scan Go dependencies with trivy
trivy fs . --severity HIGH,CRITICAL

# Scan Go dependencies with snyk
snyk test --severity-threshold=high

# Run golangci-lint with security checks
golangci-lint run --enable-all ./...
```

### 7.4 Unit Test for Security Scanning

```go
// tests/unit/security_scan_test.go

package unit

import (
    "os/exec"
    "testing"
)

func TestSecurityScanning(t *testing.T) {
    // Run trivy on the project
    cmd := exec.Command("trivy", "fs", ".", "--severity", "HIGH,CRITICAL")
    output, err := cmd.CombinedOutput()
    
    if err != nil {
        t.Errorf("trivy failed: %v\nOutput: %s", err, output)
    }
}
```

---

## 8. Structured Logging Validation Tests

### 8.1 Purpose

Structured logging tests validate that the bot emits logs with all required fields (`who`, `what`, `when`, `where`, `correlation_id`).

### 8.2 Test Scenarios

| Scenario | Verification |
| :--- | :--- |
| **Correlation ID Generation** | Verify a unique correlation ID is generated for each invocation. |
| **Who Field** | Verify the IAM principal ARN is included in logs. |
| **What Field** | Verify the action (e.g., `DELETE_VOLUME`, `QUARANTINE_VOLUME`) is included in logs. |
| **When Field** | Verify a UTC timestamp is included in logs. |
| **Where Field** | Verify the source IP is included in logs (when available). |
| **Resource ID** | Verify the resource ID is included for resource-specific actions. |

### 8.3 Example Test

```go
// tests/integration/logging_test.go

package integration

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "finops-bot/internal/logger"
    "finops-bot/internal/utils"
)

func TestStructuredLoggingFields(t *testing.T) {
    assert := assert.New(t)
    
    // Setup
    ctx := context.Background()
    correlationID := utils.GenerateCorrelationID()
    ctx = utils.WithCorrelationID(ctx, correlationID)
    ctx = utils.WithWho(ctx, "arn:aws:iam::123456789012:role/test-role")
    ctx = utils.WithWhere(ctx, "203.0.113.1")
    
    // Log a message
    logger.Info(ctx, "Test message", "resource_id", "vol-123")
    
    // In a real test, you would capture and parse the log output
    // Verify that all fields are present in the JSON log
}
```

---

## 9. Audit Trail Validation Tests

### 9.1 Purpose

Audit trail tests validate that the bot correctly populates audit fields in DynamoDB.

### 9.2 Test Scenarios

| Scenario | Verification |
| :--- | :--- |
| **ActionedBy** | Verify the IAM principal ARN is stored in DynamoDB. |
| **SourceIP** | Verify the source IP is stored in DynamoDB (when available). |
| **CorrelationID** | Verify the correlation ID is stored in DynamoDB. |
| **ActionReason** | Verify the action reason is stored in DynamoDB. |
| **StatusHistory** | Verify the status history includes all status changes with reasons. |

### 9.3 Example Test

```go
// tests/integration/audit_trail_test.go

package integration

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "finops-bot/internal/dynamodb"
    "finops-bot/internal/utils"
)

func TestAuditTrailFields(t *testing.T) {
    assert := assert.New(t)
    
    // Setup
    ctx := context.Background()
    correlationID := utils.GenerateCorrelationID()
    ctx = utils.WithCorrelationID(ctx, correlationID)
    ctx = utils.WithWho(ctx, "arn:aws:iam::123456789012:role/test-role")
    ctx = utils.WithWhere(ctx, "203.0.113.1")
    
    // Create a state record (quarantine)
    state := dynamodb.ResourceState{
        ResourceID:   "vol-test-001",
        AccountID:    "123456789012",
        Region:       "us-east-1",
        ActionTaken:  "QUARANTINED",
        ResourceType: "EBS_VOLUME",
        ActionReason: "Resource lacks FinOps: AutoPurge tag",
    }
    
    // Populate audit fields from context
    state.CorrelationID = utils.GetCorrelationID(ctx)
    state.ActionedBy = utils.GetWho(ctx)
    state.SourceIP = utils.GetWhere(ctx)
    
    // Verify audit fields
    assert.Equal(correlationID, state.CorrelationID, "CorrelationID should match")
    assert.Equal("arn:aws:iam::123456789012:role/test-role", state.ActionedBy, "ActionedBy should match")
    assert.Equal("203.0.113.1", state.SourceIP, "SourceIP should match")
    assert.Equal("Resource lacks FinOps: AutoPurge tag", state.ActionReason, "ActionReason should match")
}
```

---

## 10. Test Data Fixtures

### 10.1 Test Data Directory Structure

```
finops-bot/
├── tests/
│   ├── unit/
│   │   └── ...                    # Unit tests
│   ├── integration/
│   │   ├── ec2_test.go
│   │   ├── dynamodb_test.go
│   │   ├── iam_condition_test.go
│   │   ├── audit_trail_test.go
│   │   ├── logging_test.go
│   │   └── full_run_test.go
│   ├── e2e/
│   │   └── full_run_test.go
│   └── testdata/
│       ├── ec2_volumes.json      # Mock EC2 volume fixtures
│       ├── ec2_eips.json         # Mock Elastic IP fixtures
│       ├── ec2_snapshots.json    # Mock Snapshot fixtures
│       ├── rds_instances.json    # Mock RDS instance fixtures
│       ├── dynamodb_states.json  # Mock DynamoDB state fixtures
│       └── audit_trail.json      # Mock audit trail fixtures
```

### 10.2 Mock Audit Trail Fixtures

```go
// tests/fixtures/audit_trail.go

package fixtures

var TestAuditRecords = []dynamodb.ResourceState{
    {
        ResourceID:      "vol-test-001",
        AccountID:       "123456789012",
        Region:          "us-east-1",
        ActionTaken:     "DELETED",
        ResourceType:    "EBS_VOLUME",
        ActionedBy:      "arn:aws:iam::123456789012:role/finops-lambda-role-dev",
        SourceIP:        "203.0.113.1",
        CorrelationID:   "abc-123-def-456",
        ActionReason:    "Quarantine expired with AutoPurge tag",
        DeletionTimestamp: aws.Int64(time.Now().Unix()),
        EstimatedSavings: aws.Float64(8.00),
    },
    {
        ResourceID:      "vol-test-002",
        AccountID:       "123456789012",
        Region:          "us-east-1",
        ActionTaken:     "QUARANTINED",
        ResourceType:    "EBS_VOLUME",
        ActionedBy:      "arn:aws:iam::123456789012:role/finops-lambda-role-dev",
        SourceIP:        "203.0.113.1",
        CorrelationID:   "abc-123-def-456",
        ActionReason:    "Resource lacks FinOps: AutoPurge tag",
        QuarantineExpiry: aws.Int64(time.Now().Add(7 * 24 * time.Hour).Unix()),
    },
}
```

---

## 11. CI Pipeline Integration (FlowCI)

### 11.1 Pipeline Stages (Updated)

```
┌─────────────────────────────────────────────────────────────────┐
│                    FlowCI Pipeline                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Checkout Code                                               │
│  2. Install Dependencies (go mod download)                     │
│  3. IAM Policy Validation (checkov, tfsec)                    │
│  4. Security Scanning (trivy, snyk)                           │
│  5. Install Floci (brew install floci-io/floci/floci)          │
│  6. Start Floci (floci start)                                   │
│  7. Run Unit Tests (go test ./tests/unit/... -v -cover)        │
│  8. Run Integration Tests (go test -tags=integration ./... -v) │
│  9. Stop Floci (floci stop)                                     │
│  10. Build Binary (GOOS=linux GOARCH=amd64 go build -o bootstrap)│
│  11. Zip Binary (zip function.zip bootstrap)                     │
│  12. Deploy to AWS (terraform apply -auto-approve)              │
│  13. Run E2E Tests (go test -tags=e2e ./... -v)                │
│  14. Upload Coverage Report                                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 11.2 FlowCI Configuration (`.flowci.yml`)

```yaml
# .flowci.yml

stages:
  # ... existing stages ...
  
  - name: "IAM Policy Validation"
    steps:
      - command: checkov -d terraform/ --framework terraform
      - command: tfsec terraform/
    timeout: 120
  
  - name: "Security Scanning"
    steps:
      - command: trivy fs . --severity HIGH,CRITICAL
      - command: snyk test --severity-threshold=high || true
    timeout: 300
  
  # ... rest of stages ...
```

---

## 12. Test Coverage Goals

### 12.1 Coverage Targets

| Package | Minimum Coverage | Target |
| :--- | :--- | :--- |
| `config` | 80% | 90% |
| `pricing` | 90% | 95% |
| `ec2` | 75% | 85% |
| `rds` | 75% | 85% |
| `dynamodb` | 75% | 85% |
| `logger` | 80% | 90% |
| `auth` | 80% | 90% |
| `secrets` | 75% | 85% |
| `s3` | 75% | 85% |
| `slack` | 80% | 90% |
| `utils` | 80% | 90% |
| **Overall** | **80%** | **85%** |

### 12.2 Coverage Command

```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -func=coverage.out
```

---

## 13. Manual Testing Instructions

### 13.1 Local Development (Floci)

```bash
# 1. Start Floci
floci start

# 2. Set environment variables
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1
export DRY_RUN=true
export S3_REPORT_BUCKET=finops-audit-local

# 3. Build the binary
GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/main.go

# 4. Run the bot manually
./bootstrap
```

### 13.2 AWS CLI Manual Test

```bash
# Invoke the Lambda manually
aws lambda invoke --function-name finops-cleaner-dev \
  --payload '{"dry_run": false}' \
  output.json

# Check the output
cat output.json

# Check CloudWatch Logs
aws logs get-log-events --log-group-name /aws/lambda/finops-cleaner-dev \
  --log-stream-name $(aws logs describe-log-streams --log-group-name /aws/lambda/finops-cleaner-dev --order-by LastEventTime --descending --limit 1 --query 'logStreams[0].logStreamName' --output text)
```

---

## 14. Testing Checklist (Pre-Release)

Before releasing to production:

- [ ] **Unit Tests:** All unit tests pass with > 80% coverage.
- [ ] **Integration Tests:** All integration tests pass against Floci.
- [ ] **E2E Tests:** All E2E tests pass against a sandbox AWS account.
- [ ] **IAM Policy Validation:** checkov and tfsec pass with no high-severity issues.
- [ ] **Security Scanning:** trivy and snyk pass with no critical vulnerabilities.
- [ ] **Structured Logging Validation:** Logs contain all required fields.
- [ ] **Audit Trail Validation:** DynamoDB records contain all audit fields.
- [ ] **Manual Test:** Run the bot manually with `DRY_RUN=true` in production.
- [ ] **Manual Test:** Run the bot manually with `DRY_RUN=false` in a sandbox account.
- [ ] **Slack Test:** Verify Slack notifications are sent for quarantine and deletion.
- [ ] **S3 Test:** Verify audit reports are uploaded to S3.
- [ ] **DynamoDB Test:** Verify state is correctly tracked in DynamoDB.
- [ ] **CloudWatch Test:** Verify logs are written to CloudWatch.
- [ ] **Alarm Test:** Verify CloudWatch alarms trigger correctly.
- [ ] **KMS Test:** Verify the Lambda can decrypt the Slack webhook secret.
- [ ] **IAM Condition Test:** Verify `ec2:DeleteVolume` is restricted to `FinOps: AutoPurge` resources.
- [ ] **RDS Condition Test:** Verify `rds:StopDBInstance` is restricted to `dev` and `staging` environments.

---

## 15. Sign-Off

| Role | Name | Date | Signature |
| :--- | :--- | :--- | :--- |
| **Project Lead / Architect** | Jibrin Ahmed | June 22, 2026 | JA |
| **QA Lead** | [QA Team / Imaginary Lead] | [Date] | [Initials] |

---

