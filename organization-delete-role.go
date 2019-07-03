package main

import (
	"os"
	"fmt"
	"flag"
	"sort"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/organizations"
)


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

func deleteRole(accountId string, roleName string) {
	sess := session.Must(session.NewSession())
	role := fmt.Sprintf("arn:aws:iam::%s:role/CloudCoreEng", accountId)
	cred := stscreds.NewCredentials(sess, role)
	iamClient := iam.New(sess, &aws.Config{Credentials: cred})

	_, err := iamClient.GetRole(&iam.GetRoleInput{RoleName: &roleName})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Printf("[%s] Role '%s' not present\n", accountId, roleName)
				return
			default:
				fmt.Println("GetRole error: %s", aerr.Error())
				os.Exit(1)
			}
		}
	}

	fmt.Printf("[%s] Deleting role '%s'...\n", accountId, roleName)
	// Inline Policies
	var inlinePolicyNames []string
	err = iamClient.ListRolePoliciesPages(&iam.ListRolePoliciesInput{RoleName: &roleName}, 
		func(page *iam.ListRolePoliciesOutput, lastPage bool) bool {
			for _, policyName := range page.PolicyNames {
				inlinePolicyNames = append(inlinePolicyNames, *policyName)
			}
			return true 
		})
	if err != nil {
		fmt.Println("iam:ListRolePolicies error:", err.Error())
		os.Exit(1)
	}

	for _, policyName := range inlinePolicyNames {
		fmt.Printf("[%s]   Deleting inline policy '%s'...\n", accountId, policyName)
		_, err := iamClient.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
			RoleName: &roleName, 
			PolicyName: &policyName,
		})
		if err != nil {
			fmt.Printf("iam:DeleteRolePolicy [%s] error: %s\n", policyName, err.Error())
			os.Exit(1)
		}
	}

	// Attached Policies
	var attachedPolicies []iam.AttachedPolicy
	err = iamClient.ListAttachedRolePoliciesPages(&iam.ListAttachedRolePoliciesInput{RoleName: &roleName}, 
		func(page *iam.ListAttachedRolePoliciesOutput, lastPage bool) bool {
			for _, attachedPolicy := range page.AttachedPolicies {
				attachedPolicies = append(attachedPolicies, *attachedPolicy)
			}
			return true 
		})
	if err != nil {
		fmt.Println("iam:ListAttachedRolePolicies error:", err.Error())
		os.Exit(1)
	}

	for _, attachedPolicy := range attachedPolicies {
		fmt.Printf("[%s]   Detaching policy '%s'...\n", accountId, *attachedPolicy.PolicyName)
		_, err := iamClient.DetachRolePolicy(&iam.DetachRolePolicyInput{
			RoleName: &roleName, 
			PolicyArn: attachedPolicy.PolicyArn,
		})
		if err != nil {
			fmt.Printf("iam:DetachRolePolicy [%s] error: %s\n", *attachedPolicy.PolicyName, err.Error())
			os.Exit(1)
		}
	}

	// Delete Role
	_, err = iamClient.DeleteRole(&iam.DeleteRoleInput{RoleName: &roleName})
	if err != nil {
		fmt.Printf("iam:DeleteRole [%s] error: %s\n", roleName, err.Error())
		os.Exit(1)
	}
	fmt.Printf("[%s]   Deleted role '%s'\n", accountId, roleName)

	return
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
	roleName := flag.String("role", "", "Name of role to delete")
	flag.Parse()
	if *roleName == "" {
		fmt.Printf("Missing required role name input\n")
		os.Exit(1)
	}

	accounts := getOrganizationAccounts()

	// Corporate proxy can struggle with heavy concurrency; serialize execution
	for _, account := range accounts {
		deleteRole(*account.Id, *roleName)
	}
}