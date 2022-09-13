package main

import (
	"os"
	"fmt"
	"sort"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/iam"
)

// var wg sync.WaitGroup

func getCallerIdentity() sts.GetCallerIdentityOutput {
	stsClient := sts.New(session.New())
	res, err  := stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		fmt.Println("sts:GetCallerIdentity error:", err.Error())
		os.Exit(1)
	}

	return *res
}

func getOrganization() organizations.Organization {
	callerIdentity := getCallerIdentity()

	sess := session.Must(session.NewSession())
	role := fmt.Sprintf("arn:aws:iam::%s:role/CloudCoreRO", *callerIdentity.Account)
	cred := stscreds.NewCredentials(sess, role)
	orgClient := organizations.New(sess, &aws.Config{Credentials: cred})

	res, err := orgClient.DescribeOrganization(&organizations.DescribeOrganizationInput{})
	if err != nil {
		fmt.Println("organizations:DescribeOrganization error:", err.Error())
		os.Exit(1)
	}

	return *res.Organization
}

func getOrganizationAccounts() []organizations.Account {
	var accounts []organizations.Account

	organization := getOrganization()
	
	sess := session.Must(session.NewSession())
	role := fmt.Sprintf("arn:aws:iam::%s:role/CloudCoreRO", *organization.MasterAccountId)
	cred := stscreds.NewCredentials(sess, role)
	orgClient := organizations.New(sess, &aws.Config{Credentials: cred})
	
	err := orgClient.ListAccountsPages(&organizations.ListAccountsInput{}, 
		func(page *organizations.ListAccountsOutput, lastPage bool) bool {
			for _, account := range page.Accounts {
				accounts = append(accounts, *account)
			}
			return true 
		})
	if err != nil {
		fmt.Println("organizations:ListAccounts error:", err.Error())
		os.Exit(1)
	}

	sort.Slice(accounts, func(i, j int) bool {
		return *accounts[i].Id < *accounts[j].Id
	})
	return accounts
}

func main() {
	accounts := getOrganizationAccounts()

	input := &iam.CreateServiceLinkedRoleInput{
		AWSServiceName: aws.String("transitgateway.amazonaws.com"),
	}

	for _, account := range accounts {
		sess := session.Must(session.NewSession())
		role := fmt.Sprintf("arn:aws:iam::%s:role/CloudCoreEng", *account.Id)
		cred := stscreds.NewCredentials(sess, role)
		iamClient := iam.New(sess, &aws.Config{Credentials: cred})
		
		_, err := iamClient.CreateServiceLinkedRole(input)
		if err != nil {
			fmt.Printf("[%s] iam:CreateServiceLinkedRole error: %s\n", *account.Id, err.Error())
		}
	}
}