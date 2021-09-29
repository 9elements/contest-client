package pushtoS3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/9elements/contest-client/pkg/clientapi"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/facebookincubator/contest/pkg/api"
	"github.com/facebookincubator/contest/pkg/job"
	"github.com/facebookincubator/contest/pkg/transport"
	"github.com/facebookincubator/contest/pkg/types"
)

// Function that upload the job report into a S3 bucket
func PushReportsToS3(ctx context.Context, cd client.ClientDescriptor,
	transport transport.Transport, parameter PushToS3, runData client.RunData) error {

	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := CreateAwsSession(parameter)
	if err != nil {
		return err
	}

	// Creating link to read out the status of the running job from an api
	readJobStatus := strings.Join([]string{*cd.Flags.FlagAddr, *cd.Flags.FlagPortAPI, "/readjobstatus/", fmt.Sprint(runData.JobID)}, "")

	// Loop til the job report is finished and uploaded
	for {
		// Check if the job finished
		finished, err := CheckJobStatus(readJobStatus)
		if err != nil {
			return err
		}
		// finished contains the job status (true = job finished, false = job still running)
		// If the job is not already finished
		if !finished {
			// Sleep for the time thats configured in the clientconfig.json and than continue
			time.Sleep(time.Duration(*cd.Flags.FlagjobWaitPoll) * time.Second)
			continue
		}
		// If the job is finished
		// Retrieve the response of the report and the URL of the uploaded report.
		respBodyBytes, statusResp, err := RetrieveJobReport(transport, cd, parameter, runData.JobID)
		if err != nil {
			return err
		}

		// Invoke function that uploads the test report to a S3 Bucket
		// The function returns the name of the file that was uploaded to use it for the reportURL
		reportURL, err := AddFileToS3(s, parameter, respBodyBytes.Bytes(), runData.JobID)
		if err != nil {
			return fmt.Errorf("could upload the jobReport to the S3 bucket: %w", err)
		}

		// Check if the job succeded
		jobSuccess := CheckJobSuccess(statusResp.Data.Status.JobReport.RunReports)

		// Adapt the Github Commit statuses and Slack Msg depending on the job success
		// Creating the description for the report status and for the binary status
		reportDesc := runData.JobName + ". Test-Report:"
		binaryDesc := runData.JobName + ". Binary in S3 Bucket:"

		// Parse the status resp for the binary url to update the binary status
		// TODO: Find a way to differentiate multiple uploads in the report
		regex := "https://" + parameter.S3Bucket + `[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`
		r, _ := regexp.Compile(regex)
		binaryURL := r.FindString(respBodyBytes.String())
		if binaryURL != "" {
			// Update the Github status
			err = UpdateGithubStatus(ctx, jobSuccess, binaryURL, binaryDesc, runData)
			if err != nil {
				return err
			}
		}

		// Update the Github status
		err = UpdateGithubStatus(ctx, jobSuccess, reportURL, reportDesc, runData)
		if err != nil {
			return err
		}
		err = SendSlackMsg(jobSuccess, runData)
		if err != nil {
			return err
		}

		return nil
	}
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, parameter PushToS3, response []byte, jobID int) (string, error) {

	// Creating an upload path where the file should be uploaded with a timestamp
	currentTime := time.Now()
	uploadPath := strings.Join([]string{parameter.S3Path, currentTime.Format("20060102_150405")}, "/")
	uploadPath = strings.Join([]string{uploadPath, fmt.Sprintf("%v", jobID)}, "_")
	uploadPath = strings.Join([]string{uploadPath, "json"}, ".")

	// Uploading the file
	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(parameter.S3Bucket),
		Key:                  aws.String(uploadPath),
		ACL:                  aws.String("public-read"),
		Body:                 bytes.NewReader(response),
		ContentLength:        aws.Int64(int64(len(response))),
		ContentType:          aws.String(http.DetectContentType(response)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	if err != nil {
		return uploadPath, err
	}
	// Creating link where the job report can be downloaded.
	// This link will be put into the commit message right after the test status
	reportURL := strings.Join([]string{"https://", parameter.S3Bucket, ".s3.", parameter.S3Region, ".amazonaws.com/", uploadPath}, "")
	// Return the upload path
	return reportURL, nil
}

// CreateAwsSession creates an AWS Session and returns it to reuse it
func CreateAwsSession(parameter PushToS3) (*session.Session, error) {
	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String(parameter.S3Region),
		Credentials: credentials.NewSharedCredentials(
			parameter.AwsFile,    // AwsFile name
			parameter.AwsProfile, // AwsProfile name
		)})
	if err != nil {
		return nil, fmt.Errorf("starting an aws session failed: %w", err)
	}
	return s, nil
}

