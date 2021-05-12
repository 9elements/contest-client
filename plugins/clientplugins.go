package clientplugins

import (
	"sync"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/9elements/contest-client/pkg/clientpluginregistry"
	"github.com/facebookincubator/contest/pkg/xcontext"
)

var PreExecutionHooks = []client.PreJobExecutionHookLoader{}

var PostExecutionHooks = []client.PostJobExecutionHookLoader{}

var testInitOnce sync.Once

// Init initializes the client plugin registry
func Init(clientPluginRegistry *clientpluginregistry.ClientPluginRegistry, log xcontext.Logger) {

	// Register PreJobExecutionHook plugins
	for _, preloader := range PreExecutionHooks {
		if err := clientPluginRegistry.RegisterPreJobExecutionHook(preloader()); err != nil {
			log.Fatalf("%v", err)
		}
	}

	// Register PostJobExecutionHook plugins
	for _, postloader := range PostExecutionHooks {
		if err := clientPluginRegistry.RegisterPostJobExecutionHook(postloader()); err != nil {
			log.Fatalf("%v", err)
		}
	}
}
