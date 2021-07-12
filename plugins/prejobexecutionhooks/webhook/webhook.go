package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/facebookincubator/contest/pkg/transport"
)

const templateamount = 5

// Name defines the name of the preexecutionhook used within the plugin registry
var Name = "webhook"

//define a maximum of different targettemplates
var Targettemplates [templateamount]string

// webhook is a github webhook fetcher to automate testing new pullrequests
type Webhook struct{}

type webhookparameter struct {
	Templates []string
	URI       string
}

// ValidateRunParameters validates the parameters for the run reporter
func (n *Webhook) ValidateParameters(params []byte) (interface{}, error) {
	var Wp webhookparameter
	if err := json.Unmarshal(params, &Wp); err != nil {
		return nil, fmt.Errorf("could not validate webhook parameters")
	}
	for m, n := range Wp.Templates {
		Targettemplates[m] = n
	}
	return nil, nil
}

// Name returns the Name of the reporter
func (n *Webhook) Name() string {
	return Name
}

// RunReport calculates the report to be associated with a job run.
func (n *Webhook) Run(ctx context.Context, parameters interface{}, cd client.ClientDescriptor, transport transport.Transport, webhookData []string) (interface{}, error) {
	log.Println("webhook plugin did nothing")
	return nil, nil
}

// New builds a new webhook
func New() client.PreJobExecutionHooks {
	return &Webhook{}
}

// Load returns the name and factory which are needed to register the Reporter
func Load() (string, client.PreJobExecutionHooksFactory) {
	return Name, New
}
