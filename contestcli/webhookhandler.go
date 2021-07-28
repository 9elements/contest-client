package contestcli

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
)

type Channel struct {
	webhookdata chan []string
}

func webhook(webhookData chan []string) {
	//start webhook listener
	channel := &Channel{webhookdata: webhookData}
	log.Println("webhook listener is running and running")
	http.HandleFunc("/", channel.handleWebhook)
	log.Fatal(http.ListenAndServeTLS("0.0.0.0:6000", "/certs/fullchain.crt", "/certs/server.key", nil))
}

//handleWebhook handles incoming webhooks
func (channel *Channel) handleWebhook(w http.ResponseWriter, r *http.Request) {
	//declare variable to pass data through a channel
	var tmp []string

	//retrieve the github_secret for the webhook from .env
	github_secret := os.Getenv("GITHUB_SECRET")
	//receiving and validating the incoming webhook
	payload, err := github.ValidatePayload(r, []byte(github_secret))
	if err != nil {
		log.Printf("error reading request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()

	//parsing the incoming webhook
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("could not parse incoming webhook: %v\n", err)
		return
	}

	//switch to handle the different eventtypes of the webhook
	switch e := event.(type) {
	case *github.PullRequestEvent:
		if *e.Action == "synchronize" || *e.Action == "closed" {
			return
		}
		fmt.Printf("successful received pullrequest event\n")
		//add, 1.the HEAD SHA, 2.the SSH link, 3.the Reference SHA, to tmp and than pass it to the channel
		tmp = append(tmp, *e.PullRequest.Head.SHA, *e.PullRequest.Head.Repo.SSHURL, *e.PullRequest.Head.Ref)
		channel.webhookdata <- tmp
	case *github.PushEvent:
		//add, 1.the commmit SHA, 2.the SSH link, to tmp and than pass it to the channel
		fmt.Printf("successful received push event\n")
		tmp = append(tmp, *e.After, *e.Repo.SSHURL)
		channel.webhookdata <- tmp
	default:
		log.Printf("successful received unknown event %s %s\n", github.WebHookType(r), e)
		return
	}
}
