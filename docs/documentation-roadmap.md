# 🗂️ MASTER DOCUMENT ROADMAP: Project C (FinOps Bot)
*Version: 2.0 - Floci-Integrated*  
*Sequence: Read & Write in this exact order. Each document feeds the next.*

---

### 📄 Document 01: Project Charter (The "North Star")
*Purpose:* To lock the scope so you don't suffer from feature creep.

- **Problem Statement:** Define the specific cloud waste problem (unattached EBS, idle EIPs, stale snapshots).
- **Project Goal:** Build an autonomous, serverless agent that safely identifies and removes orphaned cloud resources.
- ### Scope: Tiered Approach

- ### Tier 1: Core v1 (Must Have)
- EC2 EBS Volumes: Delete unattached volumes older than 7 days.
- Elastic IPs: Release unassociated EIPs.
- EBS Snapshots: Delete snapshots older than 30 days (preserve latest 3 per volume).
- Quarantine Period: Tag resources with `Pending_Deletion` for 7 days before deletion.
- State Tracking: DynamoDB table to prevent duplicate deletions (idempotency).
- Slack Notifications: Warning on quarantine, confirmation on deletion.

- ### Tier 2: Extended v1 (Should Have - Zero Cost with Floci)
- **RDS Idle Detection:** Identify RDS instances with no connections for >7 days and automatically stop them (not delete). Send a Slack warning before stopping.
- **Static HTML Dashboard:** Generate a lightweight HTML report (with Chart.js) showing monthly savings trends and upload it to an S3 bucket as a static website.
- **Multi-Region Scanning:** Scan across 3 regions (us-east-1, us-west-2, eu-west-1) in a single Lambda invocation.

- ### Tier 3: Out-of-Scope (v1 Non-Goals)
- Multi-account AWS Organizations scanning.
- Full React/Angular single-page application with user authentication.
- Cost Explorer API integration (requires real billing data).
- Auto-deletion of untagged or production-tagged resources (Safety First).

---

### 📄 Document 02: High-Level Design (HLD) - "System Narrative"
*Purpose:* To explain the architecture to a non-technical stakeholder or hiring manager.

- **System Context Diagram:** The Mermaid.js chart from your README (EventBridge → Lambda → DynamoDB → S3 → Slack).
- **Narrative Data Flow:** A 5-step paragraph description of how a resource moves from discovery to deletion.
- **Failover Strategy:** Explicitly state that failed Lambda invocations (after 3 retries) route to an SQS Dead-Letter Queue (DLQ) for manual inspection.
- **NEW - Development & Testing Architecture:** A dedicated section explaining how **Floci** (the AWS emulator) enables local development and CI testing. Diagram showing: `Local Dev → Floci (http://localhost:4566) → Tests` vs. `Production → Real AWS`.

---

### 📄 Document 03: DynamoDB State Schema (The "Source of Truth")
*Purpose:* To define the exact database structure *before* writing a single query.

- **Table Name:** `FinOps-State`
- **Primary Key:** `ResourceId` (String - Partition Key).
- **Sort Key:** `Region` (String) – *To allow cross-region scanning if you ever scale.*
- **Required Attributes:**
  - `ActionTaken` (String: 'QUARANTINED', 'DELETED', 'SKIPPED')
  - `DeletionTimestamp` (Number - Unix Epoch)
  - `SizeGB` (Number)
  - `EstimatedSavings` (Float)
- **Global Secondary Index (GSI):** `ActionTaken-index` to quickly query all 'DELETED' resources for monthly reports.
- **TTL (Time-to-Live):** Enable TTL on `DeletionTimestamp` to auto-delete records after 90 days (cost optimization for the database itself).

---

### 📄 Document 04: Configuration Matrix (Env Vars vs. Secrets)
*Purpose:* To map exactly where every configuration value lives. *This was a critical gap in the original plan.*

- **The Rule:** Non-sensitive values go in Lambda Environment Variables. Sensitive values go in AWS Secrets Manager.
- **The Matrix:**

| Configuration Key | Source | Rationale |
| :--- | :--- | :--- |
| `DRY_RUN` | Lambda Env Var | Toggled frequently; non-sensitive. |
| `COST_PER_GB` | Lambda Env Var | Changes per region; non-sensitive. |
| `EXCLUDED_IDS` | Lambda Env Var | Comma-separated list for emergency kill-switch. |
| `SLACK_WEBHOOK_URL` | **AWS Secrets Manager** | Sensitive; encrypted and rotatable. |
| `AWS_ACCESS_KEY` | IAM Role (Instance Metadata) | **Never stored**; Lambda assumes role via IMDS. |
| `FLOCI_ENDPOINT` | Lambda Env Var | Local override: `http://localhost:4566` for development only. |

