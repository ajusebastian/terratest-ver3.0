package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
)

var planstruct terraform.PlanStruct

const (
	substr       = "runs/run-"
	planFileName = "CloudPlan.json"
)

// An example of how to test the Terraform module in examples/terraform-aws-example using Terratest.
func TestTerraformAzureExamplePlan(t *testing.T) {
	t.Parallel()
	//Read variable values from tfvars file for comparison
	expectedSAName := terraform.GetVariableAsStringFromVarFile(t, "../template/terraform.tfvars", "storageaccountname")
	expectedSAHttpSettings, _ := strconv.ParseBool(terraform.GetVariableAsStringFromVarFile(t, "../template/terraform.tfvars", "enable_https_traffic_only"))
	//Define Terraform Options
	expectedStorageKind := "StorageV2"
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: "../template",
		Vars: map[string]interface{}{
			"storageaccountname":        expectedSAName,
			"enable_https_traffic_only": expectedSAHttpSettings,
			"storage_kind":              expectedStorageKind,
		},
	})
	//Invoke Terraform Init and plan
	planTfc := terraform.InitAndPlan(t, terraformOptions)
	//Download terraform plan from TFC
	downloadTFCPlan(planTfc, substr, planFileName)
	planJson, _ := os.ReadFile(planFileName)
	//Parse the planJson and unmarshall
	plan := parseJsonPlan(string(planJson))
	//Unit Tests
	terraform.RequirePlannedValuesMapKeyExists(t, plan, "module.storage_account.azurerm_storage_account.aju-storageaccount")
	azureResource := planstruct.ResourcePlannedValuesMap["module.storage_account.azurerm_storage_account.aju-storageaccount"]
	azurestoreagename := azureResource.AttributeValues["name"]
	enable_https_traffic_only := azureResource.AttributeValues["enable_https_traffic_only"]
	storage_account_kind := azureResource.AttributeValues["account_kind"]
	//Unit test 1 : Storage Account Name Check
	assert.Equal(t, expectedSAName, azurestoreagename)
	//Unit test 2 : Storage Account HTTPS Settings Check
	assert.Equal(t, expectedSAHttpSettings, enable_https_traffic_only)
	//Unit test 3 : Storage Account Kind Check
	assert.Equal(t, expectedStorageKind, storage_account_kind)
	fmt.Println("===================================================")
	fmt.Println("Unit test 1")
	fmt.Println("Expected Storage Name: " + expectedSAName + " | Actual Storage Name: " + fmt.Sprintf("%v", azurestoreagename))
	fmt.Println("Unit test 2")
	fmt.Println("Expected Storage HTTP Settings: " + fmt.Sprintf("%v", expectedSAHttpSettings) + " | Actual Storage HTTP Settings: " + fmt.Sprintf("%v", enable_https_traffic_only))
	fmt.Println("Unit test 3")
	fmt.Println("Expected Storage Kind: " + expectedStorageKind + " | Actual Storage Kind: " + fmt.Sprintf("%v", storage_account_kind))
	fmt.Println("===================================================")
}

func parseJsonPlan(planJson string) *terraform.PlanStruct {
	plan := &planstruct
	json.Unmarshal([]byte(planJson), &plan.RawPlan)

	plan.ResourcePlannedValuesMap = parsePlannedValues(plan)
	plan.ResourceChangesMap = parseResourceChanges(plan)
	return plan
}

func downloadTFCPlan(planTfc, substr, planFileName string) {
	runId := getTFCRunId(planTfc, substr)
	url := "https://app.terraform.io/api/v2/runs/#runid#/plan/json-output"
	url = strings.Replace(url, "#runid#", runId, -1)
	token := "fzSb6O7WUAMAug.atlasv1.G2htBdBurrLaDNu1zIz8BttWJylfrLzF6P8Vyi2psBgm848tvOCf6LCFbliloIFcRx0"

	bearer := "Bearer " + token

	req, err := http.NewRequest("GET", url, bytes.NewBuffer(nil))
	req.Header.Set("Authorization", bearer)
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		for key, val := range via[0].Header {
			req.Header[key] = val
		}
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error on response.\n[ERRO] -", err)
	} else {
		defer resp.Body.Close()
		jsonPlan, _ := io.ReadAll(resp.Body)
		err := os.WriteFile(planFileName, []byte(string(jsonPlan)), 0666)
		if err != nil {

		}
	}
}
func getTFCRunId(planTfc, substr string) string {
	i := strings.Index(planTfc, substr)
	runId := planTfc[i+5 : i+5+20]

	return runId
}

func parsePlannedValues(plan *terraform.PlanStruct) map[string]*tfjson.StateResource {
	plannedValues := plan.RawPlan.PlannedValues
	if plannedValues == nil {
		// No planned values, so return empty map.
		return map[string]*tfjson.StateResource{}
	}

	rootModule := plannedValues.RootModule
	if rootModule == nil {
		// No module resources, so return empty map.
		return map[string]*tfjson.StateResource{}
	}
	return parseModulePlannedValues(rootModule)
}
func parseModulePlannedValues(module *tfjson.StateModule) map[string]*tfjson.StateResource {
	out := map[string]*tfjson.StateResource{}
	for _, resource := range module.Resources {
		// NOTE: the Address attribute of the module resource always returns the full address, even when the resource is
		// nested within sub modules.
		out[resource.Address] = resource
	}

	// NOTE: base case of recursion is when ChildModules is empty list.
	for _, child := range module.ChildModules {
		// Recurse in to the child module. We take a recursive approach here despite limitations of the recursion stack
		// in golang due to the fact that it is rare to have heavily deep module calls in Terraform. So we optimize for
		// code readability as opposed to performance.
		childMap := parseModulePlannedValues(child)
		for k, v := range childMap {
			out[k] = v
		}
	}
	return out
}
func parseResourceChanges(plan *terraform.PlanStruct) map[string]*tfjson.ResourceChange {
	out := map[string]*tfjson.ResourceChange{}
	for _, change := range plan.RawPlan.ResourceChanges {
		out[change.Address] = change
	}
	return out
}
