package clientapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type TestAPI struct {
}

type Github interface {
	EditGithubStatus(ctx context.Context, state string, targeturl string, description string, sha string) error
}

type Slack interface {
	MsgToSlack(msg string) error
}

func (t TestAPI) EditGithubStatus(ctx context.Context, state string, targeturl string, description string, sha string) error {
	// Getting env variable GH_TOKEN
	githubToken := os.Getenv("GITHUB_TOKEN")

	// Setting up the github authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	// Setting up a github client
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	if client == nil {
		return fmt.Errorf("the github client has not set up")
	}

	// If the state is different to the possible github states, error
	if state != "error" && state != "failure" && state != "pending" && state != "success" {
		return fmt.Errorf("state has no correct value")
	}

	// If the targetURL is not empty and wrong formatted, error
	_, err := url.ParseRequestURI(targeturl)
	if err != nil {
		return fmt.Errorf("TargetURL of the results is not formatted right! GithubStatus could not be edited")
	}
	// Check if sha is a correct formatted sha1 hash else, error
	match, err := regexp.MatchString("[a-f0-9]{40}", sha)
	if !match {
		return fmt.Errorf("the commit sha was not handed over correctly: %w", err)
	}
	return nil
}

func (t TestAPI) MsgToSlack(msg string) error {
	// Getting env variable SLACK_WEBHOOK_URL
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")

	// Creating the body and the request
	Body, _ := json.Marshal(RequestBody{Message: msg})
	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(Body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	return nil
}
