package pushtoS3

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/facebookincubator/contest/pkg/transport"
)

// Name defines the name of the preexecutionhook used within the plugin registry
var Name = "pushtoS3"

// PushtoS3 uploads a file to a specific AWS S3 Bucket
type PushtoS3 struct {
	s3Region   string // Defines the S3 server region
	s3Bucket   string // Defines the S3 bucket name
	s3Path     string // Defines the S3 bucket upload path
	awsFile    string // Defines the AWS config file location
	awsProfile string // Defines the AWS config file profile
}

// ValidateRunParameters validates the parameters for the run reporter
func (n *PushtoS3) ValidateParameters(params []byte) (interface{}, error) {
	// Retrieve the parameter into PushtoS3 struct
	var s3Param PushtoS3
	err := json.Unmarshal(params, &s3Param)
	if err != nil {
		return nil, fmt.Errorf("pushToS3 could not unmarshal the parameter while validating them: %w", err)
	}

	// Validate the s3Region
	if s3Param.s3Region == "" {
		return nil, fmt.Errorf("s3Region cannot be empty: %w", err)
	}
	// Validate the s3Bucket
	if s3Param.s3Bucket == "" {
		return nil, fmt.Errorf("s3Bucket cannot be empty: %w", err)
	}
	// Validate the s3Path
	if s3Param.s3Path == "" {
		return nil, fmt.Errorf("s3Path cannot be empty: %w", err)
	}

	// Validate the awsFile
	// If awsFile was not set to default
	if s3Param.awsFile != "" {
		err = validateAWS(s3Param.awsFile, s3Param.awsProfile)
		if err != nil {
			return nil, err
		}
		// If awsFile was set to default
	} else if s3Param.awsFile == "" {
		err = validateAWS("~/.aws/credentials", s3Param.awsProfile)
		if err != nil {
			return nil, err
		}
	}
	return s3Param, nil
}

func validateAWS(file string, awsProfile string) error {
	// Open the awsFile and parse it as string
	_, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("awsFile does not exist: %w", err)
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("could not read the awsFile: %w", err)
	}
	fileString := string(data)

	// Check if the given awsProfile exists or not
	if awsProfile != "" {
		exists := strings.Contains(fileString, awsProfile)
		if !exists {
			return fmt.Errorf("awsProfile does not exist!")
		}
	} else if awsProfile == "" {
		exists := strings.Contains(fileString, "default")
		if !exists {
			return fmt.Errorf("default awsProfile does not exist!")
		}
	}
	return nil
}

// Name returns the Name of the reporter
func (n *PushtoS3) Name() string {
	return Name
}

// Run invokes PushResultsToS3 for each job, which uploads the job result
func (n *PushtoS3) Run(ctx context.Context, parameter interface{}, cd client.ClientDescriptor, transport transport.Transport,
	rundata map[int]client.RunData) (interface{}, error) {

	// Retrieving the parameter
	var s3Param PushtoS3 = parameter.(PushtoS3)

	// Iterate over the different jobs
	for jobID, jobData := range rundata {
		// Retrieve the name of the job
		jobName := jobData.JobName
		// Retrieve the SHA of the commit
		jobSHA := jobData.JobSHA

		// Start the main logic of the plugin
		err := PushResultsToS3(ctx, cd, transport, s3Param, jobName, jobSHA, jobID)
		if err != nil {
			return nil, fmt.Errorf("PushResultToS3 did not finished: %w", err)
		}
	}
	return nil, nil
}

// New builds a new TargetSuccessReporter
func New() client.PostJobExecutionHooks {
	return &PushtoS3{}
}

// Load returns the name and factory which are needed to register the Reporter
func Load() (string, client.PostJobExecutionHooksFactory) {
	return Name, New
}
