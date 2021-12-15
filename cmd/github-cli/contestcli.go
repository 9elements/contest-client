package main

import (
	"fmt"
	"io"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/9elements/contest-client/pkg/client/clientpluginregistry"
	"github.com/9elements/contest-client/pkg/webhook"
	"github.com/9elements/contest-client/plugins/clientplugins"
	"github.com/facebookincubator/contest/pkg/logging"
	"github.com/facebookincubator/contest/pkg/transport/http"
	"github.com/facebookincubator/contest/pkg/xcontext"
	"github.com/facebookincubator/contest/pkg/xcontext/bundles/logrusctx"
)

func CLIMain(config *client.ClientDescriptor, stdout io.Writer) error {

	// Create context
	ctx, cancel := xcontext.WithCancel(logrusctx.NewContext(6, logging.DefaultOptions()...))
	defer cancel()

	// Initialize the plugin registry
	clientPluginRegistry := clientpluginregistry.NewClientPluginRegistry(ctx)
	clientplugins.Init(clientPluginRegistry, ctx.Logger())

	// Init and Start the webhook listener
	var weblistener webhook.Webhook
	weblistener.NewListener()

	go weblistener.Start()

	// Iterate over every incoming webhook
	for nextWebhookData := range weblistener.Data {
		// Iterate over all PreJobExecution plugins
		for _, eh := range config.PreJobExecutionHooks {
			// Validate the current plugin
			if err := eh.PreValidate(); err != nil {
				return err
			}
			// Register the current plugin
			bundlePreExecutionHook, err := clientPluginRegistry.NewPreJobExecutionHookBundle(ctx, eh)
			if err != nil {
				return err
			}
			// Run the plugin
			if _, err = bundlePreExecutionHook.PreJobExecutionHooks.Run(ctx, bundlePreExecutionHook.Parameters, *config, &http.HTTP{Addr: *config.Configuration.Addr + *config.Configuration.PortServer}); err != nil {
				return err
			}
		}
		// Run the job and receive the rundata
		var rundata []client.RunData
		rundata, err := run(ctx, *config, &http.HTTP{Addr: *config.Configuration.Addr + *config.Configuration.PortServer}, stdout, nextWebhookData, weblistener)
		if err != nil {
			_ = fmt.Errorf("running the job failed (err: %w) You should probably check the connection and restart the test", err)
			continue
		}
		// Iterate over all PostJobExecution plugins
		for _, eh := range config.PostJobExecutionHooks {
			// Validate the current plugin
			if err := eh.PostValidate(); err != nil {
				return err
			}
			// Register the current plugin
			bundlePostExecutionHook, err := clientPluginRegistry.NewPostJobExecutionHookBundle(ctx, eh)
			if err != nil {
				return err
			}
			// Run the plugin
			if _, err := bundlePostExecutionHook.PostJobExecutionHooks.Run(ctx, bundlePostExecutionHook.Parameters, *config, &http.HTTP{Addr: *config.Configuration.Addr + *config.Configuration.PortServer}, rundata); err != nil {
				return err
			}
		}
	}

	return nil
}
