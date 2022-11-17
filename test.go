package test

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/azure"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestTerraformAzureStorageExample(t *testing.T) {
	t.Parallel()

	// subscriptionID is overridden by the environment variable "ARM_SUBSCRIPTION_ID"
	subscriptionID := "a3d772dc-93d3-4b00-a52e-e8c86dfb1feb"

	// website::tag::1:: Configure Terraform setting up a path to Terraform code.
	terraformOptions := &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: "../template",

		// Variables to pass to our Terraform code using -var options

	}

	// website::tag::4:: At the end of the test, run `terraform destroy` to clean up any resources that were created
	defer terraform.Destroy(t, terraformOptions)

	// website::tag::2:: Run `terraform init` and `terraform apply`. Fail the test if there are any errors.
	terraform.InitAndApply(t, terraformOptions)

	// website::tag::3:: Run `terraform output` to get the values of output variables
	resourceGroupName := terraform.Output(t, terraformOptions, "resource_group_name")
	storageAccountName := terraform.Output(t, terraformOptions, "storage_account_name")

	// website::tag::4:: Verify storage account properties and ensure it matches the output.
	storageAccountExists := azure.StorageAccountExists(t, storageAccountName, resourceGroupName, subscriptionID)
	assert.True(t, storageAccountExists, "storage account does not exist")
	fmt.Println("================================================================")
	fmt.Println("================================================================")

}
