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
	webhookData WebhookData) (map[int][2]string, error) {

	jobs := make(map[int][2]string, len(cd.Flags.FlagJobTemplate))

	// Iterate over all JobTemplates that are defined in the clientconfig.json
	for i := 0; i < len(cd.Flags.FlagJobTemplate); i++ {
		//Open the configfile
		filepath, _ := filepath.Abs("descriptors/")
		filepathtemplate := filepath + "/" + *cd.Flags.FlagJobTemplate[i]
		//parse and decode the json/yaml file
		templateDescription, err := ioutil.ReadFile(filepathtemplate)
		if err != nil {
			log.Printf("could not parse the jobtemplate: %s\n", err)
		}

		jobName, err := RetrieveJobName(templateDescription, *cd.Flags.FlagYAML)

		// Adapt the jobDescriptor based on the webhookdata
		jobDesc, errorr := ChangeJobDescriptor(templateDescription, *cd.Flags.FlagYAML, webhookData)
		if errorr != nil {
			fmt.Printf("could not change the job template: %+v\n", err)
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
		var jobNameSha [2]string
		jobNameSha[0] = jobName
		jobNameSha[1] = webhookData.headSHA
		jobs[int(startResp.Data.JobID)] = jobNameSha

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
		resp, err := http.Post("http://10.93.193.82:3005/addjobstatus/", "application/json", bytes.NewBuffer(json_data))

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
func ChangeJobDescriptor(data []byte, YAML bool, webhookData WebhookData) ([]byte, error) {
	var buf bytes.Buffer
	datastring := string(data)
	jobDescData := templatedata{webhookData.headSHA}
	tmpl, err := template.New("jobDesc").Delims("[[", "]]").Parse(datastring)
	if err != nil {
		return buf.Bytes(), fmt.Errorf("Parse the data for templates: %t", err)
	}
	err = tmpl.Execute(&buf, jobDescData)
	if err != nil {
		return buf.Bytes(), fmt.Errorf("Template substitution failed: %t", err)
	}
	return buf.Bytes(), nil
}

// Unmarshal the data from the template file and retrieve the jobName
func RetrieveJobName(data []byte, YAML bool) (string, error) {
	var jobDesc map[string]string
	// Check if the file is YAML or JSON and unmarshal it
	if !YAML {
		// Retrieve the data and decode it into the jobDesc map
		if err := json.Unmarshal(data, &jobDesc); err != nil {
			return "", nil, fmt.Errorf("failed to parse JSON job descriptor: %w", err)
		}
		// Retrieve the jobName from the JSON file
		jobName := jobDesc["JobName"]
		// Return the jobName
		return jobName, nil
	} else {
		// Retrieve the data and decode it into the jobDesc map
		if err := yaml.Unmarshal(data, &jobDesc); err != nil {
			return "", nil, fmt.Errorf("failed to parse YAML job descriptor: %w", err)
		}
		// Retrieve the jobName from the YAML file
		jobName := jobDesc["JobName"]
		// Return the jobName
		return jobName.(string), nil
	}
}
