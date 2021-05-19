package test

import (
	"context"
	"fmt"

	"github.com/9elements/contest-client/pkg/client"
)

// Name defines the name of the preexecutionhook used within the plugin registry
var Name = "test"

// Noop is a reporter that does nothing. Probably only useful for testing.
type Test struct{}

// ValidateRunParameters validates the parameters for the run reporter
func (n *Test) ValidateParameters(params []byte) (interface{}, error) {
	fmt.Println("I am validating the test parameters")
	var s string
	return s, nil
}

// Name returns the Name of the reporter
func (n *Test) Name() string {
	return Name
}

// RunReport calculates the report to be associated with a job run.
func (n *Test) Run(ctx context.Context, parameters interface{}) (interface{}, error) {
	fmt.Println("I am running the test plugin")
	return "I did nothing", nil
}

// New builds a new TargetSuccessReporter
func New() client.PreJobExecutionHooks {
	return &Test{}
}

// Load returns the name and factory which are needed to register the Reporter
func Load() (string, client.PreJobExecutionHooksFactory) {
	return Name, New
}
