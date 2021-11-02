
Taken from https://gist.github.com/Cyberax

## Example

```go
func TestGetConfig(t *testing.T){

	AssumeRoleFunc := func(ctx context.Context, params *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {	
			role := *params.RoleArn
			assert.Equal(t,"arn:aws:iam::12345679812:role/aggregate",role)
			return &sts.AssumeRoleOutput{
				Credentials: &types.Credentials{
					AccessKeyId: aws.String("AKAMAI123"),
					SecretAccessKey: aws.String("verysecret"),
					SessionToken: aws.String("tokentoken"),
				},
				},nil
	}
	
	// Create a Mock Handler
	mockCfg := utils.NewAwsMockHandler()
	// add a function to the handler
	// Routing per paramater types
	mockCfg.AddHandler(AssumeRoleFunc)

	client := sts.NewFromConfig(mockCfg.AwsConfig())

	newConfig,_ := aggregate.GetCfgSub(client,"12345679812" )
	creds, _ := newConfig.Credentials.Retrieve(context.TODO())
	assert.Equal(t, "AKAMAI123",creds.AccessKeyID)
}
```