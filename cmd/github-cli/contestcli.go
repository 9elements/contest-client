package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/9elements/contest-client/pkg/client/clientpluginregistry"
	"github.com/9elements/contest-client/pkg/webhook"
	"github.com/9elements/contest-client/plugins/clientplugins"
	"github.com/facebookincubator/contest/pkg/api"
	"github.com/facebookincubator/contest/pkg/job"
	"github.com/facebookincubator/contest/pkg/logging"
	"github.com/facebookincubator/contest/pkg/transport"
	"github.com/facebookincubator/contest/pkg/transport/http"
	"github.com/facebookincubator/contest/pkg/types"
	"github.com/facebookincubator/contest/pkg/xcontext"
	"github.com/facebookincubator/contest/pkg/xcontext/bundles/logrusctx"
	"github.com/icza/dyno"
	"gopkg.in/yaml.v2"
)

var bundleIntegrationHooks []*client.IntegrationHookBundle

func CLIMain(config *client.ClientDescriptor, stdout io.Writer) error {

	// We hard code this here now

	// Create context
	ctx, cancel := xcontext.WithCancel(logrusctx.NewContext(6, logging.DefaultOptions()...))
	defer cancel()

	// Initialize the plugin registry
	clientPluginRegistry := clientpluginregistry.NewClientPluginRegistry(ctx)
	clientplugins.Init(clientPluginRegistry, ctx.Logger())

	// Init and Start the webhook listener
	var weblistener webhook.Webhook
	weblistener.NewListener()

	go weblistener.Start()

	// Iterate over every incoming webhook
	for nextWebhookData := range weblistener.Data {
		// Iterate over all PreJobExecution plugins
		for _, eh := range config.PreJobExecutionHooks {
			// Validate the current plugin
			if err := eh.PreValidate(); err != nil {
				return err
			}
			// Register the current plugin
			bundlePreExecutionHook, err := clientPluginRegistry.NewPreJobExecutionHookBundle(ctx, eh)
			if err != nil {
				return err
			}
			// Run the plugin
			if _, err = bundlePreExecutionHook.PreJobExecutionHooks.Run(ctx, bundlePreExecutionHook.Parameters, *config, &http.HTTP{Addr: *config.Configuration.Addr + *config.Configuration.PortServer}); err != nil {
				return err
			}
		}

		for _, eh := range config.IntegrationHooks {
			err := eh.PreValidate()
			if err != nil {
				return err
			}

			bundleIntegrationHook, err := clientPluginRegistry.NewIntegrationHookBundle(ctx, eh)
			if err != nil {
				return err
			}

			err = bundleIntegrationHook.IntegrationHooks.Setup(ctx, bundleIntegrationHook.Parameters)
			if err != nil {
				fmt.Printf("Setup throws an error: %v\n", err)
			}

			err = bundleIntegrationHook.IntegrationHooks.BeforeJob(ctx, nextWebhookData.HeadSHA, *config)
			if err != nil {
				fmt.Printf("BeforeJob throws an error: %v\n", err)
			}

			bundleIntegrationHooks = append(bundleIntegrationHooks, bundleIntegrationHook)
		}

		// Run the job and receive the rundata
		var rundata []client.RunData
		rundata, err := run(ctx, *config, &http.HTTP{Addr: *config.Configuration.Addr + *config.Configuration.PortServer}, stdout, nextWebhookData, weblistener)
		if err != nil {
			fmt.Printf("running the job failed (err: %v) You should probably check the connection and restart the test", err)
			continue
		}
		// Iterate over all PostJobExecution plugins
		for _, eh := range config.PostJobExecutionHooks {
			// Validate the current plugin
			if err := eh.PostValidate(); err != nil {
				return err
			}
			// Register the current plugin
			bundlePostExecutionHook, err := clientPluginRegistry.NewPostJobExecutionHookBundle(ctx, eh)
			if err != nil {
				fmt.Println("Name()")
				return err
			}
			// Run the plugin
			if _, err := bundlePostExecutionHook.PostJobExecutionHooks.Run(ctx, bundlePostExecutionHook.Parameters, *config, &http.HTTP{Addr: *config.Configuration.Addr + *config.Configuration.PortServer}, rundata); err != nil {
				return err
			}
		}

		for _, bundleIntegrationHook := range bundleIntegrationHooks {
			err = bundleIntegrationHook.IntegrationHooks.AfterJob(ctx, rundata)
			if err != nil {
				fmt.Printf("AfterJob throw an error: %v\n", err)
			}
		}
	}

	return nil
}

