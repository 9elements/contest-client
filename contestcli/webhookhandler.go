package contestcli

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/google/go-github/github"
)

type WebhookData struct {
	headSHA string
	sshURL  string
	refSHA  string
}
type Channel struct {
	webhookdata chan WebhookData
}

func webhook(webhookData chan WebhookData) {
	// Start webhook listener
	channel := &Channel{webhookdata: webhookData}
	log.Println("webhook listener is running and running")
	http.HandleFunc("/", channel.handleWebhook)
	log.Fatal(http.ListenAndServeTLS("0.0.0.0:6000", "/certs/fullchain.crt", "/certs/server.key", nil))
}

// HandleWebhook handles incoming webhooks
func (channel *Channel) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Log request
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(dump)

	//retrieve the github_secret for the webhook from .env
	github_secret := os.Getenv("GITHUB_SECRET")
	// Receiving and validating the incoming webhook
	payload, err := github.ValidatePayload(r, []byte(github_secret))
	if err != nil {
		log.Printf("error reading request body: err=%s\n", err)
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
		webhookdata.headSHA = *e.PullRequest.Head.SHA
		webhookdata.sshURL = *e.PullRequest.Head.Repo.SSHURL
		webhookdata.refSHA = *e.PullRequest.Head.Ref
		channel.webhookdata <- webhookdata
	case *github.PushEvent:
		// Assign 1.the commmit SHA, 2.the SSH link and than pass it to the channel
		fmt.Printf("successful received push event\n")
		var webhookdata WebhookData
		webhookdata.headSHA = *e.After
		webhookdata.sshURL = *e.Repo.SSHURL
		channel.webhookdata <- webhookdata
	default:
		log.Printf("successful received unknown event %s %s\n", github.WebHookType(r), e)
		return
	}
}
