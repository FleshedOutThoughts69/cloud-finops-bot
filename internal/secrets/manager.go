// internal/secrets/manager.go

package secrets

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
    "github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SlackSecret defines the structure of the secret stored in Secrets Manager.
type SlackSecret struct {
    WebhookURL string `json:"SLACK_WEBHOOK_URL"`
}

// GetSlackWebhook fetches the Slack webhook URL from AWS Secrets Manager.
// Returns an empty string if the secret is not found or error occurs.
func GetSlackWebhook(ctx context.Context, client *secretsmanager.Client, secretARN string) (string, error) {
    if secretARN == "" {
        return "", fmt.Errorf("secret ARN is empty")
    }

    input := &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretARN),
    }

    result, err := client.GetSecretValue(ctx, input)
    if err != nil {
        return "", fmt.Errorf("failed to get secret: %w", err)
    }

    var secret SlackSecret
    if err := json.Unmarshal([]byte(*result.SecretString), &secret); err != nil {
        return "", fmt.Errorf("failed to unmarshal secret: %w", err)
    }

    if secret.WebhookURL == "" {
        return "", fmt.Errorf("secret does not contain SLACK_WEBHOOK_URL")
    }

    return secret.WebhookURL, nil
}

// GetParameter fetches a parameter from SSM Parameter Store.
func GetParameter(ctx context.Context, client *ssm.Client, name string) (string, error) {
    if name == "" {
        return "", fmt.Errorf("parameter name is empty")
    }

    input := &ssm.GetParameterInput{
        Name:           aws.String(name),
        WithDecryption: aws.Bool(true),
    }

    result, err := client.GetParameter(ctx, input)
    if err != nil {
        return "", fmt.Errorf("failed to get parameter %s: %w", name, err)
    }

    if result.Parameter == nil || result.Parameter.Value == nil {
        return "", fmt.Errorf("parameter %s is empty", name)
    }

    return *result.Parameter.Value, nil
}

// GetParametersByPath fetches multiple parameters from SSM by path.
// Returns a map of parameter names to values.
func GetParametersByPath(ctx context.Context, client *ssm.Client, path string, recursive bool) (map[string]string, error) {
    if path == "" {
        return nil, fmt.Errorf("path is empty")
    }

    input := &ssm.GetParametersByPathInput{
        Path:           aws.String(path),
        Recursive:      aws.Bool(recursive),
        WithDecryption: aws.Bool(true),
    }

    result, err := client.GetParametersByPath(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to get parameters by path %s: %w", path, err)
    }

    params := make(map[string]string)
    for _, p := range result.Parameters {
        if p.Name != nil && p.Value != nil {
            // Extract the key (last part of the path)
            key := *p.Name
            // Remove the path prefix if needed
            params[key] = *p.Value
        }
    }

    return params, nil
}