// Struct that contains all possible template parameters
type templatedata struct {
	SHA string
}

/* Function run runs the main functionility of the contest-client.
   It creates new jobDescriptors and kicks off new jobs.
   It also sets the github commit status to pending if the job was started */
func run(ctx context.Context, cd client.ClientDescriptor, transport transport.Transport, stdout io.Writer,
	webhookData webhook.WebhookData, listener webhook.Webhook) ([]client.RunData, error) {

	// Declare a jobs []struct that contains the rundata that shall be passed
	var jobs []client.RunData

	// Iterate over all JobTemplates that are defined in the clientconfig.json
	for i := 0; i < len(cd.Configuration.JobTemplate); i++ {

		// Create Path to the jobTemplate
		filePath, _ := filepath.Abs("descriptors/")
		filePathTemplate := strings.Join([]string{filePath, *cd.Configuration.JobTemplate[i]}, "/")

		// Parse the json/yaml file
		templateDescription, err := os.ReadFile(filePathTemplate)
		if err != nil {
			return nil, fmt.Errorf("could not parse the jobtemplate: %w", err)
		}

		// Retrieve the jobName for further usages
		jobName, err := RetrieveJobName(templateDescription, *cd.Configuration.YAML)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve the job name: %w", err)
		}

		// Adapt the jobDescriptor based on the webhookdata
		jobDesc, err := ChangeJobDescriptor(templateDescription, webhookData)
		if err != nil {
			return nil, fmt.Errorf("could not change the job template: %w", err)
		}

		// If template file is YAML convert it to JSON
		if *cd.Configuration.YAML {
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
		startResp, err := transport.Start(context.Background(), *cd.Configuration.Requestor, string(jobDesc))
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

		// Filling the map with job data for postjobexecutionhooks
		jobData := client.RunData{
			JobID:      int(startResp.Data.JobID),
			JobContext: *cd.Configuration.JobTemplate[i],
			JobName:    jobName,
			JobSHA:     webhookData.HeadSHA}

		// Now wait for the results if desired
		if *cd.Configuration.Wait {
			// print immediately if wait is used
			buffer := &bytes.Buffer{}
			encoder := json.NewEncoder(buffer)
			encoder.SetEscapeHTML(false)
			encoder.SetIndent("", " ")
			err = encoder.Encode(startResp)
			if err != nil {
				return nil, fmt.Errorf("cannot re-encode api.Respose object: %v", err)
			}
			indentedJSON := buffer.String()
			fmt.Fprintf(stdout, "%s", string(indentedJSON))

			fmt.Fprintf(os.Stderr, "\nWaiting for job to complete...\n")
			jobData.JobStatus, err = wait(context.Background(), startResp.Data.JobID, time.Second*time.Duration(*cd.Configuration.JobWaitPoll), *cd.Configuration.Requestor, transport)
			if err != nil {
				return nil, err
			}
		}

		jobs = append(jobs, jobData)

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

func wait(ctx context.Context, jobID types.JobID, jobWaitPoll time.Duration, requestor string, transport transport.Transport) (*api.StatusResponse, error) {
	// keep polling for status till job is completed, used when -wait is set
	for {
		resp, err := transport.Status(context.Background(), requestor, jobID)
		if err != nil {
			return nil, err
		}
		if resp.Err != nil {
			return nil, fmt.Errorf("server responded with an error: %s", resp.Err)
		}

		jobState := resp.Data.Status.State

		for _, eventName := range job.JobCompletionEvents {
			if string(jobState) == string(eventName) {
				return resp, nil
			}
		}
		// TODO use  time.Ticker instead of time.Sleep
		time.Sleep(jobWaitPoll)
	}
}
