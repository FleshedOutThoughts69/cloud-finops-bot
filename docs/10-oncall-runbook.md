Here is the **fully patched version of Document 10: On-Call Runbook** with all critical gaps fixed, including the **Disaster Recovery Strategy**, updated incident drill scenarios, and integration with the new Health Check and CloudWatch Dashboard.

---

# On-Call Runbook: Cloud FinOps Bot

**Document:** 10
**Version:** 3.1 (Go Edition - Fully Patched)
**Author:** Jibrin Ahmed
**Date:** June 22, 2026
**Status:** Final

---

## 1. Document Purpose

This document defines the **incident response procedures** for the Cloud FinOps Bot. It is designed for on-call engineers who need to:

- **Troubleshoot** failures quickly.
- **Recover** the bot to normal operation.
- **Escalate** issues when necessary.
- **Manually intervene** when the bot is misbehaving.
- **Respond to security incidents** (compromised bot, broad IAM permissions, secrets exposure).
- **Conduct incident drills** to test preparedness.
- **Execute Disaster Recovery** procedures for regional failure, data corruption, and service degradation.

**Audience:**
- **On-Call Engineers:** First responders.
- **SRE Team:** Escalation contacts.
- **Security Team:** Incident response coordination.
- **Future Employers:** Demonstrates operational maturity and security readiness.

---

## 2. Quick Reference

### 2.1 Emergency Contacts

| Role | Name | Contact |
| :--- | :--- | :--- |
| **Primary On-Call** | [Your Name] | [Your Email/Phone] |
| **Secondary On-Call** | [Secondary Name] | [Secondary Email/Phone] |
| **Escalation (SRE)** | [SRE Team] | [SRE Email/Phone] |
| **Security Team** | [Security Team] | [Security Email/Phone] |
| **AWS Support** | AWS Enterprise Support | [AWS Support Console] |

### 2.2 Critical Services Summary

| Service | Purpose | Status Check |
| :--- | :--- | :--- |
| **Lambda (Main)** | Executes the bot logic | Check CloudWatch for errors |
| **Lambda (Health)** | Health check monitoring | Check CloudWatch dashboard |
| **EventBridge** | Triggers the bot daily | Check rule status |
| **DynamoDB** | State tracking with audit fields | Check table health |
| **S3** | Audit logs and dashboard | Check bucket access |
| **SQS** | Dead-letter queue | Check DLQ depth |
| **Secrets Manager** | Slack webhook URL | Check secret exists and is valid |
| **KMS** | Encryption for secrets | Check key availability |
| **SSM Parameter Store** | Non-sensitive configuration | Check parameter availability |
| **CloudWatch Dashboard** | Operational monitoring | Check dashboard widgets |

### 2.3 Key Commands (One-Liners)

**Prerequisite:** Ensure `AWS_DEFAULT_REGION` is set (e.g., `us-east-1`) or add `--region us-east-1` to all commands.

