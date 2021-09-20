package clientapi

import (
	"context"
	"testing"
)

const (
	state       = "pending"
	targeturl   = "https://www.test.com/"
	description = "test"
	sha         = "a94a8fe5ccb19ba61c4c0873daa391e9872fbbd3"
)

// Test for EditGithubStatus, if the function wraps correctly the githubAPI
func TestClientApi(t *testing.T) {

	t.Run("Github", func(t *testing.T) {
		Test := TestAPI{}
		var ctx context.Context

		err := Test.EditGithubStatus(ctx, state, targeturl, description, sha)
		if err != nil {
			t.Errorf("function 'EditGithubStatus' returned an error: %w", err)
		}
		// Define want and got and compare them
		var want error = nil
		got := err
		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Slack", func(t *testing.T) {
		Test := TestAPI{}
		var msg string = "testmsg"

		err := Test.MsgToSlack(msg)
		if err != nil {
			t.Errorf("function 'MsgToSlack' returned an error: %w", err)
		}
		// Define want and got and compare them
		var want error = nil
		got := err
		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

}
