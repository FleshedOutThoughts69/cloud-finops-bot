// tests/integration/dynamodb_test.go

package integration

import (
    "context"
    "testing"

    "cloud-finops-bot/internal/dynamodb"
)

func TestDynamoDBState(t *testing.T) {
    // This test will run with Floci (if running).
    // We'll just check that we can connect and get a state.
    ctx := context.Background()
    client, err := dynamodb.NewClient(ctx, "us-east-1")
    if err != nil {
        t.Skip("Skipping test: DynamoDB client not available")
    }

    // Try to get a state that doesn't exist
    state, err := dynamodb.GetState(ctx, client, "FinOps-State-dev", "vol-test", "123")
    if err != nil {
        t.Fatalf("GetState failed: %v", err)
    }
    if state != nil {
        t.Errorf("Expected nil state, got %+v", state)
    }
}