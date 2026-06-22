# Pricing Engine: Cloud FinOps Bot

**Document:** 05  
**Version:** 2.0 (Go Edition - Audited & Patched)  
**Author:** Jibrin Ahmed  
**Date:** June 19, 2026  
**Status:** Final

---

## 1. Document Purpose

This document defines the **cost calculation engine** used by the FinOps Bot to estimate monthly savings from resource deletions and stoppages. It covers:

- **EBS Volumes:** Cost per GB-month for different volume types.
- **Elastic IPs:** Cost per hour for unattached EIPs.
- **EBS Snapshots:** Cost per GB-month for snapshot storage.
- **RDS Instances:** Cost per hour for stopped vs. running instances (savings = cost of running).
- **Regional Pricing Variability:** How prices differ across AWS regions.

Accurate pricing is essential for:
- **Credibility:** Reports must be mathematically correct.
- **Business Value:** Demonstrating real savings to stakeholders.
- **Dashboard Accuracy:** The static HTML dashboard relies on these calculations.

---

## 2. Pricing Data Sources

| Data Source | Purpose | Update Frequency |
| :--- | :--- | :--- |
| **AWS Price List API** | Real-time pricing for all services and regions. | Daily (AWS updates) |
| **Hardcoded Pricing Map (v1)** | Static fallback for offline/local development. | Manual (as needed) |
| **AWS Cost Explorer** | Actual historical costs (planned for v2). | N/A |

**v1 Strategy:** The bot will use a **hardcoded pricing map** for EBS, EIPs, and Snapshots, with a fallback to the AWS Price List API if available. This ensures the bot works offline (with Floci) and in production without requiring additional API calls.

---

## 3. Supported Resource Types & Pricing Formulas

### 3.1 EBS Volumes

**Pricing Model:** EBS volumes are charged per GB-month of provisioned capacity, regardless of usage.

| Volume Type | us-east-1 Price (USD/GB-month) | Description |
| :--- | :--- | :--- |
| `gp3` | $0.08 | General Purpose SSD (default) |
| `gp2` | $0.10 | General Purpose SSD (older generation) |
| `io1` | $0.125 | Provisioned IOPS SSD (plus IOPS charges) |
| `io2` | $0.125 | Provisioned IOPS SSD (plus IOPS charges) |
| `st1` | $0.045 | Throughput Optimized HDD |
| `sc1` | $0.025 | Cold HDD |
| `standard` | $0.05 | Magnetic (legacy) |

**Formula:**
```
Monthly Savings (USD) = VolumeSize (GB) × PricePerGB (USD)
```

**Example:**
- Volume Type: `gp3`
- Size: 100 GB
- Price: $0.08/GB-month
- **Monthly Savings = 100 × 0.08 = $8.00**

**IOPS Charges (v2 Planned):**
- `io1` and `io2` volumes have additional IOPS charges ($0.065 per provisioned IOPS-month).
- v1 will ignore IOPS charges but log a warning: `WARN: IOPS charges not included for io1/io2 volumes`.
- v2 will calculate IOPS savings using the formula: `IOPS × 0.065`.

---

### 3.2 Elastic IPs (EIPs)

**Pricing Model:** EIPs are free when attached to a running EC2 instance. When unattached, they are charged at an hourly rate.

| Region | Price (USD/hour) | Price (USD/month) |
| :--- | :--- | :--- |
| All Commercial Regions | $0.005 | ~$3.60 |
| AWS GovCloud | $0.005 | ~$3.60 |

**Formula:**
```
Monthly Savings (USD) = HoursUnattached × PricePerHour
```

**Example:**
- EIP unattached for 720 hours (30 days)
- Price: $0.005/hour
- **Monthly Savings = 720 × 0.005 = $3.60**

---

### 3.3 EBS Snapshots

**Pricing Model:** Snapshots are charged per GB-month of data stored, based on the **actual data size** (not the volume size). Incremental snapshots store only changed blocks.

