package pushtoS3

import (
	"context"
	"fmt"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/facebookincubator/contest/pkg/transport"
)

// Name defines the name of the preexecutionhook used within the plugin registry
var Name = "pushtoS3"

// Noop is a reporter that does nothing. Probably only useful for testing.
type PushtoS3 struct{}

// ValidateRunParameters validates the parameters for the run reporter
func (n *PushtoS3) ValidateParameters(params []byte) (interface{}, error) {
	fmt.Println("I am validating the pushtoS3 parameters")
	var s string
	return s, nil
}

// Name returns the Name of the reporter
func (n *PushtoS3) Name() string {
	return Name
}

// RunReport calculates the report to be associated with a job run.
func (n *PushtoS3) Run(ctx context.Context, parameters interface{}, cd client.ClientDescriptor, transport transport.Transport, rundata map[int][2]string) (interface{}, error) {
	fmt.Println("Started the postjobplugin PushtoS3")
	for jobid, jobdetails := range rundata {
		jobname := jobdetails[0]
		jobsha := jobdetails[1]
		go PushResultsToS3(ctx, cd, transport, jobname, jobsha, jobid)
	}
	return nil, nil
}

// New builds a new TargetSuccessReporter
func New() client.PostJobExecutionHooks {
	return &PushtoS3{}
}

// Load returns the name and factory which are needed to register the Reporter
func Load() (string, client.PostJobExecutionHooksFactory) {
	return Name, New
}
