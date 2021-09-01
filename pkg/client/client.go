package client

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/facebookincubator/contest/pkg/transport"
)

// JobExecutionHookFactory is a type representing a function which builds a JobExecutionHook object
type PreJobExecutionHooksFactory func() PreJobExecutionHooks
type PostJobExecutionHooksFactory func() PostJobExecutionHooks

// Pre/PostJobExecutionHookLoader is a type representing a function which returns all the
// needed things to be able to load a Pre/PostJobExecutionHook object
type PreJobExecutionHookLoader func() (string, PreJobExecutionHooksFactory)
type PostJobExecutionHookLoader func() (string, PostJobExecutionHooksFactory)

// ClientDescriptor models the deserialized version of the JSON text given as
// input to the client at start.
type ClientDescriptor struct {
	Flags                 Flags
	PreJobExecutionHooks  []*PreHookDescriptor
	PostJobExecutionHooks []*PostHookDescriptor
}

// ExecutionHookParam represents a ExecutionHook parameter. It is initialized from JSON,
// and can be a string or a more complex JSON structure.
// ExecutionHookPlugins are expected to know which one they expect and use the
// provided convenience functions to obtain either the string or
// json.RawMessage representation.
type ExecutionHookParam struct {
	json.RawMessage
}

// ExecutionHookParameters represents the parameters that a ExecutionHook should consume
// according to the client descriptor
type PreJobExecutionHookParameters map[string][]ExecutionHookParam
type PostJobExecutionHookParameters map[string][]ExecutionHookParam

type Flags struct {
	// Flag-related parameters
	FlagAddr        *string   //ConTest server [scheme://]host to connect to
	FlagPortServer  *string   //Port that the server is using ":port"
	FlagPortAPI     *string   //Port that the API is using ":port"
	FlagRequestor   *string   //Identifier of the requestor of the API call
	FlagWait        *bool     //After starting a job, wait for it to finish, and exit 0 only if it is successful
	FlagYAML        *bool     //JSON or YAML
	FlagS3          *bool     //Upload Job Result to S3 Bucket
	FlagjobWaitPoll *int      //Time in seconds for the interval requesting the job status
	FlagLogLevel    *string   //possible values: debug, info, warning, error, panic, fatal
	FlagJobTemplate []*string //filenames to the job templates, no default
}
type PreHookDescriptor struct {
	// PreJobExecutionHook-related parameters
	Name       string
	Parameters json.RawMessage
}

type PostHookDescriptor struct {
	// PostJobExecutionHook-related parameters
	Name       string
	Parameters json.RawMessage
}

// PreHookExecutionBundle bundles the selected PreExecutionHooks together with its parameters
type PreHookExecutionBundle struct {
	PreJobExecutionHooks PreJobExecutionHooks
	Parameters           interface{}
}

// PostHookExecutionBundle bundles the selected PreExecutionHooks together with its parameters
type PostHookExecutionBundle struct {
	PostJobExecutionHooks PostJobExecutionHooks
	Parameters            interface{}
}

// RunData cointains data that can be used to hand over data through the program flow
type RunData struct {
	JobID   int
	JobName string
	JobSHA  string
}

// PreValidate performs sanity check on the PreExecutionHookContent
func (d *PreHookDescriptor) PreValidate() error {
	if d.Name == "" {
		return errors.New("PreJobExecutionHook name cannot be empty")
	}
	return nil
}

// PostValidate performs sanity check on the PostExecutionHookContent
func (d *PostHookDescriptor) PostValidate() error {
	if d.Name == "" {
		return errors.New("PostJobExecutionHook name cannot be empty")
	}
	return nil
}

type PreJobExecutionHooks interface {
	Run(ctx context.Context, parameters interface{}, clientDescriptor ClientDescriptor,
		transport transport.Transport) (interface{}, error)
	ValidateParameters([]byte) (interface{}, error)
}

type PostJobExecutionHooks interface {
	Run(ctx context.Context, parameters interface{}, clientDescriptor ClientDescriptor,
		transport transport.Transport, rundata []RunData) (interface{}, error)
	ValidateParameters([]byte) (interface{}, error)
}