---

### 📄 Document 05: Pricing & Financial Calculation Engine
*Purpose:* To define the math that generates your cost-savings reports.

- **EBS Pricing Map:** A Python dictionary mapping volume types (`gp3`, `gp2`, `io1`) to their regional $/GB-month cost.
- **Snapshot Formula:** `SnapshotCost = SnapshotDataSize (GB) * $0.05` (adjust for region).
- **EIP Formula:** `EIPCost = (Hours_Unattached * $0.005)`.
- **Explicit Note:** Document that v1 uses hardcoded pricing maps to avoid throttling the AWS Price List API, but v2 will integrate `boto3` to fetch live prices.

---

### 📄 Document 06: Low-Level Design (LLD) - Functional Spec
*Purpose:* The exact pseudo-code blueprint. The Python code will be a direct translation of this document.

- **Function 1: `get_zombie_candidates()`** – Filters EBS volumes where `Status == 'available'` AND `CreateTime < (today - 7 days)`.
- **Function 2: `apply_quarantine_tag()`** – Calls `ec2.create_tags()` with key `Pending_Deletion` and TTL value.
- **Function 3: `check_dynamodb_state()`** – Queries the `FinOps-State` table. Returns `'EXISTS'`, `'DELETED'`, or `'NOT_FOUND'`.
- **Function 4: `execute_deletion()`** – Only runs if `dry_run=False`, `tag_check=True`, `quarantine_expired=True`, and `dynamodb_status='NOT_FOUND'`. Writes `'DELETED'` status to DynamoDB upon success.
- **NEW - Environment Abstraction:** All boto3 clients will check for a `FLOCI_ENDPOINT` environment variable. If present, they will point to `http://localhost:4566` instead of real AWS.

---

### 📄 Document 07: Infrastructure as Code (IaC) Manifest
*Purpose:* The inventory of AWS resources Terraform will provision.

- **Provider:** AWS (region passed as variable).
- **Variables:** `aws_region`, `dry_run` (default `true`), `slack_webhook` (sensitive).
- **Resource 1:** `aws_iam_role` (Lambda execution role).
- **Resource 2:** `aws_iam_policy` (Attached to role – list the exact 6 actions).
- **Resource 3:** `aws_lambda_function` (Runtime: Python 3.9, Handler: `app.lambda_handler`, Timeout: 300s, Memory: 256MB).
- **Resource 4:** `aws_dynamodb_table` (Name: `FinOps-State`).
- **Resource 5:** `aws_cloudwatch_event_rule` (Cron: `0 2 * * ? *`).
- **Resource 6:** `aws_sqs_queue` (Dead-Letter Queue - retention: 14 days).
- **NEW - Local Development Override:** Document how to use Terraform workspaces to conditionally skip AWS resource creation when running against Floci.

---

### 📄 Document 08: Security & IAM Matrix (The "Least Privilege" Bible)
*Purpose:* To prove you understand AWS security. This is what interviewers will grill you on.

- **Authorization Table:**

| Action | Required IAM Permission | Resource Restriction |
| :--- | :--- | :--- |
| List Volumes | `ec2:DescribeVolumes` | `*` (Read-only, safe) |
| Apply Quarantine Tag | `ec2:CreateTags` | `arn:aws:ec2:*:*:volume/*` |
| Delete Volume | `ec2:DeleteVolume` | **Condition**: `aws:ResourceTag/FinOps == "AutoPurge"` |
| Read State | `dynamodb:GetItem` | `arn:aws:dynamodb:*:*:table/FinOps-State` |

- **Secrets Fetching:** Explicitly document that the Lambda fetches the Slack Webhook from Secrets Manager using `boto3.client('secretsmanager').get_secret_value()`.
- **Floci Note:** Floci fully supports IAM authentication and SigV4 validation, meaning your security policies can be tested locally without touching real AWS.

---

### 📄 Document 09: Test Strategy with Floci (Major Overhaul)
*Purpose:* To define a comprehensive, multi-layered testing strategy using Floci for integration tests.