```bash
# === EMERGENCY STOP ===
# Disable the bot immediately
aws events disable-rule --name finops-daily-trigger-dev --region us-east-1

# Set DRY_RUN to true immediately
aws lambda update-function-configuration --function-name finops-cleaner-dev \
  --environment "Variables={DRY_RUN=true}" --region us-east-1

# Add critical resources to EXCLUDED_IDS
aws lambda update-function-configuration --function-name finops-cleaner-dev \
  --environment "Variables={EXCLUDED_IDS=vol-123,vol-456}" --region us-east-1

# === HEALTH CHECKS ===
# Check CloudWatch Dashboard (pre-configured)
# Navigate to: CloudWatch → Dashboards → FinOps-Bot-Dashboard-dev

# Check Health Check Lambda status
aws lambda get-function --function-name finops-health-check-dev --query 'Configuration.State' --output text --region us-east-1

# Check Health Check alarm
aws cloudwatch describe-alarms --alarm-names finops-health-check-alarm-dev --region us-east-1

# Comprehensive health check
echo "=== Lambda Status ==="
aws lambda get-function --function-name finops-cleaner-dev --query 'Configuration.State' --output text --region us-east-1

echo "=== EventBridge Status ==="
aws events describe-rule --name finops-daily-trigger-dev --query 'State' --output text --region us-east-1

echo "=== DLQ Depth ==="
aws sqs get-queue-attributes --queue-url $(aws sqs get-queue-url --queue-name finops-dlq-dev --query QueueUrl --output text --region us-east-1) --attribute-names ApproximateNumberOfMessages --query 'Attributes.ApproximateNumberOfMessages' --output text --region us-east-1

echo "=== DynamoDB Status ==="
aws dynamodb describe-table --table-name FinOps-State-dev --query 'Table.TableStatus' --output text --region us-east-1

echo "=== Secrets Manager Status ==="
aws secretsmanager describe-secret --secret-id finops/slack-webhook-dev --query 'DeletedDate' --output text --region us-east-1

echo "=== Last Invocation ==="
aws logs get-log-events --log-group-name /aws/lambda/finops-cleaner-dev --limit 1 --order-by LastEventTime --descending --query 'logEvents[0].timestamp' --output text --region us-east-1 | xargs -I{} date -d @{} '+%Y-%m-%d %H:%M:%S'

# === SECURITY INCIDENT ===
# Rotate Slack webhook secret
aws secretsmanager update-secret --secret-id finops/slack-webhook-dev \
  --secret-string '{"SLACK_WEBHOOK_URL":"https://hooks.slack.com/services/NEW_WEBHOOK"}' --region us-east-1

# Query audit trail for specific IAM principal
aws dynamodb query --table-name FinOps-State-dev --index-name ActionedBy-Index \
  --key-condition-expression "ActionedBy = :arn" \
  --expression-attribute-values '{":arn":{"S":"arn:aws:iam::123456789012:role/finops-lambda-role-dev"}}' \
  --region us-east-1

# Query CloudWatch Logs Insights for correlation ID
aws logs start-query --log-group-name /aws/lambda/finops-cleaner-dev \
  --query-string "fields @timestamp, level, message, correlation_id, who, what, resource_id | filter correlation_id = 'abc-123-def-456' | sort @timestamp asc" \
  --start-time $(date -d '24 hours ago' +%s) --end-time $(date +%s) --region us-east-1

# === RECOVERY ===
# Re-enable the bot
aws events enable-rule --name finops-daily-trigger-dev --region us-east-1

# Set DRY_RUN back to false (only after validation)
aws lambda update-function-configuration --function-name finops-cleaner-dev \
  --environment "Variables={DRY_RUN=false}" --region us-east-1
```

---

## 3. Incident Response Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Incident Response Flow                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Alert Received (Slack, Email, PagerDuty)                   │
│     ├── Severity: P1 (Critical) - Immediate action             │
│     ├── Severity: P2 (High) - Respond within 1 hour            │
│     └── Severity: P3 (Low) - Respond within 24 hours           │
│                                                                 │
│  2. Acknowledge the Alert (to prevent escalation)              │
│                                                                 │
│  3. Assess the Impact: Does the bot need to be stopped?        │
│     ├── If YES: Disable EventBridge rule OR set DRY_RUN=true   │
│     └── If NO: Continue troubleshooting                        │
│                                                                 │
│  4. Investigate: Check logs, metrics, and state                │
│     ├── Check CloudWatch Logs (structured logs)               │
│     ├── Check CloudWatch Dashboard (operational metrics)       │
│     ├── Check Health Check Lambda (service connectivity)       │
│     ├── Check DynamoDB State (audit trail)                     │
│     ├── Check CloudTrail (API calls)                           │
│     └── Identify root cause                                    │
│                                                                 │
│  5. Take Action: Apply fix, retry, or escalate                 │
│     ├── Security Incident: Rotate secrets, revoke access      │
│     ├── Operational Incident: Fix configuration, retry        │
│     ├── Disaster Recovery: Execute DR procedure               │
│     └── Escalate to Security Team / AWS Support               │
│                                                                 │
│  6. Resolve: Confirm bot is healthy and re-enable if stopped   │
│                                                                 │
│  7. Document: Write post-mortem and update this runbook        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Incident Severity Levels

| Severity | Description | Response Time | Example |
| :--- | :--- | :--- | :--- |
| **P1 (Critical)** | Bot is deleting resources incorrectly OR security incident confirmed OR regional failure | Immediate | Unauthorized deletion of production resources, compromised bot, AWS region outage |
| **P2 (High)** | Bot is failing to run OR critical alerts firing | 1 hour | Lambda timeouts, DLQ overflowing, health check failing |
| **P3 (Medium)** | Bot is partially working OR non-critical failures | 4 hours | Slack notifications failing, S3 upload failing |
| **P4 (Low)** | Informational or minor issues | 24 hours | Warning logs, low test coverage |

---

## 5. Incident Drill Scenarios

### 5.1 Drill Scenario: Compromised Bot

**Scenario:** The bot is compromised and attempting to delete resources it shouldn't.

**Detection:**
1. CloudWatch Alarm triggers for unexpected deletions.
2. DynamoDB audit trail shows deletions of resources without `FinOps: AutoPurge` tag.
3. Slack alerts show unexpected activity.
4. Health Check may show degradation.

