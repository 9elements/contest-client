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
	// getting env variables SITE_TITLE and DB_HOST
	githubToken := os.Getenv("GH_TOKEN")

	//setting up the github authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	_, err := url.ParseRequestURI(targeturl)
	if err != nil {
		return fmt.Errorf("TargetURL of the results is not formatted right! GithubStatus could not be edited")
	}

	match, err := regexp.MatchString("[a-f0-9]{40}", sha)

	if !match {
		log.Printf("the commit sha was not handed over correctly!\n")
		return err
	}
	input := &github.RepoStatus{State: &state, TargetURL: &targeturl, Context: &description}
	_, _, err = client.Repositories.CreateStatus(ctx, "9eCyberSec", "coreboot-spr-sp", sha, input)
	if err != nil {
		log.Printf("could not set status of the commit to %s: err=%s\n", state, err)
	}
	return nil
}
