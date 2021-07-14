package githubAPI

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func EditGithubStatus(ctx context.Context, state string, targeturl string, description string, sha string) error {
	//setting up the github authentication
	f, err := os.Open("githubtoken.json")
	if err != nil {
		fmt.Printf("Could no open the github token file!\n")
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	if err != nil {
		fmt.Printf("Could not decode the json file")
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: tok.AccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	_, err = url.ParseRequestURI(targeturl)
	if err != nil {
		return fmt.Errorf("TargetURL of the results is not formatted right! GithubStatus could not be edited")
	}

	match, err := regexp.MatchString("[a-f0-9]{40}", sha)
	if !match { //catch wrong commit sha's
		log.Printf("the commit sha was not handed over correctly!\n")
		return err
	}
	input := &github.RepoStatus{State: &state, TargetURL: &targeturl, Context: &description}
	_, _, err = client.Repositories.CreateStatus(ctx, "llogen", "webhook", sha, input)
	if err != nil {
		log.Printf("could not set status of the commit to %s: err=%s\n", state, err)
	}
	return nil
}
