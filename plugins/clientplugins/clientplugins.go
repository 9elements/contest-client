package clientplugins

import (
	"github.com/9elements/contest-client/pkg/client"

	"github.com/9elements/contest-client/pkg/client/clientpluginregistry"
	"github.com/facebookincubator/contest/pkg/xcontext"

	"github.com/9elements/contest-client/plugins/postexecutionhooks/pushtos3"
)

var PreExecutionHooks = []client.PreJobExecutionHookLoader{}

var PostExecutionHooks = []client.PostJobExecutionHookLoader{
	pushtoS3.Load
}

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
