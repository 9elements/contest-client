package contestcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/9elements/contest-client/pkg/client"
	githubAPI "github.com/9elements/contest-client/pkg/client/github"
	"github.com/Navops/yaml"
	"github.com/facebookincubator/contest/pkg/transport"
	"github.com/facebookincubator/contest/pkg/types"
)

func run(ctx context.Context, cd client.ClientDescriptor, transport transport.Transport, stdout io.Writer, webhookData []string) (map[int][2]string, error) {

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
			*cd.Flags.FlagJobTemplate[i], webhookData[0])
		if res != nil {
			log.Printf("could not change the github status: %s\n", res)
		}
		//adapt the jobDescriptor based on the pullrequest
		jobDesc, errorr := ChangeJobDescriptor(templateDescription, *cd.Flags.FlagYAML, webhookData)
		if errorr != nil {
			fmt.Printf("could not change the job template: %+v\n", err)
		}
		//kick off the generated Job
		startResp, err := transport.Start(context.Background(), *cd.Flags.FlagRequestor, string(jobDesc))
		if err != nil {
			fmt.Printf("could not send the Job with the jobDesc: %s\n", *cd.Flags.FlagJobTemplate[i])
		}
		if int(startResp.Data.JobID) == 0 {
			fmt.Printf("The Job could not executed. Server returned JobID 0! \n") //TODO: ERROR HANDLING
		}
		var jobNameSha [2]string
		jobNameSha[0] = *cd.Flags.FlagJobTemplate[i]
		jobNameSha[1] = webhookData[0]
		jobs[int(startResp.Data.JobID)] = jobNameSha

		//Create Json Body for API Request to set a status for the started Job
		data := map[string]interface{}{
			"ID":     startResp.Data.JobID,
			"Status": false,
		}
		json_data, err := json.Marshal(data) //Marshal that data
		if err != nil {
			fmt.Println("Could not parse data to json format.")
		}
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

	if !YAML { //check if the file is YAML or JSON and depending on it Unmarshal it, adapt it and marshal it again
		if err := json.Unmarshal(data, &jobDesc); err != nil {
			return nil, fmt.Errorf("failed to parse JSON job descriptor: %w", err)
		}
		if testD, ok := jobDesc["TestDescriptors"].([]interface{}); !ok {
			return nil, fmt.Errorf("JSON File is not valid for this usecase", ok)
		} else if testF, ok := testD[0].(map[string]interface{})["TestFetcherFetchParameters"]; !ok {
			return nil, fmt.Errorf("JSON File is not valid for this usecase", ok)
		} else if steps, ok := testF.(map[string]interface{})["Steps"]; !ok {
			return nil, fmt.Errorf("JSON File is not valid for this usecase", ok)
		} else {
			switch val := steps.(type) {
			case []interface{}:
				for _, v := range val {
					switch val := v.(type) {
					case map[string]interface{}:
						for k, v := range val {
							if k == "label" && v == "Cloning coreboot" {
								switch val := val["parameters"].(type) {
								case map[string]interface{}:
									for k, v := range val {
										if k == "args" {
											args := v.([]interface{})
											args[1] = webhookData[1] //Repository URL
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
		return jobDescJSON, nil
	} else {
		if err := yaml.Unmarshal(data, &jobDesc); err != nil {
			return nil, fmt.Errorf("failed to parse YAML job descriptor: %w", err)
		}
		//fmt.Println(jobDesc)
		if testD, ok := jobDesc["TestDescriptors"].([]interface{}); !ok {
			return nil, fmt.Errorf("YAML File is not valid for this usecase", ok)
		} else if testF, ok := testD[0].(map[string]interface{})["TestFetcherFetchParameters"]; !ok {
			return nil, fmt.Errorf("YAML File is not valid for this usecase", ok)
		} else if steps, ok := testF.(map[string]interface{})["Steps"]; !ok {
			return nil, fmt.Errorf("YAML File is not valid for this usecase", ok)
		} else {
			switch val := steps.(type) {
			case []interface{}:
				for _, v := range val {
					switch val := v.(type) {
					case map[string]interface{}:
						for k, v := range val {
							if k == "label" && v == "Cloning coreboot" {
								switch val := val["parameters"].(type) {
								case map[string]interface{}:
									for k, v := range val {
										if k == "args" {
											args := v.([]interface{})
											args[1] = webhookData[1] //Repository URL
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
		return jobDescJSON, nil
	}
}

func parseJob(jobIDStr string) (types.JobID, error) {
	if jobIDStr == "" {
		return 0, errors.New("missing job ID")
	}
	var jobID types.JobID
	jobIDint, err := strconv.Atoi(jobIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid job ID: %s: %v", jobIDStr, err)
	}
	jobID = types.JobID(jobIDint)
	if jobID <= 0 {
		return 0, fmt.Errorf("invalid job ID: %s: it must be positive", jobIDStr)
	}
	return jobID, nil
}
