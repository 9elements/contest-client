package pushtoS3

import (
	"context"

	"github.com/9elements/contest-client/pkg/client"
)

// Name defines the name of the preexecutionhook used within the plugin registry
var Name = "pushtoS3"

// Noop is a reporter that does nothing. Probably only useful for testing.
type PushtoS3 struct{}

// ValidateRunParameters validates the parameters for the run reporter
func (n *PushtoS3) ValidateParameters(params []byte) (interface{}, error) {
	var s string
	return s, nil
}

// Name returns the Name of the reporter
func (n *PushtoS3) Name() string {
	return Name
}

// RunReport calculates the report to be associated with a job run.
func (n *PushtoS3) Run(ctx context.Context, parameters interface{}) (interface{}, error) {
	return "I did nothing", nil
}

// New builds a new TargetSuccessReporter
func New() client.PreJobExecutionHooks {
	return &PushtoS3{}
}

// Load returns the name and factory which are needed to register the Reporter
func Load() (string, client.PreJobExecutionHooksFactory) {
	return Name, New
}