- **Test Pyramid (Floci-First):**
  - **Layer 1: Unit Tests (pytest, no AWS)**
    - 20+ tests covering pure Python logic.
    - Run: `pytest tests/unit/`
    - No external dependencies; runs in milliseconds.
  - **Layer 2: Integration Tests (Floci)**
    - 30+ tests against Floci's emulated AWS environment.
    - Run: `pytest tests/integration/`
    - Floci spins up in 24ms and emulates EC2, DynamoDB, S3, and Lambda.
    - **Zero AWS credentials required. Zero cost.**
    - Tests cover: Volume discovery, quarantine tagging, deletion logic, DynamoDB state tracking, and Slack formatting.
  - **Layer 3: End-to-End Tests (Real AWS - Optional)**
    - 5+ critical path tests.
    - Run only on `main` branch via FlowCI.
    - Validates that Floci's behavior matches real AWS.
    - Cost: < $1/month if run sparingly.
- **Floci Setup Commands (for README):**
  ```bash
  # Install Floci
  brew install floci-io/floci/floci
  
  # Start the emulator
  floci start
  
  # Set environment variables
  eval $(floci env)
  ```
- **Test Data:** Define mock EC2 volumes, snapshots, and EIPs that will be pre-loaded into Floci before each integration test run.

---

### 📄 Document 10: On-Call Runbook (Incident Response)
*Purpose:* The exact manual steps to take when the bot breaks at 2:15 AM.

- **Alert 1: Lambda Timeout.** *Action:* SSH into jumpbox, run `aws lambda update-function-configuration --function-name finops-cleaner --memory-size 512`.
- **Alert 2: High SQS DLQ Depth.** *Action:* Pull a message from SQS via `aws sqs receive-message --queue-url <url>`. Inspect the JSON error payload. Manually delete the offending Volume ID if it's a false positive.
- **Manual Fallback Trigger:** The exact Bash command to bypass the scheduler and trigger the Lambda manually for emergency testing: `aws lambda invoke --function-name finops-cleaner --payload '{"manual_override": true}' manual_output.json`.

---

### 📄 Document 11: CI/CD Pipeline Design (FlowCI + Floci) - **NEW**
*Purpose:* To explicitly define how FlowCI orchestrates the entire build, test, and deploy process with Floci at its core.

- **Pipeline Stages:**
  1. **Checkout:** Pull the latest code from GitHub.
  2. **Install Dependencies:** `pip install -r requirements.txt`
  3. **Install Floci:** `brew install floci-io/floci/floci`
  4. **Start Floci:** `floci start` (24ms startup)
  5. **Run Unit Tests:** `pytest tests/unit/`
  6. **Run Integration Tests (against Floci):** `pytest tests/integration/`
  7. **Stop Floci:** `floci stop`
  8. **Build Docker Image:** `docker build -t finops-bot .`
  9. **Push to Container Registry:** (ECR/Docker Hub)
  10. **Deploy to AWS (only on `main` branch):** `terraform apply -auto-approve`
  11. **Run E2E Tests (against real AWS):** `pytest tests/e2e/`
- **Cost Breakdown:** 
  - Stages 1-7: $0 (everything runs inside FlowCI containers).
  - Stages 8-10: Minimal cost (ECR storage + Terraform state).
  - Stage 11: < $0.01 per run (spinning up a tiny test volume).
- **Pipeline Diagram:** A visual flowchart showing the exact sequence of stages and their dependencies.

---

# ✅ Your Immediate 10-Minute Action Plan

You now have a complete, production-grade document roadmap that integrates:

- **Safety:** Quarantine periods, Positive Tagging, DynamoDB idempotency.
- **Security:** Least-privilege IAM, Secrets Manager, SigV4 validation.
- **Modern Tooling:** Floci for zero-cost local development and CI testing.
- **Operational Readiness:** Runbooks, SQS DLQs, CloudWatch alarms.

**Your immediate next step:** 
Open a new Markdown file called `01-project-charter.md`. Copy the template below and fill in the bracketed `[Your text]` sections:

```markdown
# Project Charter: Cloud FinOps Bot

## 1. Executive Summary
[2-3 sentences on the problem and solution]

## 2. Scope
**In-Scope (v1):**
- EC2 EBS Volumes (unattached)
- Elastic IPs (unassociated)
- EBS Snapshots (older than 30 days)

**Out-of-Scope (v1):**
- RDS Instances
- S3 Buckets
- Multi-account scanning

## 3. Success Criteria
- [Criterion 1]
- [Criterion 2]
- [Criterion 3]

## 4. Development Philosophy
- Local-first development with Floci
- Zero cloud costs during testing
- CI pipeline validates everything before touching real AWS
```

Once you fill out that 1-page Charter and reply with *"Charter is drafted,"* we will move to **Document 02 (HLD)** and draw the full architecture diagram together—with Floci integrated right into the flow. 

Let's start building this properly. Reply when you're ready!