**Response Steps:**
1. **Immediate Stop:**
   ```bash
   aws events disable-rule --name finops-daily-trigger-dev --region us-east-1
   aws lambda update-function-configuration --function-name finops-cleaner-dev --environment "Variables={DRY_RUN=true}" --region us-east-1
   ```

2. **Investigate:**
   ```bash
   # Query audit trail for suspicious activity
   aws dynamodb query --table-name FinOps-State-dev --index-name ActionedBy-Index \
     --key-condition-expression "ActionedBy = :arn" \
     --expression-attribute-values '{":arn":{"S":"arn:aws:iam::123456789012:role/finops-lambda-role-dev"}}' \
     --region us-east-1
   
   # Check CloudTrail for API calls
   aws cloudtrail lookup-events --lookup-attributes AttributeKey=EventName,AttributeValue=DeleteVolume --region us-east-1
   
   # Check CloudWatch Dashboard for anomalies
   # Navigate to: CloudWatch → Dashboards → FinOps-Bot-Dashboard-dev
   ```

3. **Rotate Secrets:**
   ```bash
   aws secretsmanager update-secret --secret-id finops/slack-webhook-dev \
     --secret-string '{"SLACK_WEBHOOK_URL":"https://hooks.slack.com/services/NEW_WEBHOOK"}' --region us-east-1
   ```

4. **Revoke Access (if needed):**
   ```bash
   # Detach the compromised policy
   aws iam detach-role-policy --role-name finops-lambda-role-dev --policy-arn <policy-arn> --region us-east-1
   ```

5. **Post-Incident Forensics:**
   - Export DynamoDB audit trail for analysis.
   - Export CloudTrail logs for review.
   - Review Lambda code for vulnerabilities.

**Success Criteria:**
- Bot is stopped within 5 minutes.
- Secrets are rotated within 10 minutes.
- Audit trail is exported for analysis.
- Post-mortem is completed within 24 hours.

---

### 5.2 Drill Scenario: Broad IAM Permissions

**Scenario:** IAM permissions are accidentally too broad (e.g., `ec2:*` instead of specific actions).

**Detection:**
1. Security scanner (checkov/tfsec) fails in CI with "overly permissive policy" warning.
2. CloudTrail shows API calls using permissions beyond the intended scope.
3. Audit trail shows actions on resources without the required tags.

**Response Steps:**
1. **Immediate Stop:**
   ```bash
   aws events disable-rule --name finops-daily-trigger-dev --region us-east-1
   aws lambda update-function-configuration --function-name finops-cleaner-dev --environment "Variables={DRY_RUN=true}" --region us-east-1
   ```

2. **Review IAM Policy:**
   ```bash
   aws iam get-policy --policy-arn <policy-arn> --region us-east-1
   aws iam get-policy-version --policy-arn <policy-arn> --version-id v1 --region us-east-1
   ```

3. **Rollback Terraform:**
   ```bash
   cd terraform
   terraform state pull > previous_state.tfstate
   terraform state push previous_state.tfstate
   ```

4. **Apply Least-Privilege Policy:**
   ```bash
   terraform apply -var-file=terraform.tfvars
   ```

5. **Validate the Fix:**
   ```bash
   checkov -d terraform/ --check CKV_AWS_109,CKV_AWS_111,CKV_AWS_112
   tfsec terraform/ --include aws-iam-no-policy-wildcards
   ```

**Success Criteria:**
- IAM policy is updated to least-privilege within 15 minutes.
- checkov/tfsec pass in CI.
- Bot is re-enabled and functions correctly.

---

### 5.3 Drill Scenario: Secrets Exposure

**Scenario:** The Slack webhook URL is accidentally exposed in logs or source code.

**Detection:**
1. Secrets Manager access audit shows unusual `GetSecretValue` calls.
2. CloudTrail shows `GetSecretValue` from unexpected IP addresses.
3. Security scan detects hardcoded secrets in the repository.

**Response Steps:**
1. **Rotate the Secret Immediately:**
   ```bash
   aws secretsmanager update-secret --secret-id finops/slack-webhook-dev \
     --secret-string '{"SLACK_WEBHOOK_URL":"https://hooks.slack.com/services/NEW_WEBHOOK"}' --region us-east-1
   ```

2. **Investigate Access:**
   ```bash
   # Query CloudTrail for GetSecretValue calls
   aws cloudtrail lookup-events --lookup-attributes AttributeKey=EventName,AttributeValue=GetSecretValue --region us-east-1
   
   # Check for unauthorized access
   aws cloudtrail lookup-events --lookup-attributes AttributeKey=Username,AttributeValue=* --region us-east-1
   ```