| Region | Price (USD/GB-month) |
| :--- | :--- |
| us-east-1 | $0.05 |
| us-west-2 | $0.05 |
| eu-west-1 | $0.05 |

**Formula:**
```
Monthly Savings (USD) = SnapshotDataSize (GB) × PricePerGB
```

**Example:**
- Snapshot Data Size: 50 GB
- Price: $0.05/GB-month
- **Monthly Savings = 50 × 0.05 = $2.50**

**Note:** The bot will use `SnapshotSize` (actual data size) from the `DescribeSnapshots` API, not the volume size. If `SnapshotSize` is unavailable, it falls back to `VolumeSize × 0.5` (estimated compression ratio).

---

### 3.4 RDS Instances (Stopped vs. Running)

**Pricing Model:** RDS instances are charged per hour based on instance class and storage. Stopping an instance saves the compute cost but **does not** save storage costs (EBS volumes remain).

**Important:** When an RDS instance is stopped, compute costs are saved, but **storage costs continue**. The bot only calculates savings on compute costs. Storage costs are **not** included in the savings estimate. This is conservative and safe.

**Savings Formula:**
```
Monthly Savings (USD) = InstanceHourlyCost × HoursStopped
```
*Note: Storage costs (EBS volumes) are not included in savings calculations.*

| Instance Class | us-east-1 Price (USD/hour) | Monthly Cost (Running) |
| :--- | :--- | :--- |
| `db.t4g.micro` | $0.012 | ~$8.64 |
| `db.t4g.small` | $0.024 | ~$17.28 |
| `db.t4g.medium` | $0.048 | ~$34.56 |
| `db.t3.micro` | $0.017 | ~$12.24 |
| `db.t3.small` | $0.034 | ~$24.48 |
| `db.t3.medium` | $0.068 | ~$48.96 |
| `db.t3.large` | $0.136 | ~$97.92 |
| `db.r6g.large` | $0.252 | ~$181.44 |
| `db.r6g.xlarge` | $0.504 | ~$362.88 |
| `db.r6g.2xlarge` | $1.008 | ~$725.76 |
| `db.r6g.4xlarge` | $2.016 | ~$1,451.52 |
| `db.m5.large` | $0.106 | ~$76.32 |
| `db.m5.xlarge` | $0.212 | ~$152.64 |
| `db.m5.2xlarge` | $0.424 | ~$305.28 |
| `db.x1e.xlarge` | $0.896 | ~$645.12 |
| `db.x1e.2xlarge` | $1.792 | ~$1,290.24 |

**Fallback:** If an instance class is not in the map, the bot will use `db.t3.micro` pricing and log a warning: `WARN: Unknown RDS instance class '{class}' - using db.t3.micro pricing`.

**Example:**
- Instance Class: `db.t3.micro`
- Hourly Cost: $0.017/hour
- Stopped for 720 hours (30 days)
- **Monthly Savings = 0.017 × 720 = $12.24**

**Constraint:** The bot will **only** stop RDS instances, never delete them. This prevents data loss while still saving compute costs.

---

## 4. Regional Pricing Map (Hardcoded v1)

The bot will use a hardcoded pricing map for regions that differ from the default. Prices are sourced from the AWS Price List API as of June 2026.

| Region | EBS gp3 (USD/GB-month) | EIP (USD/hour) | Snapshot (USD/GB-month) |
| :--- | :--- | :--- | :--- |
| `us-east-1` | $0.08 | $0.005 | $0.05 |
| `us-east-2` | $0.08 | $0.005 | $0.05 |
| `us-west-1` | $0.08 | $0.005 | $0.05 |
| `us-west-2` | $0.08 | $0.005 | $0.05 |
| `eu-west-1` | $0.08 | $0.005 | $0.05 |
| `eu-west-2` | $0.08 | $0.005 | $0.05 |
| `eu-central-1` | $0.08 | $0.005 | $0.05 |
| `ap-southeast-1` | $0.08 | $0.005 | $0.05 |
| `ap-southeast-2` | $0.08 | $0.005 | $0.05 |
| `ap-northeast-1` | $0.08 | $0.005 | $0.05 |

