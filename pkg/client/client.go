package client

import (
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
	PreJobExecutionHookName       string
	PreJobExecutionHookParameters json.RawMessage
}

type PostHookDescriptor struct {
	// PostJobExecutionHook-related parameters
	PostJobExecutionHookName       string
	PostJobExecutionHookParameters json.RawMessage
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

// Validate performs sanity check on the PreHookDescriptor
func (d *PreHookDescriptor) Validate() error {
	if d.PreJobExecutionHookName == "" {
		return errors.New("PreJobExecutionHook name cannot be empty")
	}
	return nil
}

// Validate performs sanity check on the PostHookDescriptor
func (d *PostHookDescriptor) Validate() error {
	if d.PostJobExecutionHookName == "" {
		return errors.New("PostJobExecutionHook name cannot be empty")
	}
	return nil
}

type PreJobExecutionHooks interface {
	Run([]byte) (interface{}, error)
	ValidateParameters([]byte) (interface{}, error)
}

type PostJobExecutionHooks interface {
	Run([]byte) (interface{}, error)
	ValidateParameters([]byte) (interface{}, error)
}