3. **Revoke Access (if needed):**
   - Remove unauthorized IAM users/roles.
   - Attach a more restrictive policy.

4. **Update Security Scanners:**
   - Add secret detection to CI (gitleaks, trufflehog).
   - Add secret scanning to pre-commit hooks.

**Success Criteria:**
- Secret is rotated within 5 minutes.
- Unauthorized access is identified and blocked.
- Security scanners are updated.

---

### 5.4 Drill Scenario: Regional AWS Failure (NEW)

**Scenario:** The primary AWS region (us-east-1) is unavailable.

**Detection:**
1. Health Check Lambda fails for multiple consecutive checks.
2. CloudWatch Dashboard shows service degradation.
3. Manual verification confirms region is unavailable.

**Response Steps:**
1. **Verify the Outage:**
   ```bash
   # Check AWS Service Health Dashboard
   curl https://status.aws.amazon.com/
   
   # Test connectivity to DynamoDB in the region
   aws dynamodb list-tables --region us-east-1 --max-items 1
   ```

2. **Activate Secondary Region (us-west-2):**
   ```bash
   # Update Terraform to use secondary region
   export TF_VAR_aws_region=us-west-2
   
   # Apply configuration to secondary region
   cd terraform
   terraform workspace select secondary
   terraform apply -var-file=terraform.tfvars.secondary -auto-approve
   ```

3. **Restore Data from Backup:**
   ```bash
   # Restore DynamoDB from PITR
   aws dynamodb restore-table-from-backup \
     --target-table-name FinOps-State-dev-secondary \
     --backup-arn <backup-arn> \
     --region us-west-2
   
   # Sync S3 bucket
   aws s3 sync s3://finops-audit-dev-123456789012-us-east-1 \
     s3://finops-audit-dev-123456789012-us-west-2 \
     --region us-west-2
   ```

4. **Update Route53 (if using custom domain):**
   ```bash
   aws route53 change-resource-record-sets \
     --hosted-zone-id <zone-id> \
     --change-batch '{
       "Changes": [{
         "Action": "UPSERT",
         "ResourceRecordSet": {
           "Name": "dashboard.finops.example.com",
           "Type": "CNAME",
           "TTL": 60,
           "ResourceRecords": [{"Value": "finops-audit-dev-123456789012.s3-website-us-west-2.amazonaws.com"}]
         }
       }]
     }'
   ```

5. **Verify Recovery:**
   ```bash
   # Check health check in secondary region
   aws lambda invoke --function-name finops-health-check-dev-secondary --region us-west-2
   ```

**Success Criteria:**
- Bot is operational in secondary region within 30 minutes.
- Data is restored with RPO of 24 hours.
- Dashboard is accessible via secondary region.

---

### 5.5 Drill Scenario: DynamoDB Corruption (NEW)

**Scenario:** The DynamoDB state table is corrupted or accidentally deleted.

**Detection:**
1. Lambda logs show `TableNotFoundException` or `ValidationException`.
2. Health Check Lambda fails on DynamoDB connectivity.
3. Manual query returns unexpected results.

**Response Steps:**
1. **Immediate Stop:**
   ```bash
   aws events disable-rule --name finops-daily-trigger-dev --region us-east-1
   ```

2. **Check Point-in-Time Recovery (PITR):**
   ```bash
   aws dynamodb describe-continuous-backups \
     --table-name FinOps-State-dev \
     --region us-east-1
   ```

3. **Restore from PITR:**
   ```bash
   # Get latest restore timestamp
   aws dynamodb list-backups --table-name FinOps-State-dev --region us-east-1
   
   # Restore to the latest available time
   aws dynamodb restore-table-to-point-in-time \
     --source-table-name FinOps-State-dev \
     --target-table-name FinOps-State-dev-restored \
     --restore-date-time <latest-restore-time> \
     --region us-east-1
   ```

4. **Replace Corrupted Table:**
   ```bash
   # Delete corrupted table
   aws dynamodb delete-table --table-name FinOps-State-dev --region us-east-1
   
   # Rename restored table
   aws dynamodb update-table --table-name FinOps-State-dev-restored \
     --table-name FinOps-State-dev --region us-east-1
   ```

5. **Re-enable Bot:**
   ```bash
   aws events enable-rule --name finops-daily-trigger-dev --region us-east-1
   ```

**Success Criteria:**
- DynamoDB table is restored within 15 minutes.
- Data loss is limited to the PITR window (5 minutes).
- Bot is re-enabled and functioning correctly.

