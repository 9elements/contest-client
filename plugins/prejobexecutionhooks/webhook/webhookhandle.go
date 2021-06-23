package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/Navops/yaml"
	"github.com/facebookincubator/contest/pkg/transport"
	"github.com/google/go-github/github"
)

var globalctx context.Context
var globalcd client.ClientDescriptor
var globaltransport transport.Transport

func startWebhook(ctx context.Context, cd client.ClientDescriptor, transport transport.Transport) {
	//define the parameter to use it in handleWebhook
	globalctx = ctx
	globalcd = cd
	globaltransport = transport
	//start webhook listener
	log.Println("webhook listener started")
	http.HandleFunc("/", handleWebhook)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

//handleWebhook handles the incoming webhooks and then adapt the jobDescriptor templates to kick off new jobs depending on them
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte("secret")) //receiving and validating the incoming webhook
	if err != nil {
		log.Printf("error reading request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()
	event, err := github.ParseWebHook(github.WebHookType(r), payload) //parsing the incoming webhook
	if err != nil {
		log.Printf("could not parse incoming webhook: %v\n", err)
		return
	}
	fmt.Printf("successful received incoming webhook event\n")
	switch e := event.(type) { //switch to handle the different eventtypes of the webhook
	case *github.PullRequestEvent:
		fmt.Printf("successful received pullrequestevent event\n")
		githubRepoURL := *e.PullRequest.Head.Repo.HTMLURL
		githubHeadBranch := *e.PullRequest.Head.Ref
		fmt.Printf("event: %+v\n", *e.PullRequest.Head.Repo.HTMLURL)
		fmt.Printf("event: %+v\n", *e.PullRequest.Head.Ref)
		//if e.PullRequest.MergedAt != nil { //if merged
		OpenAndWriteJobDescriptor(globalctx, globalcd, globaltransport, githubRepoURL, githubHeadBranch) //call to write content of the webhook into the jobDesc template
		//}

	default:
		log.Printf("unknown event type %s %s\n", github.WebHookType(r), e)
		return
	}
}

//read the specific template, change the content and write it back to the target descriptor file
func OpenAndWriteJobDescriptor(ctx context.Context, cd client.ClientDescriptor, transport transport.Transport, githubRepoURL string, githubHeadBranch string) {
	var resp interface{}
	for i := 0; i < len(Targettemplates); i++ {
		//Open the configfile
		filepath, _ := filepath.Abs("descriptors/")
		filepathtemplate := filepath + "/" + Targettemplates[i]
		//parse and decode the json/yaml file
		templateDescription, err := ioutil.ReadFile(filepathtemplate)
		if err != nil {
			fmt.Println(err)
			continue
		}
		//adapt the jobDescriptor based on the pullrequest
		jobDesc, err := ChangeJobDescriptor(templateDescription, *cd.Flags.FlagYAML, githubRepoURL, githubHeadBranch)
		if err != nil {
			fmt.Printf("could not change the job template: %+v\n", err)
		}
		//kick off the generated Job
		startResp, err := transport.Start(context.Background(), *cd.Flags.FlagRequestor, string(jobDesc))
		if err != nil {
			fmt.Printf("could not send the Job with the jobDesc: %s\n", Targettemplates[i])
		}
		resp = startResp
		_ = resp
	}
}

//unmarshal the data from the template file then change the specific value and marshal it back to specific format
func ChangeJobDescriptor(data []byte, YAML bool, githubRepoURL string, githubHeadBranch string) ([]byte, error) {
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
											args[0] = "clone --branch"
											args[1] = githubHeadBranch
											args[2] = githubRepoURL
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
											args[0] = "clone --branch"
											args[1] = githubHeadBranch
											args[2] = githubRepoURL
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
