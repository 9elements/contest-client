package client

import (
	"context"
	"encoding/json"
	"errors"
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

// Validate performs sanity check on the ExecutionHookDescriptor
func (d *ClientDescriptor) Validate() error {
	if len(d.PreJobExecutionHooks) == 0 {
		return errors.New("PreJobExecutionHook cannot be empty")
	}
	if len(d.PostJobExecutionHooks) == 0 {
		return errors.New("PostJobExecutionHook cannot be empty")
	}
	return nil
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
	Run(ctx context.Context, parameters interface{}) (interface{}, error)
	ValidateParameters([]byte) (interface{}, error)
}

type PostJobExecutionHooks interface {
	Run(ctx context.Context, parameters interface{}) (interface{}, error)
	ValidateParameters([]byte) (interface{}, error)
}
