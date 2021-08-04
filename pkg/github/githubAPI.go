package githubAPI

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func EditGithubStatus(ctx context.Context, state string, targeturl string, description string, sha string) error {
	// getting env variable GH_TOKEN
	githubToken := os.Getenv("GH_TOKEN")

	//setting up the github authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	//setting up a github client
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	//if the targetURL is not empty and wrong formatted, return
	_, err := url.ParseRequestURI(targeturl)
	if err != nil && targeturl != "" {
		return fmt.Errorf("TargetURL of the results is not formatted right! GithubStatus could not be edited")
	}
	//check if sha is a correct formatted sha1 hash else return
	match, err := regexp.MatchString("[a-f0-9]{40}", sha)
	if !match {
		fmt.Printf("Error is: %v\n", err)
		return fmt.Errorf("the commit sha was not handed over correctly")
	}
	//Putting the CreateStatus input together and change the status of the commit
	var input *github.RepoStatus
	if state != "" {
		input = &github.RepoStatus{State: &state, TargetURL: &targeturl, Context: &description}
	} else {
		input = &github.RepoStatus{TargetURL: &targeturl, Context: &description}
	}

	_, _, err = client.Repositories.CreateStatus(ctx, "9elements", "coreboot-spr-sp", sha, input)
	if err != nil {
		log.Printf("could not set status of the commit to %s: err=%s\n", state, err)
	}
	return nil
}