---

## 6. Common Failure Scenarios

### 6.1 Lambda Timeout

**Symptoms:**
- CloudWatch Alarm `finops-lambda-duration-alarm` triggers.
- Lambda logs show `Task timed out after X seconds`.
- Bot fails to complete the daily run.

**Root Causes:**
- Too many resources to scan.
- Network latency (if running in VPC).
- Memory insufficient (causing GC overhead).

**Immediate Actions:**
1. Check the duration in CloudWatch:
   ```bash
   aws cloudwatch get-metric-statistics --namespace AWS/Lambda --metric-name Duration --dimensions Name=FunctionName,Value=finops-cleaner-dev --statistics Average --start-time $(date -d '1 hour ago' -u +'%Y-%m-%dT%H:%M:%SZ') --end-time $(date -u +'%Y-%m-%dT%H:%M:%SZ') --period 3600 --region us-east-1
   ```

2. Increase timeout to 600 seconds:
   ```bash
   aws lambda update-function-configuration --function-name finops-cleaner-dev --timeout 600 --region us-east-1
   ```

3. If still timing out, increase memory:
   ```bash
   aws lambda update-function-configuration --function-name finops-cleaner-dev --memory-size 512 --region us-east-1
   ```

**Escalation:** If timeout persists, investigate the number of resources in each region and consider adding pagination limits.

---

### 6.2 SQS Dead-Letter Queue Overflow

**Symptoms:**
- CloudWatch Alarm `finops-dlq-alarm` triggers.
- SQS DLQ depth > 1.
- Lambda function has errors.

**Root Causes:**
- Lambda failures (timeout, permission errors, API throttling).
- Transient AWS issues.

**Immediate Actions:**
1. Check DLQ depth:
   ```bash
   aws sqs get-queue-attributes --queue-url $(aws sqs get-queue-url --queue-name finops-dlq-dev --region us-east-1 --query QueueUrl --output text) --attribute-names ApproximateNumberOfMessages --region us-east-1
   ```

2. Pull a message from DLQ to inspect the error:
   ```bash
   aws sqs receive-message --queue-url $(aws sqs get-queue-url --queue-name finops-dlq-dev --region us-east-1 --query QueueUrl --output text) --max-number-of-messages 1 --region us-east-1
   ```

3. Check Lambda logs for errors:
   ```bash
   aws logs get-log-events --log-group-name /aws/lambda/finops-cleaner-dev --limit 50 --order-by LastEventTime --descending --region us-east-1
   ```

4. If the error is transient, flush the DLQ:
   ```bash
   aws sqs purge-queue --queue-url $(aws sqs get-queue-url --queue-name finops-dlq-dev --region us-east-1 --query QueueUrl --output text) --region us-east-1
   ```

**Escalation:** If the DLQ continues to fill, escalate to the SRE team to investigate the root cause of Lambda failures.

---

### 6.3 Slack Notifications Not Sending

**Symptoms:**
- No Slack notifications for quarantine or deletion events.
- Lambda logs show `Failed to send Slack notification`.

**Root Causes:**
- Slack webhook URL is invalid or expired.
- Secrets Manager secret is not accessible.
- Network issues (if running in VPC).

**Immediate Actions:**
1. Verify the Slack webhook URL in Secrets Manager:
   ```bash
   aws secretsmanager get-secret-value --secret-id finops/slack-webhook-dev --query SecretString --output text --region us-east-1 | jq
   ```

2. Test the webhook URL manually:
   ```bash
   curl -X POST -H 'Content-type: application/json' \
     --data '{"text":"FinOps Bot test notification from runbook"}' \
     https://hooks.slack.com/services/...
   ```

3. If the URL is invalid, rotate the secret:
   ```bash
   aws secretsmanager update-secret --secret-id finops/slack-webhook-dev --secret-string '{"SLACK_WEBHOOK_URL":"https://hooks.slack.com/services/NEW_WEBHOOK"}' --region us-east-1
   ```

**Escalation:** If the secret is correct but notifications still fail, check Lambda IAM permissions for Secrets Manager and KMS.

---

### 6.4 Lambda Permission Errors

**Symptoms:**
- Lambda logs show `AccessDeniedException` or `UnauthorizedOperation`.
- Bot fails to discover or delete resources.

**Root Causes:**
- IAM policy has missing permissions.
- IAM policy condition is too restrictive.
- KMS key permissions are misconfigured.

**Immediate Actions:**
1. Check the error message in CloudWatch logs.

