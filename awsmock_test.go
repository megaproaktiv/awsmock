package awsmock_test

import (
	"context"
	"testing"

	"github.com/megaproaktiv/awsmock"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/aws/aws-sdk-go/aws"
	"gotest.tools/v3/assert"
)

func TestGetConfig(t *testing.T) {

	AssumeRoleFunc := func(ctx context.Context, params *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
		role := *params.RoleArn
		assert.Equal(t, "arn:aws:iam::12345679812:role/aggregate", role)
		return &sts.AssumeRoleOutput{
			Credentials: &types.Credentials{
				AccessKeyId:     aws.String("AKAMAI123"),
				SecretAccessKey: aws.String("verysecret"),
				SessionToken:    aws.String("tokentoken"),
			},
		}, nil
	}

	// Create a Mock Handler
	mockCfg := awsmock.NewAwsMockHandler()
	// add a function to the handler
	// Routing per paramater types
	mockCfg.AddHandler(AssumeRoleFunc)

	client := sts.NewFromConfig(mockCfg.AwsConfig())

	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String("arn:aws:iam::12345679812:role/aggregate"),
		RoleSessionName: aws.String("session"),
	}
	response, err := client.AssumeRole(context.TODO(), input)
	assert.NilError(t, err)
	assert.Equal(t, "AKAMAI123", *response.Credentials.AccessKeyId)
}
