// internal/ec2/client.go
package ec2

import (
    "context"
    "os"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
)

func NewClient(ctx context.Context, region string) (*ec2.Client, error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    endpointURL := os.Getenv("AWS_ENDPOINT_URL")

    opts := []func(*config.LoadOptions) error{
        config.WithRegion(region),
    }

    // If using Floci (local endpoint), use dummy credentials
    if endpointURL != "" {
        opts = append(opts,
            config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
            config.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
                return aws.Endpoint{
                    URL:           endpointURL,
                    SigningRegion: region,
                }, nil
            })),
        )
    }

    awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
    if err != nil {
        return nil, err
    }

    return ec2.NewFromConfig(awsCfg), nil
}