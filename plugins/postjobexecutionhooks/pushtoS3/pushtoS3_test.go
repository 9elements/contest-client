package pushtoS3

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/facebookincubator/contest/pkg/api"
)

// Define the parameter that are also retrieved from clientconfig.json
var s3Param = PushToS3{
	S3Region:   "eu-central-1",
	S3Bucket:   "coreboot-spr-sp-images",
	S3Path:     "test_results",
	AwsFile:    "",
	AwsProfile: "",
}

type reportSuccess struct {
	file    string
	success bool
}

// Test for ValidateParameters, if the validation works correctly
func TestValidateParameters(t *testing.T) {
	// Retrive important data
	cd := Helper(t)
	// Iterate over all PostJobExecutionHooks and validate that the return of the function equals the parameter defined above
	for _, eh := range cd.PostJobExecutionHooks {
		t.Run(eh.Name, func(t *testing.T) {
			exp, err := s3Param.ValidateParameters(eh.Parameters)
			if err != nil {
				t.Errorf("function 'ValidateParameters' returned an error: %w", err)
			}
			// Define want and got and compare them
			want := s3Param
			got := exp.(PushToS3)
			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}

// Test the PushReportsToS3 function with the main logic (invokes different subfunctions)
func TestPushReportsToS3(t *testing.T) {
	// Test CheckJobSuccess
	// Create reports with different content
	trueReport := reportSuccess{file: "testdata/true.json", success: true}
	falseReport := reportSuccess{file: "testdata/false.json", success: false}
	var reports []reportSuccess
	reports = append(reports, trueReport, falseReport)

	// Iterate over reports and invoke CheckJobSuccess
	for _, report := range reports {
		t.Run(report.file, func(t *testing.T) {
			// api.StatusResponse type to Unmarshal the job report
			var resp *api.StatusResponse
			// Read out the jsonFile
			byteValue, _ := ioutil.ReadFile(report.file)
			err := json.Unmarshal(byteValue, &resp)
			if err != nil {
				t.Errorf("could not unmarshal the report %w", err)
			}
			// Invoke CheckJobSuccess with the jobReport data
			got := CheckJobSuccess(resp.Data.Status.JobReport.RunReports)
			want := report.success
			if got != want {
				t.Errorf("got %t want %t", got, want)
			}
		})
	}

	// Test another subfunction...
	// ...
}

// Helper function to create data that is needed to pass it to functions
func Helper(t *testing.T) client.ClientDescriptor {
	// Open clientconfig.json
	configFile, err := os.Open("../../../clientconfig.json")
	if err != nil {
		t.Errorf("could not open the clientconfig: %w", err)
	}
	defer configFile.Close()

	// Unmarshal the configfile, to pass the data to the validation function
	configDescription, _ := ioutil.ReadAll(configFile)
	var cd client.ClientDescriptor
	if err := json.Unmarshal(configDescription, &cd); err != nil {
		t.Errorf("unable to decode the config file with err: %w", err)
	}
	// Return ClientDescriptor data
	return cd
}