**Note on Non-Standard Regions:**
- Regions not listed in the map (e.g., `sa-east-1`, `ap-south-1`) will fall back to the default price (`0.08`).
- A warning will be logged: `WARN: Unknown region '{region}' - using default pricing`.
- To override, set the `COST_PER_GB` environment variable.

---

## 5. Go Implementation: Pricing Engine

### 5.1 Pricing Map Structure

```go
package pricing

type PricingMap struct {
    EBS map[string]float64      // VolumeType -> PricePerGB
    EIP map[string]float64      // Region -> PricePerHour
    Snapshot map[string]float64 // Region -> PricePerGB
    RDS map[string]float64      // InstanceClass -> HourlyCost
}

func GetDefaultPricing() PricingMap {
    return PricingMap{
        EBS: map[string]float64{
            "gp3":       0.08,
            "gp2":       0.10,
            "io1":       0.125,
            "io2":       0.125,
            "st1":       0.045,
            "sc1":       0.025,
            "standard":  0.05,
        },
        EIP: map[string]float64{
            "us-east-1":   0.005,
            "us-east-2":   0.005,
            "us-west-1":   0.005,
            "us-west-2":   0.005,
            "eu-west-1":   0.005,
            "eu-west-2":   0.005,
            "eu-central-1": 0.005,
            "ap-southeast-1": 0.005,
            "ap-southeast-2": 0.005,
            "ap-northeast-1": 0.005,
        },
        Snapshot: map[string]float64{
            "us-east-1":   0.05,
            "us-east-2":   0.05,
            "us-west-1":   0.05,
            "us-west-2":   0.05,
            "eu-west-1":   0.05,
            "eu-west-2":   0.05,
            "eu-central-1": 0.05,
            "ap-southeast-1": 0.05,
            "ap-southeast-2": 0.05,
            "ap-northeast-1": 0.05,
        },
        RDS: map[string]float64{
            "db.t4g.micro":   0.012,
            "db.t4g.small":   0.024,
            "db.t4g.medium":  0.048,
            "db.t3.micro":    0.017,
            "db.t3.small":    0.034,
            "db.t3.medium":   0.068,
            "db.t3.large":    0.136,
            "db.r6g.large":   0.252,
            "db.r6g.xlarge":  0.504,
            "db.r6g.2xlarge": 1.008,
            "db.r6g.4xlarge": 2.016,
            "db.m5.large":    0.106,
            "db.m5.xlarge":   0.212,
            "db.m5.2xlarge":  0.424,
            "db.x1e.xlarge":  0.896,
            "db.x1e.2xlarge": 1.792,
        },
    }
}
```

### 5.2 Environment Variable Override

The pricing engine will check for the `COST_PER_GB` environment variable and use it to override the default EBS pricing map:

```go
func GetPricingMap() PricingMap {
    pricingMap := GetDefaultPricing()
    
    // Check for environment variable override
    costPerGB := os.Getenv("COST_PER_GB")
    if costPerGB != "" {
        if val, err := strconv.ParseFloat(costPerGB, 64); err == nil {
            // Override all EBS volume types with the custom value
            for volumeType := range pricingMap.EBS {
                pricingMap.EBS[volumeType] = val
            }
        }
    }
    
    // Check if RDS savings should be enabled
    enableRDS := os.Getenv("ENABLE_RDS_SAVINGS")
    if enableRDS == "false" {
        // RDS savings are disabled; clear the RDS map
        pricingMap.RDS = map[string]float64{}
    }
    
    return pricingMap
}
```

### 5.3 Core Calculation Functions

