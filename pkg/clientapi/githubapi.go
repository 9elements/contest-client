package clientapi

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type GithubAPI struct {
}

func (g GithubAPI) EditGithubStatus(ctx context.Context, state string, targeturl string, description string, sha string) error {
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
	// If the targetURL is not empty and wrong formatted, return
	_, err := url.ParseRequestURI(targeturl)
	if err != nil {
		return fmt.Errorf("TargetURL of the results is not formatted right! GithubStatus could not be edited")
	}
	// Check if sha is a correct formatted sha1 hash else return
	match, err := regexp.MatchString("[a-f0-9]{40}", sha)
	if !match {
		return fmt.Errorf("the commit sha was not handed over correctly: %w", err)
	}
	// Putting the CreateStatus input together and change the status of the commit
	input := &github.RepoStatus{State: &state, TargetURL: &targeturl, Context: &description}

	_, _, err = client.Repositories.CreateStatus(ctx, "9elements", "coreboot-spr-sp", sha, input)
	if err != nil {
		return fmt.Errorf("could not set status of the commit to %s, err: %s", state, err)
	}
	return nil
}