2. Verify IAM role permissions:
   ```bash
   aws iam list-attached-role-policies --role-name finops-lambda-role-dev --region us-east-1
   ```

3. Verify the IAM policy document:
   ```bash
   aws iam get-policy --policy-arn arn:aws:iam::<account-id>:policy/finops-lambda-policy-dev --region us-east-1
   ```

4. Temporarily add broad permissions for debugging (remember to revert):
   ```bash
   aws iam attach-role-policy --role-name finops-lambda-role-dev --policy-arn arn:aws:iam::aws:policy/AmazonEC2ReadOnlyAccess --region us-east-1
   ```

**Escalation:** If permissions are correct but still failing, investigate whether the IAM condition is blocking the action. Check the resource tags to ensure `FinOps: AutoPurge` is present.

---

### 6.5 Bot Is Deleting Unexpected Resources

**Symptoms:**
- Resources without `FinOps: AutoPurge` tag are being deleted.
- Critical production resources are affected.

**Root Causes:**
- IAM policy condition is misconfigured (allowing deletion without tags).
- Bot logic has a bug (e.g., `hasAutoPurge` check is inverted).
- `DRY_RUN` is set to `false`.

**Immediate Actions:**

1. **EMERGENCY STOP: Disable the bot:**
   ```bash
   aws events disable-rule --name finops-daily-trigger-dev --region us-east-1
   ```

2. **Set DRY_RUN to true immediately:**
   ```bash
   aws lambda update-function-configuration --function-name finops-cleaner-dev --environment "Variables={DRY_RUN=true}" --region us-east-1
   ```

3. **Add critical resources to EXCLUDED_IDS:**
   ```bash
   aws lambda update-function-configuration --function-name finops-cleaner-dev --environment "Variables={EXCLUDED_IDS=vol-123,vol-456}" --region us-east-1
   ```

4. **Investigate the cause:**
   - Check CloudWatch logs for the deletion events.
   - Review the IAM policy condition (Section 6.4).
   - Review the bot's logic for the `evaluateVolume` function.
   - Check DynamoDB state for the affected resources.

5. **Restore resources if possible:**
   - EBS volumes: Restore from snapshot.
   - EIPs: Re-allocate (but IP will change).
   - Snapshots: Cannot be restored (but can be recreated from volume).
   - RDS: Start the instance again.

**Escalation:** Immediately escalate to the SRE team and security team. Post-incident review is mandatory.

---

## 7. Structured Log Analysis (Forensics)

### 7.1 CloudWatch Logs Insights Queries

```sql
-- Find all actions performed by a specific IAM principal
fields @timestamp, level, message, who, what, resource_id, resource_type, correlation_id
| filter who like /finops-lambda-role/
| sort @timestamp desc
| limit 100

-- Find all DELETE actions in the last 24 hours
fields @timestamp, level, message, what, resource_id, resource_type, estimated_savings
| filter what = "DELETE_VOLUME"
| filter @timestamp > now() - 24h
| sort @timestamp desc

-- Correlation ID tracking across the entire invocation
fields @timestamp, correlation_id, what, resource_id, message
| filter correlation_id = "abc-123-def-456"
| sort @timestamp asc

-- Identify permission errors (potential privilege escalation)
fields @timestamp, level, who, what, message
| filter level = "error"
| filter message like /AccessDenied|Unauthorized/
| sort @timestamp desc

-- Find all actions with a specific ActionReason
fields @timestamp, level, who, what, resource_id, action_reason
| filter action_reason like /AutoPurge/
| sort @timestamp desc

-- SLO tracking: Bot run success rate
fields @timestamp, slo_metric, level
| filter slo_metric = "bot_run_success"
| stats count(*) as total, count_if(level != "error") as success by bin(1d)
| eval success_rate = success / total * 100
```

### 7.2 Audit Trail Queries (DynamoDB)

```sql
-- Find all actions performed by a specific IAM principal
SELECT * FROM "FinOps-State"
WHERE ActionedBy = 'arn:aws:sts::123456789012:assumed-role/finops-lambda-role-dev/finops-cleaner'
AND DeletionTimestamp > unix_timestamp(now() - interval 7 day)

-- Find all DELETE actions in the last 24 hours
SELECT * FROM "FinOps-State"
WHERE ActionTaken = 'DELETED'
AND DeletionTimestamp > unix_timestamp(now() - interval 1 day)

-- Find all DELETION_FAILED resources
SELECT * FROM "FinOps-State"
WHERE ActionTaken = 'DELETION_FAILED'
AND RetryCount > 3

-- Find all resources with DeleteProtection = true
SELECT * FROM "FinOps-State"
WHERE DeleteProtection = true

-- Correlation ID tracing
SELECT * FROM "FinOps-State"
WHERE CorrelationID = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'
```

