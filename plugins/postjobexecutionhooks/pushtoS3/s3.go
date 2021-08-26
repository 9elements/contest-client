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
	githubAPI "github.com/9elements/contest-client/pkg/github"
	slackAPI "github.com/9elements/contest-client/pkg/slack"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/facebookincubator/contest/pkg/job"
	"github.com/facebookincubator/contest/pkg/transport"
	"github.com/facebookincubator/contest/pkg/types"
)

// Function that upload the job result into a S3 bucket
func PushResultsToS3(ctx context.Context, cd client.ClientDescriptor,
	transport transport.Transport, parameter PushtoS3, jobName string, jobSha string, jobID int) error {

	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String(parameter.s3Region),
		Credentials: credentials.NewSharedCredentials(
			parameter.awsFile,    // awsfile name
			parameter.awsProfile, // awsprofile name
		)})
	if err != nil {
		return fmt.Errorf("starting an aws session failed: %w", err)
	}

	// Creating link to read out the status of the running job from an api
	readJobStatus := strings.Join([]string{*cd.Flags.FlagAddr, *cd.Flags.FlagPortAPI, "/readjobstatus/", fmt.Sprint(jobID)}, "")

	// create a job.Report variable to store the job reports in further proceeding
	var jobStatus [][]*job.Report

	// Loop til the job report is finished and uploaded
	for {
		// API request
		resp, err := http.Get(readJobStatus)
		if err != nil {
			return fmt.Errorf("could not post data to API: %w", err)
		}
		defer resp.Body.Close()

		// If API request was not 200 (StatusOK)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("the status code of the respone is != 200 (StatusOk)")
		}

		// Unmarshal the Status of the job that was requested
		var Status bool
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		json.Unmarshal(bodyBytes, &Status)
		if err != nil {
			return fmt.Errorf("error reading the HTTP response: %w", err)
		}
		// Status contains the job status (true = job finished, false = job still running)
		// If the job is not already finished
		if !Status {
			// Sleep for the time thats configured in the clientconfig.json and than continue
			time.Sleep(time.Duration(*cd.Flags.FlagjobWaitPoll) * time.Second)
			continue
		}
		// If the job is finished
		// Than retrieve the jobReport
		statusResp, err := transport.Status(context.Background(), *cd.Flags.FlagRequestor, types.JobID(jobID))
		if err != nil {
			return fmt.Errorf("could not retrieve the jobReport from the server: %w", err)
		}
		// Decoding
		respBodyBytes := new(bytes.Buffer)
		json.NewEncoder(respBodyBytes).Encode(statusResp)

		// Invoke function that uploads the test report to a S3 Bucket
		// The function returns the name of the file that was uploaded to use it for the resultURL
		uploadPath, err := AddFileToS3(s, parameter.s3Path, parameter.s3Bucket, respBodyBytes.Bytes(), jobID)
		if err != nil {
			return fmt.Errorf("could upload the jobReport to the S3 bucket: %w", err)
		}

		// Creating link where the job report can be downloaded.
		// This link will be put into the commit message right after the test status
		resultURL := strings.Join([]string{"https://", parameter.s3Bucket, ".s3.", parameter.s3Region, ".amazonaws.com/", uploadPath}, "")

		// Go through the report and retrieve the job status
		jobStatus = statusResp.Data.Status.JobReport.RunReports

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
		// Adapt the Github Commit statuses and Slack Msg depending on the job success
		// Creating the description for the result status and for the binary status
		resultDesc := jobName + ". Test-Result:"
		binaryDesc := jobName + ". Binary in S3 Bucket:"
		// Parse the status resp for the binary url to update the binary status
		regex := "https://" + parameter.s3Bucket + `[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`
		r, _ := regexp.Compile(regex)
		binaryURL := r.FindString(respBodyBytes.String())

		// If the job was successful
		if jobSuccess {
			// Update the binary status
			err := githubAPI.EditGithubStatus(ctx, "error", binaryURL, binaryDesc, jobSha)
			if err != nil {
				return fmt.Errorf("githubStatus could not be edited to status 'error': %w", err)
			}
			// Update the result status
			err = githubAPI.EditGithubStatus(ctx, "error", resultURL, resultDesc, jobSha)
			if err != nil {
				return fmt.Errorf("githubStatus could not be edited to status 'error': %w", err)
			}
			// Create a slack msg and than post it
			msg := strings.Join([]string{"Something goes wrong in the test with the jobName: '", jobName, "'. Commit: '", jobSha, "'."}, "")
			err = slackAPI.MsgToSlack(msg)
			if err != nil {
				return fmt.Errorf("error could not posted to slack: %w", err)
			}
			// If the job errors
		} else {
			// Update the binary status
			err := githubAPI.EditGithubStatus(ctx, "success", binaryURL, binaryDesc, jobSha)
			if err != nil {
				return fmt.Errorf("githubStatus could not be edited to status 'success': %w", err)
			}
			// Update the result status
			err = githubAPI.EditGithubStatus(ctx, "success", resultURL, resultDesc, jobSha)
			if err != nil {
				return fmt.Errorf("githubStatus could not be edited to status 'success': %w", err)
			}
			// Create a slack msg and than post it
			msg := strings.Join([]string{"The test with the jobName '", jobName, "' was successful. Commit: '", jobSha, "'."}, "")
			err = slackAPI.MsgToSlack(msg)
			if err != nil {
				return fmt.Errorf("success could not posted to slack: %w", err)
			}
		}
		return nil
	}
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, path string, bucket string, response []byte, jobID int) (string, error) {

	// Creating an upload path where the file should be uploaded with a timestamp
	currentTime := time.Now()
	uploadPath := strings.Join([]string{path, currentTime.Format("20060102_150405")}, "/")
	uploadPath = strings.Join([]string{uploadPath, fmt.Sprintf("%v", jobID)}, "_")

	// Uploading the file
	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(bucket),
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

	// Return the upload path
	return uploadPath, nil
}
