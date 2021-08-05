package contestcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/9elements/contest-client/pkg/client"
	githubAPI "github.com/9elements/contest-client/pkg/github"
	"github.com/Navops/yaml"
	"github.com/facebookincubator/contest/pkg/transport"
)

/* Function run runs the main functionility of the contest-client.
   It creates new jobDescriptors and kicks off new jobs.
   It also sets the github commit status to pending if the job was started */
func run(ctx context.Context, cd client.ClientDescriptor, transport transport.Transport, stdout io.Writer,
	webhookData []string) (map[int][2]string, error) {

	jobs := make(map[int][2]string, len(cd.Flags.FlagJobTemplate))

	//iterate over all JobTemplates that are defined in the clientconfig.json
	for i := 0; i < len(cd.Flags.FlagJobTemplate); i++ {
		//Open the configfile
		filepath, _ := filepath.Abs("descriptors/")
		filepathtemplate := filepath + "/" + *cd.Flags.FlagJobTemplate[i]
		//parse and decode the json/yaml file
		templateDescription, err := ioutil.ReadFile(filepathtemplate)
		if err != nil {
			log.Printf("could not parse the jobtemplate: %s\n", err)
		}

		//Updating the github status to pending
		res := githubAPI.EditGithubStatus(ctx, "pending", "",
			*cd.Flags.FlagJobTemplate[i]+" test-result:", webhookData[0])
		if res != nil {
			log.Printf("could not change the github status: %s\n", res)
		}

		//adapt the jobDescriptor based on the webhookdata
		jobDesc, errorr := ChangeJobDescriptor(templateDescription, *cd.Flags.FlagYAML, webhookData)
		if errorr != nil {
			fmt.Printf("could not change the job template: %+v\n", err)
		}

		//kick off the generated Job
		startResp, err := transport.Start(context.Background(), *cd.Flags.FlagRequestor, string(jobDesc))
		if err != nil {
			fmt.Printf("could not send the Job with the jobDesc: %s\n", *cd.Flags.FlagJobTemplate[i])
			err := githubAPI.EditGithubStatus(ctx, "error", "", *cd.Flags.FlagJobTemplate[i]+" test-result:", webhookData[0])
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
		jobNameSha[0] = *cd.Flags.FlagJobTemplate[i]
		jobNameSha[1] = webhookData[0]
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
		// Add the job to the Api db
		resp, err := http.Post("http://10.93.193.82:3005/addjobstatus/", "application/json", bytes.NewBuffer(json_data)) //HTTP Post all to the API

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

//unmarshal the data from the template file then change the specific value and marshal it back to specific format
func ChangeJobDescriptor(data []byte, YAML bool, webhookData []string) ([]byte, error) {
	var jobDesc map[string]interface{}

	//check if the file is YAML or JSON and depending on it Unmarshal it, adapt it and marshal it again
	if !YAML {
		// Retrieve the data and decode it into the jobDesc map
		if err := json.Unmarshal(data, &jobDesc); err != nil {
			return nil, fmt.Errorf("failed to parse JSON job descriptor: %w", err)
		}
		// Diving into the YAML structure to get to the parameter we want to edit
		if testD, ok := jobDesc["TestDescriptors"].([]interface{}); !ok {
			return nil, fmt.Errorf("JSON File is not valid for this usecase %w", ok)
		} else if testF, ok := testD[0].(map[string]interface{})["TestFetcherFetchParameters"]; !ok {
			return nil, fmt.Errorf("JSON File is not valid for this usecase %w", ok)
		} else if steps, ok := testF.(map[string]interface{})["Steps"]; !ok {
			return nil, fmt.Errorf("JSON File is not valid for this usecase %w", ok)
		} else {
			switch val := steps.(type) {
			case []interface{}:
				for _, v := range val {
					switch val := v.(type) {
					case map[string]interface{}:
						for k, v := range val {
							// If the label of the test step we want to edit is correct, we will edit the arguements
							if k == "label" && v == "Checkout to the right commit" {
								switch val := val["parameters"].(type) {
								case map[string]interface{}:
									for k, v := range val {
										if k == "args" {
											args := v.([]interface{})
											args[1] = webhookData[0] // Edit the commit SHA
										}
									}
								default:
									fmt.Println("JSON File has wrong format")
								}
							}
						}
					default:
						fmt.Println("JSON File has wrong format")
					}
				}
			default:
				fmt.Println("JSON File has wrong format")
			}
		}
		// marshal the jobDescriptor back to JSON format
		jobDescJSON, err := json.MarshalIndent(jobDesc, "", "    ")
		if err != nil {
			return nil, fmt.Errorf("failed to serialize job descriptor to JSON: %w", err)
		}
		// Return the adapted jobDescriptor
		return jobDescJSON, nil

	} else {
		// Retrieve the data and decode it into the jobDesc map
		if err := yaml.Unmarshal(data, &jobDesc); err != nil {
			return nil, fmt.Errorf("failed to parse YAML job descriptor: %w", err)
		}
		// Diving into the JSON structure to get to the parameter we want to edit
		if testD, ok := jobDesc["TestDescriptors"].([]interface{}); !ok {
			return nil, fmt.Errorf("YAML File is not valid for this usecase %w", ok)
		} else if testF, ok := testD[0].(map[string]interface{})["TestFetcherFetchParameters"]; !ok {
			return nil, fmt.Errorf("YAML File is not valid for this usecase %w", ok)
		} else if steps, ok := testF.(map[string]interface{})["Steps"]; !ok {
			return nil, fmt.Errorf("YAML File is not valid for this usecase %w", ok)
		} else {
			switch val := steps.(type) {
			case []interface{}:
				for _, v := range val {
					switch val := v.(type) {
					case map[string]interface{}:
						for k, v := range val {
							// If the label of the test step we want to edit is correct, we will edit the arguements
							if k == "label" && v == "Checkout to the right commit" {
								switch val := val["parameters"].(type) {
								case map[string]interface{}:
									for k, v := range val {
										if k == "args" {
											args := v.([]interface{})
											args[1] = webhookData[0] // Edit the commit SHA
										}
									}
								default:
									fmt.Println("YAML File has wrong format")
								}
							}
						}
					default:
						fmt.Println("YAML File has wrong format")
					}
				}
			default:
				fmt.Println("YAML File has wrong format")
			}
		}
		// marshal the jobDescriptor into JSON format
		jobDescJSON, err := json.MarshalIndent(jobDesc, "", "    ")
		if err != nil {
			return nil, fmt.Errorf("failed to serialize job descriptor to JSON: %w", err)
		}
		// Return the adapted jobDescriptor
		return jobDescJSON, nil
	}
}
