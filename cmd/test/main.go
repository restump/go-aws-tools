package main

import (
	"context"
	// "fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/restump/go-aws-tools/scan"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"internal/helpers"
)

var cfg aws.Config
// var log *zerolog.Logger
var pool *scan.WorkerPool

func ListSecurityGroups(region string, input interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := scan.NewResourcesResult()
	result.Region = region
	client := ec2.NewFromConfig(cfg, func(o *ec2.Options) {
		o.Region = region
	})

	pager := ec2.NewDescribeSecurityGroupsPaginator(
		client, 
		input.(*ec2.DescribeSecurityGroupsInput),
	)
	for pager.HasMorePages() {
		log.Info().Msg("a page")
		page, err := pager.NextPage(ctx)
		if err != nil {
			pool.AddResult(&scan.ErrorResult{ErrorString: err.Error()})
			return
		}

		for _, group := range page.SecurityGroups {
			result.AddResource(group)
		}
	}

	pool.AddResult(result)
}

func ListNetworkInterfaces(region string, input interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := scan.NewResourcesResult()
	result.Region = region
	client := ec2.NewFromConfig(cfg, func(o *ec2.Options) {
		o.Region = region
	})

	pager := ec2.NewDescribeNetworkInterfacesPaginator(
		client, 
		input.(*ec2.DescribeNetworkInterfacesInput),
	)
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			pool.AddResult(&scan.ErrorResult{ErrorString: err.Error()})
			return
		}

		for _, eni := range page.NetworkInterfaces {
			result.AddResource(eni)
		}
	}

	pool.AddResult(result)
}

func main() {
	// initialize logger
	// log := helpers.NewLog()
	zerolog.TimeFieldFormat = time.RFC3339

	// aws session
	region := helpers.LookupEnvWithDefault("AWS_REGION", "us-east-1")
	cfg, _ = config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))

	pool = scan.NewWorkerPool(1000000)

	// create workers
	for i := 0; i < 64; i++ {
		pool.AddWorker()
	}

	// assign workers work
	regions, err := helpers.GetEnabledRegions(cfg)
	if err != nil {
		log.Error().Err(err).Msg("an error")
	}

	for _, region := range regions {
		pool.AddWork(&scan.Work{
			WorkFn: ListSecurityGroups,
			WorkIn: scan.WorkInput{
				Input: &ec2.DescribeSecurityGroupsInput{},
				Region: region,
			},
		})

		pool.AddWork(&scan.Work{
			WorkFn: ListNetworkInterfaces,
			WorkIn: scan.WorkInput{
				Input: &ec2.DescribeNetworkInterfacesInput{},
				Region: region,
			},
		})
	}

	pool.CloseWorkers()
	pool.Wait()
	pool.CloseResults()

	counts := make(map[string]map[string]int, 0)
	for result := range pool.Results {
		if _, ok := counts[result.GetRegion()]; !ok {
			counts[result.GetRegion()] = make(map[string]int, 0)
		}

		for _, resource := range result.Resources() {
			// fmt.Printf("%T\n", resource)
			switch resource.(type) {
			case types.NetworkInterface:
				if _, ok := counts[result.GetRegion()]["NetworkInterface"]; !ok {
					counts[result.GetRegion()]["NetworkInterface"] = 0
				}

				counts[result.GetRegion()]["NetworkInterface"]++
			case types.SecurityGroup:
				if _, ok := counts[result.GetRegion()]["SecurityGroup"]; !ok {
					counts[result.GetRegion()]["SecurityGroup"] = 0
				}

				counts[result.GetRegion()]["SecurityGroup"]++	
			}
			// counts[result.GetRegion()]++
			// log.Info().Interface("resource", resource).Msg("resource")
		}
	}

	for region, _ := range counts {
		for k, v := range counts[region] {
			log.Info().Str("region", region).Str("resource", k).
				Int("count", v).Msg("resource count")
		}
	}

	return
}