---

## 8. Post-Incident Forensics

### 8.1 Preserving Evidence

| Evidence Source | Action | Command |
| :--- | :--- | :--- |
| **CloudWatch Logs** | Export structured logs | `aws logs create-export-task --task-name "incident-export" --log-group-name /aws/lambda/finops-cleaner-dev --from <start-epoch> --to <end-epoch> --destination finops-audit-<account-id> --destination-prefix logs/incident/` |
| **DynamoDB State** | Export audit trail | `aws dynamodb export-table-to-point-in-time --table-arn <table-arn> --s3-bucket finops-audit-<account-id> --s3-prefix dynamodb/incident/ --export-type FULL_EXPORT` |
| **CloudTrail** | Export API logs | `aws cloudtrail lookup-events --lookup-attributes AttributeKey=EventName,AttributeValue=DeleteVolume --region us-east-1` |
| **CloudWatch Dashboard** | Export dashboard snapshot | Screenshot of CloudWatch Dashboard for incident timeline |

### 8.2 Forensic Analysis Checklist

- [ ] Export all CloudWatch Logs for the incident timeframe.
- [ ] Export DynamoDB audit trail for the incident timeframe.
- [ ] Export CloudTrail logs for API calls.
- [ ] Identify all affected resources.
- [ ] Identify the IAM principal used for unauthorized actions.
- [ ] Determine if secrets were exposed.
- [ ] Determine if IAM permissions were abused.
- [ ] Preserve all evidence for security review.
- [ ] Review CloudWatch Dashboard metrics for anomaly detection.
- [ ] Check Health Check Lambda logs for connectivity issues.

---

## 9. Disaster Recovery Strategy (NEW)

### 9.1 Disaster Recovery Overview

| Scenario | Recovery Action | RTO | RPO |
| :--- | :--- | :--- | :--- |
| **Regional AWS Failure** | Deploy to secondary region; restore from backup | 30 minutes | 24 hours |
| **DynamoDB Corruption** | Restore from point-in-time recovery (PITR) | 15 minutes | 5 minutes |
| **S3 Bucket Deletion** | Restore from versioning; recreate bucket | 10 minutes | 0 minutes |
| **Lambda Function Failure** | Re-deploy from artifact; rollback to previous version | 5 minutes | 0 minutes |
| **Secrets Manager Failure** | Restore from backup; use fallback secrets | 15 minutes | 1 hour |
| **Slack Webhook Failure** | Rotate to alternate webhook; fallback to email | 5 minutes | 0 minutes |

### 9.2 Recovery Procedures

#### Regional Failure

1. Verify the outage and confirm primary region is unavailable.
2. Activate secondary region with Terraform.
3. Restore data from backups (DynamoDB PITR, S3 replication).
4. Update DNS records to point to secondary region.
5. Verify recovery with health checks.

#### DynamoDB Corruption

1. Stop the bot immediately.
2. Identify the last known good state.
3. Restore from PITR to the latest available time.
4. Replace the corrupted table with the restored version.
5. Re-enable the bot.

#### S3 Bucket Deletion

1. Stop the bot immediately.
2. Recreate the S3 bucket with the same name.
3. Restore objects from versioning (if enabled).
4. Re-upload the dashboard HTML (if needed).
5. Re-enable the bot.

### 9.3 DR Testing Schedule

| Test | Frequency | Success Criteria |
| :--- | :--- | :--- |
| **Regional Failover** | Quarterly | Bot operational in secondary region within 30 minutes |
| **DynamoDB Restore** | Quarterly | Table restored within 15 minutes |
| **S3 Restore** | Quarterly | Objects restored within 10 minutes |
| **Full DR Drill** | Annually | All systems recovered within 1 hour |

---

## 10. Break-Glass Procedures

| Scenario | Immediate Action | Recovery Action | Time |
| :--- | :--- | :--- | :--- |
| **Compromised Bot** | Disable EventBridge rule + Set `DRY_RUN=true` + Add resources to `EXCLUDED_IDS` | Rotate secrets, review logs, restore resources | 5 minutes |
| **Broad IAM Permissions** | Revoke suspicious permissions + Disable bot | Review IAM policies, rollback Terraform | 10 minutes |
| **Secrets Exposed** | Rotate secret immediately | Review CloudTrail for unauthorized access | 3 minutes |
| **Bot Deleting Unexpected Resources** | Disable bot immediately + Add resources to `EXCLUDED_IDS` | Restore from snapshot, review logs | 5 minutes |
| **Regional Failure** | Activate secondary region | Restore data, update DNS | 30 minutes |
| **DynamoDB Corruption** | Stop bot + Restore from PITR | Replace table, re-enable bot | 15 minutes |

