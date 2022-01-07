package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var Name = "github"

type Github struct {
	Repository     string
	Organization   string
	Token          string
	Secret         string
	GithubStatuses map[string]GithubStatus
}

type GithubStatus struct {
	context     string // This should not be changed as this is unique
	state       string
	description string
	targetURL   string
	sha         string
}

var validStates = map[string]bool{
	"error":   true,
	"failure": true,
	"pending": true,
	"success": true,
}

func checkIfStateIsValid(lookup string) bool {
	if _, ok := validStates[lookup]; ok {
		return true
	} else {
		return false
	}
}

func (g *Github) readParameters(parameters interface{}) {
	// TODO: I hope this can be done better
	g.Repository = parameters.(Github).Repository
	g.Organization = parameters.(Github).Organization
	g.Token = parameters.(Github).Token
	g.Secret = parameters.(Github).Secret
}

func (g *Github) Setup(ctx context.Context, parameters interface{}) error {
	g.readParameters(parameters)

	fmt.Printf("Read in Parameters: %v\n", g)

	g.GithubStatuses = make(map[string]GithubStatus)

	return nil
}

func (g *Github) BeforeJob(ctx context.Context, parameters interface{}, clientDescriptor client.ClientDescriptor) error {
	// We need to trigger the Github Status here
	sha := parameters.(string)

	// Setting up the github authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: g.Token},
	)
	// Setting up a github client
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	if client == nil {
		return fmt.Errorf("github.NewClient failed with oauth2 token.")
	}

	// Generate a Status for each Job
	for _, job := range clientDescriptor.Configuration.JobTemplate {
		g.GithubStatuses[*job] = GithubStatus{
			context:     fmt.Sprintf("ConTest: %s", *job),
			state:       "pending",
			description: "The Job is about to start..",
			sha:         sha,
		}

		// TODO: Why do we need to do this? Idk
		currentState := g.GithubStatuses[*job].state
		context := g.GithubStatuses[*job].context
		description := g.GithubStatuses[*job].description
		targeturl := g.GithubStatuses[*job].targetURL

		// Putting the CreateStatus input together and change the status of the commit
		input := &github.RepoStatus{State: &currentState, Context: &context, Description: &description, TargetURL: &targeturl}

		_, resp, err := client.Repositories.CreateStatus(ctx, g.Organization, g.Repository, g.GithubStatuses[*job].sha, input)
		if err != nil {
			return fmt.Errorf("could not set status of the commit. Github Status: %v\n\nerr: %s\nResponse: %v",
				g.GithubStatuses[*job], err, resp)
		}
	}

	return nil
}

func (g *Github) InJobUpdate(ctx context.Context, parameters interface{}) error {
	return nil
}

func (g *Github) AfterJob(ctx context.Context, parameters interface{}) error {
	// We need to update the Github Status here to fail/success
	var data client.RunData

	runData := parameters.([]client.RunData)

	// Setting up the github authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: g.Token},
	)
	// Setting up a github client
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	if client == nil {
		return fmt.Errorf("the github client has not set up")
	}

	// Get some context from somewhere like URLs and so on. We could pass the result in here and do the parsing here
	// Generate a Status for each Job
	fmt.Printf("We have %d github statuses right now\n", len(g.GithubStatuses))
	for _, githubstatus := range g.GithubStatuses {

		// Search correct runData
		for _, entry := range runData {
			jobcontest := fmt.Sprintf("ConTest: %s", entry.JobContext) // This needs to be in sync with the upper context
			if githubstatus.context == jobcontest {
				data = entry
			}
		}

		if data.JobID == 0 {
			// We have not found the data. Error.
			return fmt.Errorf("Matching entry not found.")
		}

		JobStatus := *data.JobStatus.Data.Status
		githubstatus.state = "failure"

		if JobStatus.State == "JobStateCompleted" {

			githubstatus.description = data.JobStatus.Data.Status.StateErrMsg

			success := true
			for _, reports := range JobStatus.JobReport.RunReports[0] {
				success = success && reports.Success
			}

			if success {
				githubstatus.state = "success"
			}
		}

		// Putting the CreateStatus input together and change the status of the commit
		input := &github.RepoStatus{State: &githubstatus.state, Context: &githubstatus.context, Description: &githubstatus.description, TargetURL: &githubstatus.targetURL}

		_, resp, err := client.Repositories.CreateStatus(ctx, g.Organization, g.Repository, githubstatus.sha, input)
		if err != nil {
			return fmt.Errorf("could not set status of the commit. Github Status: %v\n\nerr: %s\nResponse: %v",
				githubstatus, err, resp)
		}
	}

	return nil
}

func (g *Github) ValidateParameters(params []byte) (interface{}, error) {
	var githubParameters Github
	err := json.Unmarshal(params, &githubParameters)
	if err != nil {
		return nil, err
	}

	githubParameters.Token = os.Getenv("GITHUB_TOKEN")
	githubParameters.Secret = os.Getenv("GITHUB_SECRET")

	return githubParameters, nil
}

func (g *Github) Name() string {
	return Name
}

func New() client.IntegrationHooks {
	return &Github{}
}

func Load() (string, client.IntegrationHooksFactory) {
	return Name, New
}
