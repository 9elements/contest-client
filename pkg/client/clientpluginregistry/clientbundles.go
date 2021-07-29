package clientpluginregistry

import (
	"fmt"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/facebookincubator/contest/pkg/xcontext"
)

// NewPreJobExecutionHookBundle creates a PreJobExecutionHook and associated parameters based on
// the content of the client prehook descriptor
func (r *ClientPluginRegistry) NewPreJobExecutionHookBundle(ctx xcontext.Context, preHookDescriptor *client.PreHookDescriptor) (
	*client.PreHookExecutionBundle, error) {

	// Initialization and validation of the PreExecutionHook and its parameters
	if preHookDescriptor == nil {
		return nil, fmt.Errorf("pre execution hook description is null")
	}
	preJobExecutionHook, err := r.NewPreJobExecutionHook(preHookDescriptor.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get the desired PreExecutionHook (%s): %v", preHookDescriptor.Name, err)
	}
	// FetchParameters
	fp, err := preJobExecutionHook.ValidateParameters(preHookDescriptor.Parameters)
	if err != nil {
		return nil, fmt.Errorf("could not validate PreExecutionHook fetch parameters: %v", err)
	}
	preHookExecutionBundle := client.PreHookExecutionBundle{
		PreJobExecutionHooks: preJobExecutionHook,
		Parameters:           fp,
	}
	return &preHookExecutionBundle, nil
}

// NewPostJobExecutionHookBundle creates a PostJobExecutionHook and associated parameters based on
// the content of the client prehook descriptor
func (r *ClientPluginRegistry) NewPostJobExecutionHookBundle(ctx xcontext.Context, postHookDescriptor *client.PostHookDescriptor) (
	*client.PostHookExecutionBundle, error) {

	// Initialization and validation of the PostExecutionHook and its parameters
	if postHookDescriptor == nil {
		return nil, fmt.Errorf("post execution hook description is null")
	}
	postJobExecutionHook, err := r.NewPostJobExecutionHook(postHookDescriptor.Name)
	if err != nil {
		return nil, fmt.Errorf("could not get the desired PostExecutionHook (%s): %v", postHookDescriptor.Name, err)
	}
	// FetchParameters
	fp, err := postJobExecutionHook.ValidateParameters(postHookDescriptor.Parameters)
	if err != nil {
		return nil, fmt.Errorf("could not validate PostExecutionHook fetch parameters: %v", err)
	}
	postHookExecutionBundle := client.PostHookExecutionBundle{
		PostJobExecutionHooks: postJobExecutionHook,
		Parameters:            fp,
	}
	return &postHookExecutionBundle, nil
}
