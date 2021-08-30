package pushtoS3

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/9elements/contest-client/pkg/client"
)

// Test for ValidateParameters, if the validation works correctly
func TestValidateParameters(t *testing.T) {
	// Define the parameter that are also retrieved from clientconfig.json
	s3Param := PushToS3{
		S3Region:   "eu-central-1",
		S3Bucket:   "coreboot-spr-sp-images",
		S3Path:     "test_results",
		AwsFile:    "",
		AwsProfile: "",
	}
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
	// Iterate over all PostJobExecutionHooks and validate that the return of the function equals the parameter defined above
	for _, eh := range cd.PostJobExecutionHooks {
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
	}
}
