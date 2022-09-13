package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func GetCallerIdentity(c aws.Config) (*sts.GetCallerIdentityOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := sts.NewFromConfig(c)

	res, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("GetCallerIdentity: %w", err)
	}
	return res, nil
}

func GetEnabledRegions(c aws.Config) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	regions := make([]string, 0)

	client := ec2.NewFromConfig(c)
	inputs := &ec2.DescribeRegionsInput{
		Filters: []ec2types.Filter{
			{
				Name: aws.String("opt-in-status"),
				Values: []string{
					"opt-in-not-required",
					"opted-in",
				},
			},
		},
	}

	res, err := client.DescribeRegions(ctx, inputs)
	if err != nil {
		return regions, fmt.Errorf("GetEnabledRegions: %w", err)
	}

	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	return regions, nil
}