```go
// CalculateEBSSavings calculates monthly savings for an EBS volume.
func CalculateEBSSavings(volumeType string, sizeGB int, region string, priceMap PricingMap) float64 {
    pricePerGB, ok := priceMap.EBS[volumeType]
    if !ok {
        // Fallback to gp3 pricing for unknown types
        pricePerGB = 0.08
        log.Warnf("Unknown volume type '%s' - using gp3 pricing", volumeType)
    }
    return float64(sizeGB) * pricePerGB
}

// CalculateEIPSavings calculates monthly savings for an unattached Elastic IP.
func CalculateEIPSavings(hoursUnattached float64, region string, priceMap PricingMap) float64 {
    pricePerHour, ok := priceMap.EIP[region]
    if !ok {
        // Fallback to default price for unknown regions
        pricePerHour = 0.005
        log.Warnf("Unknown region '%s' for EIP - using default pricing", region)
    }
    return hoursUnattached * pricePerHour
}

// CalculateSnapshotSavings calculates monthly savings for an EBS snapshot.
func CalculateSnapshotSavings(snapshotDataSizeGB float64, volumeSizeGB int, region string, priceMap PricingMap) float64 {
    dataSize := snapshotDataSizeGB
    if dataSize <= 0 {
        // Fallback: assume 50% compression (VolumeSize × 0.5)
        dataSize = float64(volumeSizeGB) * 0.5
        log.Warn("SnapshotDataSize unavailable, using estimated compression ratio (VolumeSize × 0.5)")
    }
    pricePerGB, ok := priceMap.Snapshot[region]
    if !ok {
        pricePerGB = 0.05
        log.Warnf("Unknown region '%s' for snapshot - using default pricing", region)
    }
    return dataSize * pricePerGB
}

// CalculateRDSSavings calculates monthly savings for stopping an RDS instance.
func CalculateRDSSavings(instanceClass string, hoursStopped float64, priceMap PricingMap) float64 {
    hourlyCost, ok := priceMap.RDS[instanceClass]
    if !ok {
        // Fallback to t3.micro pricing for unknown classes
        hourlyCost = 0.017
        log.Warnf("Unknown RDS instance class '%s' - using db.t3.micro pricing", instanceClass)
    }
    return hoursStopped * hourlyCost
}
```

### 5.4 Savings Aggregation

```go
type SavingsReport struct {
    TotalMonthlySavings float64
    EBSSavings float64
    EIPSavings float64
    SnapshotSavings float64
    RDSSavings float64
    Resources map[string]ResourceSavings // ResourceId -> Savings
}

type ResourceSavings struct {
    ResourceType string
    MonthlySavings float64
    ActionTaken string // "DELETED" or "STOPPED"
}

func AggregateSavings(
    deletedVolumes []EC2Volume,
    releasedEIPs []EIP,
    deletedSnapshots []Snapshot,
    stoppedRDS []RDSInstance,
    region string,
    priceMap PricingMap,
) SavingsReport {
    report := SavingsReport{
        Resources: make(map[string]ResourceSavings),
    }

    for _, vol := range deletedVolumes {
        savings := CalculateEBSSavings(vol.VolumeType, vol.SizeGB, region, priceMap)
        report.EBSSavings += savings
        report.TotalMonthlySavings += savings
        report.Resources[vol.VolumeID] = ResourceSavings{
            ResourceType:   "EBS_VOLUME",
            MonthlySavings: savings,
            ActionTaken:    "DELETED",
        }
    }

    // ... Similar loops for EIPs, Snapshots, and RDS ...

    return report
}
```

---

## 6. AWS Price List API Integration (Optional v1.5)

For production environments, the bot can optionally fetch real-time pricing from the AWS Price List API. This ensures accuracy without manual updates.

```go
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/service/pricing"
)

func FetchPricingFromAWS(ctx context.Context, client *pricing.Client) (PricingMap, error) {
    // Implementation not required for v1; will be added in v1.5
}
```

---

## 7. Pricing Engine Validation

### 7.1 Unit Tests