### 10.1 Break-Glass Commands

```bash
# 1. EMERGENCY STOP: Disable the bot
aws events disable-rule --name finops-daily-trigger-dev --region us-east-1

# 2. Set DRY_RUN to true
aws lambda update-function-configuration --function-name finops-cleaner-dev \
  --environment "Variables={DRY_RUN=true}" --region us-east-1

# 3. Add critical resources to EXCLUDED_IDS
aws lambda update-function-configuration --function-name finops-cleaner-dev \
  --environment "Variables={EXCLUDED_IDS=vol-123,vol-456,vol-789}" --region us-east-1

# 4. Rotate secret
aws secretsmanager update-secret --secret-id finops/slack-webhook-dev \
  --secret-string '{"SLACK_WEBHOOK_URL":"https://hooks.slack.com/services/..."}' --region us-east-1

# 5. Investigate logs
aws logs get-log-events --log-group-name /aws/lambda/finops-cleaner-dev \
  --limit 100 --order-by LastEventTime --descending --region us-east-1

# 6. Check Health Check status
aws lambda invoke --function-name finops-health-check-dev --region us-east-1
```

---

## 11. Post-Mortem Checklist

After resolving any incident, complete this post-mortem:

- [ ] **Incident Summary:** What happened? When did it start? When was it resolved?
- [ ] **Impact:** Which resources were affected? Was there any data loss?
- [ ] **Root Cause:** Why did the incident occur? Was it a bug, configuration error, or external issue?
- [ ] **Detection:** How was the incident detected? Was the alerting effective? (Check CloudWatch Dashboard)
- [ ] **Response:** What actions were taken to resolve it? How long did each step take?
- [ ] **DR Activation:** Was DR activated? If so, how long did recovery take?
- [ ] **Prevention:** What changes will prevent this from happening again? (Code fix, config change, monitoring addition)
- [ ] **Lessons Learned:** What went well? What could be improved?
- [ ] **Action Items:** Assign tasks to implement prevention measures.

### 11.1 Post-Mortem Template

```markdown
# Post-Mortem: [Incident Title]

**Date:** [YYYY-MM-DD]
**Incident ID:** [IM-XXX]
**Severity:** [P1/P2/P3/P4]
**Duration:** [Start Time] - [End Time]
**Responder(s):** [Name(s)]
**DR Activated:** [Yes/No]

## Summary
[2-3 sentences describing what happened]

## Timeline
- [HH:MM] - [Event description]
- [HH:MM] - [Event description]
- [HH:MM] - [Event description]

## Root Cause
[Detailed explanation of why the incident occurred]

## Impact
- **Resources Affected:** [List]
- **Data Loss:** [Yes/No - Details]
- **Customer Impact:** [Description]
- **SLO Impact:** [Which SLOs were affected]

## Action Items
| Action Item | Owner | Due Date | Status |
| :--- | :--- | :--- | :--- |
| [Action 1] | [Owner] | [Date] | [Open/Closed] |
| [Action 2] | [Owner] | [Date] | [Open/Closed] |

## Lessons Learned
- **What went well:** [List]
- **What went poorly:** [List]
- **What we will do differently:** [List]
```

---

## 12. Runbook Maintenance

- **Review Frequency:** Quarterly, or after every incident.
- **Update Responsibility:** DevOps Team.
- **Change Log:** Track all changes to this document.

| Date | Version | Author | Changes |
| :--- | :--- | :--- | :--- |
| 2026-06-22 | 3.1 | Jibrin Ahmed | Added Disaster Recovery Strategy, updated drill scenarios |
| 2026-06-22 | 3.0 | Jibrin Ahmed | Added incident drills, security response, forensic analysis |
| 2026-06-19 | 2.0 | Jibrin Ahmed | Added structured logging and audit trail queries |
| 2026-06-19 | 1.0 | Jibrin Ahmed | Initial creation |

---

## 13. Sign-Off

| Role | Name | Date | Signature |
| :--- | :--- | :--- | :--- |
| **Project Lead / Architect** | Jibrin Ahmed | June 22, 2026 |  |
| **On-Call Lead** | Jibrin Ahmed  | June 22, 2026 | JA |
| **SRE Lead** | Jibrin Ahmed  | June 22, 2026 | JA |
| **Security Lead** | Jibrin Ahmed  | June 22, 2026 | JA |

---
