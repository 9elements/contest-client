package clientpluginregistry

import (
	"fmt"
	"strings"
	"sync"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/facebookincubator/contest/pkg/xcontext"
)

// ClientPluginRegistry manages all the plugins available in the system. It associates Client Plugin
// identifiers (implemented as simple strings) with factory functions that create instances
// of those plugins. A Client Plugin instance is owner by a single Job object.
type ClientPluginRegistry struct {
	lock sync.RWMutex

	Context xcontext.Context

	// PreJobExecutionHooks are hooks which gets executed before posting the job to the server
	PreJobExecutionHooks map[string]client.PreJobExecutionHooksFactory

	// PostJobExecutionHooks are hooks which gets executed after the job has been processed(!) by the server
	PostJobExecutionHooks map[string]client.PostJobExecutionHooksFactory
}

func NewClientPluginRegistry(ctx xcontext.Context) *ClientPluginRegistry {
	pr := ClientPluginRegistry{
		Context: ctx,
	}
	pr.PreJobExecutionHooks = make(map[string]client.PreJobExecutionHooksFactory)
	pr.PostJobExecutionHooks = make(map[string]client.PostJobExecutionHooksFactory)

	return &pr
}

// RegisterPreJobExecutionHook register a factory for PreJobExecutionHook plugins
func (r *ClientPluginRegistry) RegisterPreJobExecutionHook(pluginName string, pjf client.PreJobExecutionHooksFactory) error {
	pluginName = strings.ToLower(pluginName)
	r.lock.Lock()
	defer r.lock.Unlock()
	r.Context.Infof("Registering PreJobExecutionHook %s", pluginName)
	if _, found := r.PreJobExecutionHooks[pluginName]; found {
		return fmt.Errorf("PreJobExecutionHook %s already registered", pluginName)
	}
	r.PreJobExecutionHooks[pluginName] = pjf
	return nil
}

// NewPreJobExecutionHook returns a new instance of PreJobExecutionHook from its
// corresponding name
func (r *ClientPluginRegistry) NewPreJobExecutionHook(pluginName string) (client.PreJobExecutionHooks, error) {
	pluginName = strings.ToLower(pluginName)
	var (
		preJobExecutionHookFactory client.PreJobExecutionHooksFactory
		found                      bool
	)
	r.lock.RLock()
	preJobExecutionHookFactory, found = r.PreJobExecutionHooks[pluginName]
	r.lock.RUnlock()
	if !found {
		return nil, fmt.Errorf("preHookExecutionHook %v is not registered", pluginName)
	}
	preJobExecutionHook := preJobExecutionHookFactory()
	return preJobExecutionHook, nil
}

// RegisterPostJobExecutionHook register a factory for PostJobExecutionHook plugins
func (r *ClientPluginRegistry) RegisterPostJobExecutionHook(pluginName string, pjf client.PostJobExecutionHooksFactory) error {
	pluginName = strings.ToLower(pluginName)
	r.lock.Lock()
	defer r.lock.Unlock()
	r.Context.Infof("Registering PostJobExecutionHook %s", pluginName)
	if _, found := r.PostJobExecutionHooks[pluginName]; found {
		return fmt.Errorf("PostJobExecutionHook %s already registered", pluginName)
	}
	r.PostJobExecutionHooks[pluginName] = pjf
	return nil
}

// NewPostJobExecutionHook returns a new instance of PostJobExecutionHook from its
// corresponding name
func (r *ClientPluginRegistry) NewPostJobExecutionHook(pluginName string) (client.PostJobExecutionHooks, error) {
	pluginName = strings.ToLower(pluginName)
	var (
		postJobExecutionHookFactory client.PostJobExecutionHooksFactory
		found                       bool
	)
	r.lock.RLock()
	postJobExecutionHookFactory, found = r.PostJobExecutionHooks[pluginName]
	r.lock.RUnlock()
	if !found {
		return nil, fmt.Errorf("postHookExecutionHook %v is not registered", pluginName)
	}
	postJobExecutionHook := postJobExecutionHookFactory()
	return postJobExecutionHook, nil
}
