package pushtoS3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/9elements/contest-client/pkg/client"
	githubAPI "github.com/9elements/contest-client/pkg/client/github"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/facebookincubator/contest/pkg/job"
	"github.com/facebookincubator/contest/pkg/transport"
	"github.com/facebookincubator/contest/pkg/types"
)

const (
	S3_REGION = "eu-central-1"
	S3_BUCKET = "coreboot-spr-sp-images"
	S3_FOLDER = "test_results"
)

func PushResultsToS3(ctx context.Context, cd client.ClientDescriptor, transport transport.Transport, jobName string, jobSha string, jobID int) error {

	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String(S3_REGION), Credentials: credentials.NewSharedCredentials(
		"",           // filename
		"9e-AWS-Key", //profile
	)})
	if err != nil {
		return err
	}
	fmt.Println("S3 Bucket session established.")

	readjobstatus := "http://10.93.193.82:3005/readjobstatus/" + fmt.Sprint(jobID)

	var jobStatus [][]*job.Report

	for {
		resp, err := http.Get(readjobstatus)
		if err != nil {
			fmt.Println("Could not post data to API.")
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error reading the HTTP response", err)
			}
			bodyString := string(bodyBytes)

			if bodyString == "true" {
				resp, err := transport.Status(context.Background(), *cd.Flags.FlagRequestor, types.JobID(jobID))
				if err != nil {
					return err
				}
				respBodyBytes := new(bytes.Buffer)
				json.NewEncoder(respBodyBytes).Encode(resp)

				// Upload
				filename, err := AddFileToS3(s, respBodyBytes.Bytes(), jobID)
				if err != nil {
					fmt.Println(err)
					return err
				}
				fileurl := "https://coreboot-spr-sp-images.s3.eu-central-1.amazonaws.com/binaries/" + filename

				jobStatus = resp.Data.Status.JobReport.RunReports
				for _, finalreports := range jobStatus {
					var matcherr bool = false
					var matchsucceed bool = false
					for _, reports := range finalreports {
						var status = reports.Data
						switch statustype := status.(type) {
						case string:
							matcherror, _ := regexp.MatchString("does not pass", status.(string))
							if matcherror {
								if !matcherr {
									matcherr = matcherror
								}
								fmt.Printf("the test with JobID %d does not succeeded!\n", jobID)
							}
							matchsuccess, _ := regexp.MatchString("does pass", status.(string))
							if matchsuccess {
								if !matchsucceed {
									matchsucceed = matchsuccess
								}
								fmt.Printf("the test with JobID %d succeeded!\n", jobID)
							}
						case interface{}:
							for _, item := range status.([]interface{}) {
								matcherror, _ := regexp.MatchString("does not pass", item.(string))
								if matcherror {
									if !matcherr {
										matcherr = matcherror
									}
									fmt.Printf("the test with JobID %d does not succeeded!\n", jobID)
								}
								matchsuccess, _ := regexp.MatchString("does pass", item.(string))
								if matchsuccess {
									if !matchsucceed {
										matchsucceed = matchsuccess
									}
									fmt.Printf("the test with JobID %d succeeded!\n", jobID)

								}
							}
						default:
							fmt.Println(statustype)
							continue
						}
						if matcherr {
							err := githubAPI.EditGithubStatus(ctx, "error", fileurl, jobName, jobSha)
							if err != nil {
								fmt.Println("GithubStatus could not be edited to status: error", err)
							}
						} else if matchsucceed && !matcherr {
							err := githubAPI.EditGithubStatus(ctx, "success", fileurl, jobName, jobSha)
							if err != nil {
								fmt.Println("GithubStatus could not be edited to status: error", err)
							}
						}
					}
				}
				return nil
			}
		} else {
			fmt.Println("The HTTP Post responded a statuscode != 200")
		}
		// TODO use  time.Ticker instead of time.Sleep

		time.Sleep(time.Duration(*cd.Flags.FlagjobWaitPoll) * time.Second)
	}
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, response []byte, jobID int) (string, error) {

	buffer := response
	currentTime := time.Now()

	// Config settings: this is where you choose the bucket, filename, content-type etc.
	// of the file you're uploading.
	fileName := fmt.Sprintf("%s/%s_%d.json", S3_FOLDER, currentTime.Format("20060102_150405"), jobID)

	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(S3_BUCKET),
		Key:                  aws.String(fileName),
		ACL:                  aws.String("public-read"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(int64(len(buffer))),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	if err != nil {
		return "", err
	} else {
		fmt.Printf("Pushed the JobReport from JobID %d to S3 Bucket! \n", jobID)
	}
	return fileName, err
}
