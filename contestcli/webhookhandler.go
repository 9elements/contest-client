package contestcli

import (
	"fmt"
	"log"
	"net/http"

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
	log.Fatal(http.ListenAndServe("0.0.0.0:6000", nil))
}

//handleWebhook handles the incoming webhooks and then adapt the jobDescriptor templates to kick off new jobs depending on them
func (channel *Channel) handleWebhook(w http.ResponseWriter, r *http.Request) {
	var tmp []string //declare variable to pass data through a channel

	payload, err := github.ValidatePayload(r, []byte("thisisasecret")) //receiving and validating the incoming webhook
	if err != nil {
		log.Printf("error reading request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()
	event, err := github.ParseWebHook(github.WebHookType(r), payload) //parsing the incoming webhook
	if err != nil {
		log.Printf("could not parse incoming webhook: %v\n", err)
		return
	}
	switch e := event.(type) { //switch to handle the different eventtypes of the webhook
	case *github.PullRequestEvent:
		if *e.Action == "synchronize" || *e.Action == "closed" {
			return
		}
		fmt.Printf("successful received pullrequest event\n")
		tmp = append(tmp, *e.PullRequest.Head.SHA, *e.PullRequest.Head.Repo.HTMLURL, *e.PullRequest.Head.Ref)
		channel.webhookdata <- tmp
	case *github.PushEvent:
		fmt.Printf("successful received push event\n")
		tmp = append(tmp, *e.After, *e.Repo.HTMLURL)
		channel.webhookdata <- tmp
	default:
		log.Printf("successful received unknown event %s %s\n", github.WebHookType(r), e)
		return
	}
}
