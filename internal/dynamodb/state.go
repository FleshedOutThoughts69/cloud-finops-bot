// internal/dynamodb/state.go

package dynamodb

import (
    "context"
    "fmt"
    "strconv"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

    "cloud-finops-bot/pkg/logger"
)

// GetState retrieves a resource state from DynamoDB.
// Returns nil, nil if the item does not exist.
func GetState(ctx context.Context, client *dynamodb.Client, tableName, resourceID, accountID string) (*ResourceState, error) {
    logger.Debug(ctx, "Getting state",
        "table", tableName,
        "resource_id", resourceID,
        "account_id", accountID,
    )

    input := &dynamodb.GetItemInput{
        TableName: aws.String(tableName),
        Key: map[string]types.AttributeValue{
            "ResourceId": &types.AttributeValueMemberS{Value: resourceID},
            "AccountId":  &types.AttributeValueMemberS{Value: accountID},
        },
    }

    result, err := client.GetItem(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("GetItem failed: %w", err)
    }

    if result.Item == nil {
        logger.Debug(ctx, "State not found")
        return nil, nil
    }

    var state ResourceState
    if err := attributevalue.UnmarshalMap(result.Item, &state); err != nil {
        return nil, fmt.Errorf("failed to unmarshal state: %w", err)
    }

    logger.Debug(ctx, "State retrieved",
        "resource_id", state.ResourceID,
        "action_taken", state.ActionTaken,
    )
    return &state, nil
}

// PutState writes a resource state to DynamoDB with optimistic locking.
// If the item already exists, it will only succeed if the Version matches.
// If the item does not exist, it will create it with Version = 1.
func PutState(ctx context.Context, client *dynamodb.Client, tableName string, state ResourceState) error {
    logger.Debug(ctx, "Putting state",
        "table", tableName,
        "resource_id", state.ResourceID,
        "account_id", state.AccountID,
        "action_taken", state.ActionTaken,
    )

    // Marshal the state into a DynamoDB item
    item, err := attributevalue.MarshalMap(state)
    if err != nil {
        return fmt.Errorf("failed to marshal state: %w", err)
    }

    // Optimistic locking: require that either the item doesn't exist,
    // or the version matches the provided version.
    input := &dynamodb.PutItemInput{
        TableName: aws.String(tableName),
        Item:      item,
        ConditionExpression: aws.String(
            "attribute_not_exists(ResourceId) OR Version = :version",
        ),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":version": &types.AttributeValueMemberN{Value: strconv.Itoa(state.Version)},
        },
    }

    _, err = client.PutItem(ctx, input)
    if err != nil {
        // Check if it's a conditional check failure
        if _, ok := err.(*types.ConditionalCheckFailedException); ok {
            return fmt.Errorf("optimistic locking failed: version mismatch")
        }
        return fmt.Errorf("PutItem failed: %w", err)
    }

    logger.Debug(ctx, "State written successfully",
        "resource_id", state.ResourceID,
        "action_taken", state.ActionTaken,
    )
    return nil
}

// UpdateState updates an existing resource state, incrementing the Version.
// It uses optimistic locking to prevent concurrent modifications.
func UpdateState(ctx context.Context, client *dynamodb.Client, tableName string, state ResourceState) error {
    logger.Debug(ctx, "Updating state",
        "table", tableName,
        "resource_id", state.ResourceID,
        "account_id", state.AccountID,
        "action_taken", state.ActionTaken,
    )

    // Marshal the state into a DynamoDB item (excluding ResourceId and AccountId from update)
    item, err := attributevalue.MarshalMap(state)
    if err != nil {
        return fmt.Errorf("failed to marshal state: %w", err)
    }

    // Remove the key attributes from the update expression (they are used in the key)
    // We'll build an update expression dynamically.
    // For simplicity, we'll use PutItem with conditional version check.
    // Alternatively, we can use UpdateItem with SET expressions.
    // For simplicity, we'll use PutItem with version check (same as PutState but with version increment).
    // We can also increment version here.

    // Increment version for update
    state.Version++
    item["Version"] = &types.AttributeValueMemberN{Value: strconv.Itoa(state.Version)}

    input := &dynamodb.PutItemInput{
        TableName: aws.String(tableName),
        Item:      item,
        ConditionExpression: aws.String(
            "Version = :old_version",
        ),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":old_version": &types.AttributeValueMemberN{Value: strconv.Itoa(state.Version - 1)},
        },
    }

    _, err = client.PutItem(ctx, input)
    if err != nil {
        if _, ok := err.(*types.ConditionalCheckFailedException); ok {
            return fmt.Errorf("optimistic locking failed: version mismatch")
        }
        return fmt.Errorf("UpdateState PutItem failed: %w", err)
    }

    logger.Debug(ctx, "State updated successfully",
        "resource_id", state.ResourceID,
        "new_version", state.Version,
    )
    return nil
}

// QueryExpiredQuarantines queries all resources with ActionTaken = "QUARANTINED"
// where QuarantineExpiry < currentTime.
func QueryExpiredQuarantines(ctx context.Context, client *dynamodb.Client, tableName string, currentTime int64) ([]ResourceState, error) {
    logger.Debug(ctx, "Querying expired quarantines",
        "table", tableName,
        "current_time", currentTime,
    )

    input := &dynamodb.QueryInput{
        TableName:              aws.String(tableName),
        IndexName:              aws.String("QuarantineExpiry-Index"),
        KeyConditionExpression: aws.String("ActionTaken = :action AND QuarantineExpiry < :time"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":action": &types.AttributeValueMemberS{Value: "QUARANTINED"},
            ":time":   &types.AttributeValueMemberN{Value: strconv.FormatInt(currentTime, 10)},
        },
    }

    result, err := client.Query(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("Query failed: %w", err)
    }

    var states []ResourceState
    if err := attributevalue.UnmarshalListOfMaps(result.Items, &states); err != nil {
        return nil, fmt.Errorf("failed to unmarshal results: %w", err)
    }

    logger.Debug(ctx, "Expired quarantines found",
        "count", len(states),
    )
    return states, nil
}

// QueryDeletedResources queries all resources with ActionTaken = "DELETED" or "STOPPED"
// where DeletionTimestamp >= startTime (for dashboard reporting).
func QueryDeletedResources(ctx context.Context, client *dynamodb.Client, tableName string, startTime int64) ([]ResourceState, error) {
    logger.Debug(ctx, "Querying deleted resources",
        "table", tableName,
        "start_time", startTime,
    )

    input := &dynamodb.QueryInput{
        TableName:              aws.String(tableName),
        IndexName:              aws.String("ActionTaken-Index"),
        KeyConditionExpression: aws.String("ActionTaken IN (:del, :stop) AND DeletionTimestamp >= :time"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":del":  &types.AttributeValueMemberS{Value: "DELETED"},
            ":stop": &types.AttributeValueMemberS{Value: "STOPPED"},
            ":time": &types.AttributeValueMemberN{Value: strconv.FormatInt(startTime, 10)},
        },
    }

    result, err := client.Query(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("Query failed: %w", err)
    }

    var states []ResourceState
    if err := attributevalue.UnmarshalListOfMaps(result.Items, &states); err != nil {
        return nil, fmt.Errorf("failed to unmarshal results: %w", err)
    }

    logger.Debug(ctx, "Deleted resources found",
        "count", len(states),
    )
    return states, nil
}