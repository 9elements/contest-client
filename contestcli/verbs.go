package contestcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/9elements/contest-client/pkg/client"
	githubAPI "github.com/9elements/contest-client/pkg/github"
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
	webhookData WebhookData) (map[int]client.RunData, error) {

	jobs := make(map[int]client.RunData, len(cd.Flags.FlagJobTemplate))

	// Iterate over all JobTemplates that are defined in the clientconfig.json
	for i := 0; i < len(cd.Flags.FlagJobTemplate); i++ {
		// Open the configfile
		filepath, _ := filepath.Abs("descriptors/")
		filepathtemplate := filepath + "/" + *cd.Flags.FlagJobTemplate[i]
		// Parse and decode the json/yaml file
		templateDescription, err := ioutil.ReadFile(filepathtemplate)
		if err != nil {
			log.Printf("could not parse the jobtemplate: %s\n", err)
		}

		// Retrieve the jobName for further usages
		jobName, err := RetrieveJobName(templateDescription, *cd.Flags.FlagYAML)
		if err != nil {
			fmt.Printf("could not retrieve the job name: %+v\n", err)
		}
		// Adapt the jobDescriptor based on the webhookdata
		jobDesc, err := ChangeJobDescriptor(templateDescription, webhookData)
		if err != nil {
			fmt.Printf("could not change the job template: %+v\n", err)
		}
		// If template file is YAML convert it to JSON
		if *cd.Flags.FlagYAML {
			// Unmarshal the data in a map
			var body interface{}
			if err := yaml.Unmarshal(jobDesc, &body); err != nil {
				fmt.Printf("failed to parse YAML job descriptor: %+v", err)
			}
			body = dyno.ConvertMapI2MapS(body)
			// then marshal the structure back to JSON
			jobDesc, err = json.MarshalIndent(body, "", "    ")
			if err != nil {
				fmt.Printf("failed to serialize job descriptor to JSON: %+v", err)
			}
		}

		// Updating the github status to pending
		res := githubAPI.EditGithubStatus(ctx, "pending", "",
			jobName+". Test-Result:", webhookData.headSHA)
		if res != nil {
			log.Printf("could not change the github status: %s\n", res)
		}

		// Kick off the generated Job
		startResp, err := transport.Start(context.Background(), *cd.Flags.FlagRequestor, string(jobDesc))
		if err != nil {
			fmt.Printf("could not send the Job with the jobDesc: %s\n", *cd.Flags.FlagJobTemplate[i])
			err := githubAPI.EditGithubStatus(ctx, "error", "", jobName+". Test-Result:", webhookData.headSHA)
			if err != nil {
				fmt.Println("GithubStatus could not be edited to status: error", err)
			}
			return jobs, err
		} else {
			if int(startResp.Data.JobID) == 0 {
				fmt.Printf("The Job could not executed. Server returned JobID 0! \n") //TODO: ERROR HANDLING
			}
		}

		// Filling the map with job data for postjobexecutionhooks
		jobdata := client.RunData{JobName: jobName, JobSHA: webhookData.headSHA}
		jobs[int(startResp.Data.JobID)] = jobdata

		// Create Json Body for API Request to set a status for the started Job
		data := map[string]interface{}{
			"ID":     startResp.Data.JobID,
			"Status": false,
		}
		json_data, err := json.Marshal(data) //Marshal that data
		if err != nil {
			fmt.Println("Could not parse data to json format.")
		}
		// Add the job to the Api DB
		// TODO: Add the Address to the config!
		addr := *cd.Flags.FlagAddr + *cd.Flags.FlagPortAPI + "/addjobstatus/"
		resp, err := http.Post(addr, "application/json", bytes.NewBuffer(json_data))

		if err != nil {
			fmt.Println("Could not post data to API.")
			return nil, err
		}
		if resp.StatusCode != 200 {
			fmt.Println("The HTTP Post responded a statuscode != 200")
			return nil, err
		}
	}
	return jobs, nil
}

// Parse the jobDescriptor and substitute all template with the webhook data
func ChangeJobDescriptor(data []byte, webhookData WebhookData) ([]byte, error) {
	// Create buffer to pass the adapted data
	var buf bytes.Buffer
	datastring := string(data)
	// Create the data that should be substitute
	jobDescData := templatedata{webhookData.headSHA}
	// Parse the file data
	tmpl, err := template.New("jobDesc").Delims("[[", "]]").Parse(datastring)
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
	var jobDesc map[string]interface{}
	// Check if the file is YAML or JSON and unmarshal it
	if !YAML {
		// Retrieve the data and decode it into the jobDesc map
		if err := json.Unmarshal(data, &jobDesc); err != nil {
			return "", fmt.Errorf("failed to parse JSON job descriptor: %w", err)
		}
		// Retrieve the jobName from the JSON file
		jobName := jobDesc["JobName"]
		// Return the jobName
		return jobName.(string), nil
	} else {
		// Retrieve the data and decode it into the jobDesc map
		if err := yaml.Unmarshal(data, &jobDesc); err != nil {
			return "", fmt.Errorf("failed to parse YAML job descriptor: %w", err)
		}
		// Retrieve the jobName from the YAML file
		jobName := jobDesc["JobName"]
		// Return the jobName
		return jobName.(string), nil
	}
}