// CheckJobStatus sends request to an API that returns if a job finished or not
func CheckJobStatus(readJobStatus string) (bool, error) {
	// API request
	resp, err := http.Get(readJobStatus)
	if err != nil {
		return false, fmt.Errorf("could not post data to API: %w", err)
	}
	defer resp.Body.Close()

	// If API request was not 200 (StatusOK)
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("the status code of the respone is != 200 (StatusOk)")
	}

	// Unmarshal the status of the job that was requested
	var finished bool
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &finished)
	if err != nil {
		return false, fmt.Errorf("error reading the HTTP response: %w", err)
	}
	// Return status
	return finished, nil
}

// RetrieveJobReport retrieves the job report and returns it in 2 datatypes to use it in further proceeding
func RetrieveJobReport(transport transport.Transport, cd client.ClientDescriptor, parameter PushToS3, jobID int) (
	*bytes.Buffer, *api.StatusResponse, error) {
	// Retrieve the jobReport
	statusResp, err := transport.Status(context.Background(), *cd.Flags.FlagRequestor, types.JobID(jobID))
	if err != nil {
		return nil, nil, fmt.Errorf("could not retrieve the jobReport from the server: %w", err)
	}
	// Decoding
	respBodyBytes := new(bytes.Buffer)
	json.NewEncoder(respBodyBytes).Encode(statusResp)
	// Return data as bytes.Buffer and api.StatusResponse
	return respBodyBytes, statusResp, nil
}

// CheckJobSuccess parses the job report if the job was successful or not
func CheckJobSuccess(jobStatus [][]*job.Report) bool {
	// Go through all final reports
	jobSuccess := true
	for _, allReports := range jobStatus {
		//Go through all report in the final reports
		for _, reports := range allReports {
			var success = reports.Success
			if !success {
				jobSuccess = false
			}
		}
	}
	return jobSuccess
}

// UpdateGithubStatus updates different Github statuses depending on the success of the job
func UpdateGithubStatus(ctx context.Context, jobSuccess bool, dataURL string, statusDesc string,
	runData client.RunData) error {

	var Github = clientapi.GithubAPI{}

	// If the job was successful
	if !jobSuccess {
		// Update the github status
		err := Github.EditGithubStatus(ctx, "error", dataURL, statusDesc, runData.JobSHA)
		if err != nil {
			return fmt.Errorf("githubStatus could not be edited to status 'error': %w", err)
		}

		// If the job errors
	} else {
		// Update the github status
		err := Github.EditGithubStatus(ctx, "success", dataURL, statusDesc, runData.JobSHA)
		if err != nil {
			return fmt.Errorf("githubStatus could not be edited to status 'success': %w", err)
		}
	}
	return nil
}

// SendSlackMsg sends a msg to slack depending on the success of the job
func SendSlackMsg(jobSuccess bool, runData client.RunData) error {

	var Slack = clientapi.SlackAPI{}

	// If the job was successful
	if !jobSuccess {
		// Create a slack msg and than post it
		msg := strings.Join([]string{"Something goes wrong in the test with the jobName: '", runData.JobName, "'. Commit: '", runData.JobSHA, "'."}, "")
		err := Slack.MsgToSlack(msg)
		if err != nil {
			return fmt.Errorf("error could not posted to slack: %w", err)
		}

		// If the job errors
	} else {
		// Create a slack msg and than post it
		msg := strings.Join([]string{"The test with the jobName '", runData.JobName, "' was successful. Commit: '", runData.JobSHA, "'."}, "")
		err := Slack.MsgToSlack(msg)
		if err != nil {
			return fmt.Errorf("success could not posted to slack: %w", err)
		}
	}
	return nil
}
