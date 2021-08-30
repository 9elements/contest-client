package pushtoS3

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/facebookincubator/contest/pkg/transport"
)

// Name defines the name of the preexecutionhook used within the plugin registry
var Name = "pushtoS3"

// PushToS3 uploads a file to a specific AWS S3 Bucket
type PushToS3 struct {
	S3Region   string // Defines the S3 server region
	S3Bucket   string // Defines the S3 bucket name
	S3Path     string // Defines the S3 bucket upload path
	AwsFile    string // Defines the AWS config file location
	AwsProfile string // Defines the AWS config file profile
}

// ValidateRunParameters validates the parameters for the run reporter
func (n *PushToS3) ValidateParameters(params []byte) (interface{}, error) {
	// Retrieve the parameter into PushToS3 struct
	var s3Param PushToS3
	err := json.Unmarshal(params, &s3Param)
	if err != nil {
		return nil, fmt.Errorf("PushToS3 could not unmarshal the parameter while validating them: %w", err)
	}

	// Validate the S3Region
	if s3Param.S3Region == "" {
		return nil, fmt.Errorf("S3Region cannot be empty: %w", err)
	}
	// Validate the S3Bucket
	if s3Param.S3Bucket == "" {
		return nil, fmt.Errorf("S3Bucket cannot be empty: %w", err)
	}
	// Validate the S3Path
	if s3Param.S3Path == "" {
		return nil, fmt.Errorf("S3Path cannot be empty: %w", err)
	}

	// Validate the AwsFile
	// If AwsFile was not set to default
	if s3Param.AwsFile != "" {
		err = validateAWS(s3Param.AwsFile, s3Param.AwsProfile)
		if err != nil {
			return nil, err
		}
		// If AwsFile was set to default
	} else {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("could not retrieve the current usr: %w", err)
		}
		homeDir := usr.HomeDir + "/.aws/credentials"
		err = validateAWS(homeDir, s3Param.AwsProfile)
		if err != nil {
			return nil, err
		}
	}
	return s3Param, nil
}

func validateAWS(file string, AwsProfile string) error {
	// Open the AwsFile and parse it as string
	_, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("AwsFile does not exist: %w", err)
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("could not read the AwsFile: %w", err)
	}
	fileString := string(data)

	// Check if the given AwsProfile exists or not
	if AwsProfile != "" {
		exists := strings.Contains(fileString, AwsProfile)
		if !exists {
			return fmt.Errorf("AwsProfile does not exist")
		}
	} else {
		exists := strings.Contains(fileString, "default")
		if !exists {
			return fmt.Errorf("default AwsProfile does not exist")
		}
	}
	return nil
}

// Name returns the Name of the reporter
func (n *PushToS3) Name() string {
	return Name
}

// Run invokes PushResultsToS3 for each job, which uploads the job result
func (n *PushToS3) Run(ctx context.Context, parameter interface{}, cd client.ClientDescriptor, transport transport.Transport,
	rundata map[int]client.RunData) (interface{}, error) {

	// Retrieving the parameter
	var s3Param PushToS3 = parameter.(PushToS3)

	// Iterate over the different jobs
	for jobID, jobData := range rundata {
		// Retrieve the name of the job
		jobName := jobData.JobName
		// Retrieve the SHA of the commit
		jobSHA := jobData.JobSHA

		// Start the main logic of the plugin
		err := PushResultsToS3(ctx, cd, transport, s3Param, jobName, jobSHA, jobID)
		if err != nil {
			return nil, fmt.Errorf("PushResultToS3 in job %d did not finished: %w", jobID, err)
		}
	}
	return nil, nil
}

// New builds a new TargetSuccessReporter
func New() client.PostJobExecutionHooks {
	return &PushToS3{}
}

// Load returns the name and factory which are needed to register the Reporter
func Load() (string, client.PostJobExecutionHooksFactory) {
	return Name, New
}
