package slackIntegration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type RequestBody struct {
	Message string `json:"text"`
}

type SlackConfiguration struct {
	WebhookUrl string
}

// MsgToSlack will post a message to a slack webhook. It receives a message and post it.
func (s SlackConfiguration) MsgToSlack(msg string) error {
	// Getting env variable SLACK_WEBHOOK_URL
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")

	// Creating the body and the request
	Body, _ := json.Marshal(RequestBody{Message: msg})
	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(Body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return fmt.Errorf("slack answer was not ok")
	}
	return nil
}
