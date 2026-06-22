// cmd/generate_dashboard.go

package main

import (
    "fmt"
    "os"
    "time"

    "cloud-finops-bot/internal/dashboard"
    "cloud-finops-bot/internal/dynamodb"
)

// This is a helper to generate a dashboard from a JSON file of records.
// Usage: go run cmd/generate_dashboard.go -input records.json -output dashboard.html

func main() {
    // For simplicity, we'll just generate a sample dashboard with dummy data.
    records := []dynamodb.ResourceState{
        {
            ResourceID:        "vol-abc123",
            ActionTaken:       "DELETED",
            ResourceType:      "EBS_VOLUME",
            EstimatedSavings:  float64Ptr(8.0),
            DeletionTimestamp: int64Ptr(time.Now().AddDate(0, -1, 0).Unix()),
        },
        {
            ResourceID:        "eip-xyz789",
            ActionTaken:       "DELETED",
            ResourceType:      "EIP",
            EstimatedSavings:  float64Ptr(3.6),
            DeletionTimestamp: int64Ptr(time.Now().AddDate(0, -1, -5).Unix()),
        },
    }

    totalSavings := 11.6
    html := dashboard.GenerateDashboardHTML(records, totalSavings, time.Now(), "healthy")

    err := os.WriteFile("dashboard.html", []byte(html), 0644)
    if err != nil {
        fmt.Println("Error writing dashboard:", err)
        os.Exit(1)
    }
    fmt.Println("Dashboard generated: dashboard.html")
}

func float64Ptr(f float64) *float64 { return &f }
func int64Ptr(i int64) *int64       { return &i }