package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/9elements/contest-client/pkg/clientapi"
	"github.com/9elements/contest-client/pkg/webhook"
	"github.com/facebookincubator/contest/pkg/transport"
	"github.com/icza/dyno"
	"gopkg.in/yaml.v2"
)

// Struct that contains all possible template parameters
type templatedata struct {
	SHA string
}

/* Function run runs the main functionility of the contest-client.
   It creates new jobDescriptors and kicks off new jobs.
   It also sets the github commit status to pending if the job was started */
func run(ctx context.Context, cd client.ClientDescriptor, transport transport.Transport, stdout io.Writer,
	webhookData webhook.WebhookData) ([]client.RunData, error) {

	// Declare a jobs []struct that contains the rundata that shall be passed
	var jobs []client.RunData

	// Iterate over all JobTemplates that are defined in the clientconfig.json
	for i := 0; i < len(cd.Flags.FlagJobTemplate); i++ {

		// Create Path to the jobTemplate
		filePath, _ := filepath.Abs("descriptors/")
		filePathTemplate := strings.Join([]string{filePath, *cd.Flags.FlagJobTemplate[i]}, "/")

		// Parse the json/yaml file
		templateDescription, err := os.ReadFile(filePathTemplate)
		if err != nil {
			return nil, fmt.Errorf("could not parse the jobtemplate: %w", err)
		}

		// Retrieve the jobName for further usages
		jobName, err := RetrieveJobName(templateDescription, *cd.Flags.FlagYAML)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve the job name: %w", err)
		}

		// Adapt the jobDescriptor based on the webhookdata
		jobDesc, err := ChangeJobDescriptor(templateDescription, webhookData)
		if err != nil {
			return nil, fmt.Errorf("could not change the job template: %w", err)
		}

		// If template file is YAML convert it to JSON
		if *cd.Flags.FlagYAML {
			// Unmarshal the data in a map
			var body interface{}
			err := yaml.Unmarshal(jobDesc, &body)
			if err != nil {
				return nil, fmt.Errorf("failed to parse YAML job descriptor: %w", err)
			}
			body = dyno.ConvertMapI2MapS(body)
			// then marshal the structure back to JSON
			jobDesc, err = json.MarshalIndent(body, "", "    ")
			if err != nil {
				return nil, fmt.Errorf("failed to serialize job descriptor to JSON: %w", err)
			}
		}

		// Kick off the generated Job
		startResp, err := transport.Start(context.Background(), *cd.Flags.FlagRequestor, string(jobDesc))
		// If the server is not reachable
		if err != nil {
			return nil, fmt.Errorf("could not send the Job to the server: %w", err)

			// If the server is reachable but something else went wrong
		} else {
			// If the job could not executed
			if int(startResp.Data.JobID) == 0 {
				return nil, fmt.Errorf("the Job could not executed. Server returned JobID 0")
			}
		}

		// Updating the github status to pending after the job is kicked off
		Github := clientapi.GithubAPI{}
		err = Github.EditGithubStatus(ctx, "pending", "http://www.urltotestreport.de/", jobName+". Test-Report:", webhookData.HeadSHA)
		if err != nil {
			return nil, fmt.Errorf("could not change the github status: %w", err)
		}

		// Filling the map with job data for postjobexecutionhooks
		jobData := client.RunData{JobID: int(startResp.Data.JobID), JobName: jobName, JobSHA: webhookData.HeadSHA}
		jobs = append(jobs, jobData)

		// Create Json Body for API Request to set a status for the started Job
		data := map[string]interface{}{
			"ID":     startResp.Data.JobID,
			"Status": false,
		}

		// Marshal that data
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("could not parse data to json format: %w", err)
		}

		// Add the job to the Api DB
		addr := strings.Join([]string{*cd.Flags.FlagAddr, *cd.Flags.FlagPortAPI, "/addjobstatus/"}, "")
		resp, err := http.Post(addr, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("could not post data to API: %w", err)
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("the HTTP Post responded a statuscode != 200 %w", err)
		}
	}
	return jobs, nil
}

// Parse the jobDescriptor and substitute all template with the webhook data
func ChangeJobDescriptor(data []byte, webhookData webhook.WebhookData) ([]byte, error) {
	// Create buffer to pass the adapted data
	var buf bytes.Buffer

	// Convert data to a string that could be parsed
	dataString := string(data)
	// Create the data that should be substitute
	jobDescData := templatedata{webhookData.HeadSHA}
	// Parse the file data
	tmpl, err := template.New("jobDesc").Delims("[[", "]]").Parse(dataString)
	if err != nil {
		return buf.Bytes(), fmt.Errorf("parse the data for templates: %t", err)
	}
	// Substitute all templates with jobDescData and write it to buf
	err = tmpl.Execute(&buf, jobDescData)
	if err != nil {
		return buf.Bytes(), fmt.Errorf("template substitution failed: %t", err)
	}
	// Return buf as byte array
	return buf.Bytes(), nil
}

// Unmarshal the data from the template file and retrieve the jobName
func RetrieveJobName(data []byte, YAML bool) (string, error) {
	// String interface{} map to unmarshal the incoming data
	var jobDesc map[string]interface{}

	// Check if the file is YAML or JSON and unmarshal it
	if !YAML {
		// Retrieve the data and decode it into the jobDesc map
		if err := json.Unmarshal(data, &jobDesc); err != nil {
			return "", fmt.Errorf("failed to parse JSON job descriptor: %w", err)
		}
	} else {
		// Retrieve the data and decode it into the jobDesc map
		if err := yaml.Unmarshal(data, &jobDesc); err != nil {
			return "", fmt.Errorf("failed to parse YAML job descriptor: %w", err)
		}
	}

	// Retrieve the jobName from the JSON file
	jobName := jobDesc["JobName"]
	// Return the jobName
	return jobName.(string), nil
}
