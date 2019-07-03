package main

import (
	"os"
	"fmt"
	"sort"
	"time"
	"encoding/csv"
	"github.com/aws/aws-sdk-go/aws"
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


func writeRoleLastAccessedDetails(writer csv.Writer, accountId string) {
	sess := session.Must(session.NewSession())
	role := fmt.Sprintf("arn:aws:iam::%s:role/CloudCoreRO", accountId)
	cred := stscreds.NewCredentials(sess, role)
	iamClient := iam.New(sess, &aws.Config{Credentials: cred})

	now := time.Now()
	var roles []*iam.Role
	err := iamClient.ListRolesPages(&iam.ListRolesInput{},
		func(page *iam.ListRolesOutput, lastPage bool) bool {
			for _, role := range page.Roles {
				roles = append(roles, role)
			}
			return true
		})
	if err != nil {
		fmt.Println("iam:ListRoles error:", err.Error())
		os.Exit(1)
	}

	fmt.Printf("[%s]: Processing IAM Roles...\n",accountId)
	for _, role := range roles {
		var jobId *string
		var details *iam.GetServiceLastAccessedDetailsOutput
		
		// GenerateServiceLastAccessDetails
		res, err := iamClient.GenerateServiceLastAccessedDetails(&iam.GenerateServiceLastAccessedDetailsInput{
			Arn: role.Arn,
			})
		if err != nil {
			fmt.Println("iam:GenerateServiceLastAccessedDetails error:", err.Error())
		}

		jobId = res.JobId
		for {
			time.Sleep(500 * time.Millisecond)

			details, err = iamClient.GetServiceLastAccessedDetails(&iam.GetServiceLastAccessedDetailsInput{
				JobId: jobId,
			})
			if err != nil {
				fmt.Println("iam:GetServiceLastAccessedDetails error:", err.Error())
			}

			if *details.JobStatus != "IN_PROGRESS" {
				break
			}
		}

		// Get most recent LastAuthenticated timestamp
		var serviceLastAccessed iam.ServiceLastAccessed
		for _, service := range details.ServicesLastAccessed {
			if service.LastAuthenticated != nil {
				if serviceLastAccessed.LastAuthenticated == nil {
					serviceLastAccessed = *service
				} else {
					if serviceLastAccessed.LastAuthenticated.Before(*service.LastAuthenticated) {
						serviceLastAccessed = *service
					}
				}
			}
		}

		// CSV record
		var record []string
		var lastAccessedDays float64 = -1
		var ageDays float64 = -1
		var lastAccessedService string = ""

		record = append(record, *role.RoleName)
		record = append(record, accountId)

		if role.CreateDate != nil {
			ageDays = now.Sub(*role.CreateDate).Hours() / 24
		}
		record = append(record, fmt.Sprintf("%0.0f", ageDays))		
		
		if serviceLastAccessed.LastAuthenticated != nil {
			lastAccessedDays = now.Sub(*serviceLastAccessed.LastAuthenticated).Hours() / 24
		}
		record = append(record, fmt.Sprintf("%0.0f", lastAccessedDays))

		if serviceLastAccessed.ServiceName != nil {
			lastAccessedService = *serviceLastAccessed.ServiceName
		}
		record = append(record, lastAccessedService)
		
		record = append(record, *role.Arn)
		
		if err := writer.Write(record); err != nil {
			fmt.Println("csv writer error", err.Error())
		}
	}

	writer.Flush()
	return
}

func writeUserLastAccessedDetails(writer csv.Writer, accountId string) {
	sess := session.Must(session.NewSession())
	role := fmt.Sprintf("arn:aws:iam::%s:role/CloudCoreRO", accountId)
	cred := stscreds.NewCredentials(sess, role)
	iamClient := iam.New(sess, &aws.Config{Credentials: cred})

	now := time.Now()
	var users []*iam.User
	err := iamClient.ListUsersPages(&iam.ListUsersInput{},
		func(page *iam.ListUsersOutput, lastPage bool) bool {
			for _, user := range page.Users {
				users = append(users, user)
			}
			return true
		})
	if err != nil {
		fmt.Println("iam:ListUsers error:", err.Error())
		os.Exit(1)
	}

	fmt.Printf("[%s]: Processing IAM Users...\n",accountId)
	for _, user := range users {
		var jobId *string
		var details *iam.GetServiceLastAccessedDetailsOutput

		// GenerateServiceLastAccessDetails
		res, err := iamClient.GenerateServiceLastAccessedDetails(&iam.GenerateServiceLastAccessedDetailsInput{
			Arn: user.Arn,
			})
		if err != nil {
			fmt.Println("iam:GenerateServiceLastAccessedDetails error:", err.Error())
		}

		jobId = res.JobId
		for {
			time.Sleep(500 * time.Millisecond)

			details, err = iamClient.GetServiceLastAccessedDetails(&iam.GetServiceLastAccessedDetailsInput{JobId: jobId})
			if err != nil {
				fmt.Println("iam:GetServiceLastAccessedDetails error:", err.Error())
			}

			if *details.JobStatus != "IN_PROGRESS" {
				break
			}
		}
		// Get most recent LastAuthenticated timestamp
		var serviceLastAccessed iam.ServiceLastAccessed
		for _, service := range details.ServicesLastAccessed {
			if service.LastAuthenticated != nil {
				if serviceLastAccessed.LastAuthenticated == nil {
					serviceLastAccessed = *service
				} else {
					if serviceLastAccessed.LastAuthenticated.Before(*service.LastAuthenticated) {
						serviceLastAccessed = *service
					}
				}
			}
		}

		// CSV record
		var record []string
		var lastAccessedDays float64 = -1
		var ageDays float64 = -1
		var lastAccessedService string = ""

		record = append(record, *user.UserName)
		record = append(record, accountId)

		if user.CreateDate != nil {
			ageDays = now.Sub(*user.CreateDate).Hours() / 24
		}
		record = append(record, fmt.Sprintf("%0.0f", ageDays))	
		
		if serviceLastAccessed.LastAuthenticated != nil {
			lastAccessedDays = now.Sub(*serviceLastAccessed.LastAuthenticated).Hours() / 24
		}
		record = append(record, fmt.Sprintf("%0.0f", lastAccessedDays))

		if serviceLastAccessed.ServiceName != nil {
			lastAccessedService = *serviceLastAccessed.ServiceName
		}
		record = append(record, lastAccessedService)
		
		record = append(record, *user.Arn)
		
		if err := writer.Write(record); err != nil {
			fmt.Println("csv writer error", err.Error())
		}
	}

	writer.Flush()
	return
}

func main() {
	accounts := getOrganizationAccounts()
	currentTime := time.Now()
	headers := []string{"Name","Account","AgeDays","LastAccessedDays","LastAccessService","Arn"}

	roleFile, err := os.Create(fmt.Sprintf("role-last-access-details-%s.csv", currentTime.Format("20060102")))
	if err != nil {
		fmt.Println("Failed to create role file:", err)
		os.Exit(1)
	}
	defer roleFile.Close()
	roleWriter := csv.NewWriter(roleFile)
	if err := roleWriter.Write(headers); err != nil {
		fmt.Println("csv writer error", err.Error())
	}

	userFile, err := os.Create(fmt.Sprintf("user-last-access-details-%s.csv", currentTime.Format("20060102")))
	if err != nil {
		fmt.Println("Failed to create user file:", err)
		os.Exit(1)
	}
	defer userFile.Close()
	userWriter := csv.NewWriter(userFile)
	if err := userWriter.Write(headers); err != nil {
		fmt.Println("csv writer error", err.Error())
	}

	// iam:GetLastAccessDetails() has an aggressive API rate limit; serialize execution
	for _, account := range accounts {
		writeRoleLastAccessedDetails(*roleWriter, *account.Id)
		writeUserLastAccessedDetails(*userWriter, *account.Id)
	}
}