```go
func TestCalculateEBSSavings(t *testing.T) {
    priceMap := GetDefaultPricing()
    
    savings := CalculateEBSSavings("gp3", 100, "us-east-1", priceMap)
    expected := 8.0 // 100 * 0.08
    
    if savings != expected {
        t.Errorf("Expected %f, got %f", expected, savings)
    }
}

func TestCalculateEBSSavings_UnknownVolumeType(t *testing.T) {
    priceMap := GetDefaultPricing()
    
    savings := CalculateEBSSavings("unknown", 100, "us-east-1", priceMap)
    expected := 8.0 // Fallback to gp3 pricing
    
    if savings != expected {
        t.Errorf("Expected %f, got %f", expected, savings)
    }
}

func TestCalculateSnapshotSavings_Fallback(t *testing.T) {
    priceMap := GetDefaultPricing()
    
    // SnapshotDataSize unavailable, VolumeSize = 100GB
    savings := CalculateSnapshotSavings(0, 100, "us-east-1", priceMap)
    expected := 2.5 // 100 * 0.5 * 0.05
    
    if savings != expected {
        t.Errorf("Expected %f, got %f", expected, savings)
    }
}

func TestCalculateRDSSavings_UnknownInstanceClass(t *testing.T) {
    priceMap := GetDefaultPricing()
    
    savings := CalculateRDSSavings("db.unknown", 720, priceMap)
    expected := 12.24 // 720 * 0.017 (fallback to db.t3.micro)
    
    if savings != expected {
        t.Errorf("Expected %f, got %f", expected, savings)
    }
}
```

### 7.2 Integration Tests (with Floci)

```go
// TestCalculateSavingsWithFloci spins up Floci and tests that the pricing engine 
// correctly calculates savings for resources discovered in the emulated environment.
func TestCalculateSavingsWithFloci(t *testing.T) {
    // Start Floci, create test resources, run the bot, verify savings.
}
```

---

## 8. Reporting Output Format

The pricing engine will generate a report in the following format for the S3 audit bucket and Slack notifications:

```json
{
  "timestamp": "2026-06-19T02:00:00Z",
  "region": "us-east-1",
  "total_monthly_savings": 45.82,
  "breakdown": {
    "ebs_volumes": {
      "count": 5,
      "savings": 32.00
    },
    "elastic_ips": {
      "count": 2,
      "savings": 7.20
    },
    "snapshots": {
      "count": 3,
      "savings": 4.62
    },
    "rds_instances": {
      "count": 1,
      "savings": 12.00
    }
  },
  "resources": [
    {
      "resource_id": "vol-12345",
      "resource_type": "EBS_VOLUME",
      "action_taken": "DELETED",
      "monthly_savings": 8.00
    }
  ]
}
```

**Currency Display Standard:**
- All monetary values will be displayed with two decimal places (e.g., `$32.50`).
- Values will be truncated to two decimal places (not rounded, to avoid overstating savings).
- For Slack notifications: `$%.2f` format.
- For the HTML dashboard: `$%.2f` with Chart.js formatting.

---

## 9. Cost Estimation Edge Cases

| Scenario | Handling |
| :--- | :--- |
| **Unknown Volume Type** | Fallback to `gp3` pricing ($0.08/GB-month) and log a warning. |
| **Unknown Region** | Fallback to `us-east-1` pricing and log a warning. |
| **Missing Snapshot Data Size** | Fallback to `VolumeSize × 0.5` (estimated compression ratio) and log a warning. |
| **Unknown RDS Instance Class** | Fallback to `db.t3.micro` pricing ($0.017/hour) and log a warning. |
| **EIP in GovCloud** | Same pricing as commercial regions ($0.005/hour). |
| **Free Tier Resources** | v1 does not adjust for Free Tier. It calculates list prices (overestimating savings for Free Tier-eligible resources). v2 will account for Free Tier consumption. |
| **RDS Storage Costs** | Storage costs continue when RDS is stopped. The bot **excludes** storage from savings calculations (conservative). |

---

## 10. Sign-Off

| Role | Name | Date | Signature |
| :--- | :--- | :--- | :--- |
| **Project Lead / Architect** | Jibrin Ahmed | June 19, 2026 | JA |
| **Technical Reviewer** | [DeepSeek / Imaginary CTO] | [Date] | [Initials] |

---
