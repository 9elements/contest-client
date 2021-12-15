package webhook

import (
	"fmt"
	"log"
	"net/http"
	"os"

	githubIntegration "github.com/9elements/contest-client/pkg/integrations/github"
	slackIntegration "github.com/9elements/contest-client/pkg/integrations/slack"
	"github.com/google/go-github/github"
)

type WebhookData struct {
	HeadSHA string
	SshURL  string
	RefSHA  string
}

type Webhook struct {
	Data                chan WebhookData
	GithubConfiguration githubIntegration.GithubConfiguration
	SlackConfiguration  slackIntegration.SlackConfiguration
}

func (w *Webhook) NewListener() {
	w.Data = make(chan WebhookData, 10)
}

func (w *Webhook) Start() error {
	log.Println("webhook listener is running and running")
	http.HandleFunc("/", w.handleWebData)
	err := http.ListenAndServeTLS("0.0.0.0:6000", "/certs/fullchain.crt", "/certs/server.key", nil)
	if err != nil {
		return fmt.Errorf("error listening to the webhook, err: %v", err)
	}
	return nil
}

func (w *Webhook) handleWebData(rw http.ResponseWriter, r *http.Request) {
	//retrieve the github_secret for the webhook from .env
	github_secret := os.Getenv("GITHUB_SECRET")
	// Receiving and validating the incoming webhook
	payload, err := github.ValidatePayload(r, []byte(github_secret))
	if err != nil {
		log.Printf("error reading request body, err: %s\n", err)
		return
	}
	defer r.Body.Close()

	// Parsing the incoming webhook
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("could not parse incoming webhook: %v\n", err)
		return
	}

	// Switch to handle the different eventtypes of the webhook
	switch e := event.(type) {
	case *github.PullRequestEvent:
		if *e.Action == "synchronize" || *e.Action == "closed" {
			return
		}
		fmt.Printf("successful received pullrequest event\n")
		// Assign 1.the HEAD SHA, 2.the SSH link, 3.the Reference SHA and than pass it to the channel
		var webhookdata WebhookData
		webhookdata.HeadSHA = *e.PullRequest.Head.SHA
		webhookdata.SshURL = *e.PullRequest.Head.Repo.SSHURL
		webhookdata.RefSHA = *e.PullRequest.Head.Ref
		w.Data <- webhookdata
	case *github.PushEvent:
		// Assign 1.the commmit SHA, 2.the SSH link and than pass it to the channel
		fmt.Printf("successful received push event\n")
		var webhookdata WebhookData
		webhookdata.HeadSHA = *e.After
		webhookdata.SshURL = *e.Repo.SSHURL
		w.Data <- webhookdata
	default:
		log.Printf("successful received unknown event %s %s\n", github.WebHookType(r), e)
		return
	}
}
