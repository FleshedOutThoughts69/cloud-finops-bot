// internal/dynamodb/client.go

package dynamodb

import (
    "context"
    "os"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// NewClient creates a DynamoDB client with optional Floci endpoint override.
func NewClient(ctx context.Context, region string) (*dynamodb.Client, error) {
    endpointURL := os.Getenv("AWS_ENDPOINT_URL")

    opts := []func(*config.LoadOptions) error{
        config.WithRegion(region),
    }

    if endpointURL != "" {
        opts = append(opts, config.WithEndpointResolver(
            aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
                return aws.Endpoint{
                    URL:           endpointURL,
                    SigningRegion: region,
                }, nil
            }),
        ))
    }

    awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
    if err != nil {
        return nil, err
    }

    return dynamodb.NewFromConfig(awsCfg), nil
}