package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/Navops/yaml"
	"github.com/facebookincubator/contest/pkg/transport"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

//read the specific template, change the content and write it back to the target descriptor file
func OpenAndWriteJobDescriptor(ctx context.Context, cd client.ClientDescriptor, transport transport.Transport, webhookData []string) (jobIDs []int) {
	//setting up the github authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ""},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	match, _ := regexp.MatchString("[a-f0-9]{40}", webhookData[0])
	if !match { //catch wrong commit sha's
		log.Printf("the commit sha was not handed over correctly!\n")
		return
	}

	for i := 0; i <= len(cd.Flags.FlagJobTemplate)-1; i++ { //iterate over all JobTemplates that are defined in the clientconfig.json
		//Open the configfile
		filepath, _ := filepath.Abs("descriptors/")
		filepathtemplate := filepath + "/" + *cd.Flags.FlagJobTemplate[i]
		//parse and decode the json/yaml file
		templateDescription, err := ioutil.ReadFile(filepathtemplate)
		if err != nil {
			log.Printf("could not parse the jobtemplate: %s\n", err)
		}

		//setting the status of the commit to pending depending on what jobtemplate and database
		state := "pending"
		targeturl := "http://someurl.com"
		description := *cd.Flags.FlagJobTemplate[i]
		input := &github.RepoStatus{State: &state, TargetURL: &targeturl, Context: &description}
		_, _, erro := client.Repositories.CreateStatus(ctx, "llogen", "webhook", webhookData[0], input)
		if erro != nil {
			log.Printf("could not set status of the commit to pending: err=%s\n", err)
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
		jobIDs = append(jobIDs, int(startResp.Data.JobID))
	}
	fmt.Println(jobIDs)
	return jobIDs
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
											args[1] = webhookData[1]
										}
									}
								default:
									fmt.Println("JSON File has wrong format")
								}
							}
							if k == "label" && v == "Working on the right commit" {
								switch val := val["parameters"].(type) {
								case map[string]interface{}:
									for k, v := range val {
										if k == "args" {
											args := v.([]interface{})
											args[1] = webhookData[0]
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
											args[1] = webhookData[1]
										}
									}
								default:
									fmt.Println("YAML File has wrong format")
								}
							}
							if k == "label" && v == "Working on the right commit" {
								switch val := val["parameters"].(type) {
								case map[string]interface{}:
									for k, v := range val {
										if k == "args" {
											args := v.([]interface{})
											args[1] = webhookData[0]
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
		// marshal the jobDescriptor back to YAML format
		jobDescYAML, err := yaml.Marshal(jobDesc)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize job descriptor to JSON: %w", err)
		}
		return jobDescYAML, nil
	}
}
