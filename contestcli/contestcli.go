package contestcli

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/9elements/contest-client/pkg/client/clientpluginregistry"
	"github.com/9elements/contest-client/plugins/clientplugins"
	"github.com/facebookincubator/contest/pkg/logging"
	"github.com/facebookincubator/contest/pkg/transport/http"
	"github.com/facebookincubator/contest/pkg/xcontext"
	"github.com/facebookincubator/contest/pkg/xcontext/bundles/logrusctx"
	"github.com/facebookincubator/contest/pkg/xcontext/logger"

	flag "github.com/spf13/pflag"
)

// Define flags
var (
	flagSet    *flag.FlagSet
	flagConfig *string
)

// Init the flags
func initFlags(cmd string) {
	flagSet = flag.NewFlagSet(cmd, flag.ContinueOnError)
	flagConfig = flagSet.StringP("config", "c", "clientconfig.json", "Path to the configuration file that describes the client")

	// Define flag usage
	flagSet.Usage = func() {
		fmt.Fprintf(flagSet.Output(),
			`Usage:

  contestcli [flags] command

Commands:
  version
        request the API version to the server
Flags:
`)
		flagSet.PrintDefaults()
	}
}

func CLIMain(cmd string, args []string, stdout io.Writer) error {
	// Init the flags
	initFlags(cmd)
	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	// Open the configfile
	configFile, err := os.Open(*flagConfig)
	if err != nil {
		fmt.Println(err)
	}
	defer configFile.Close()

	// Parse and decode the json configfile
	configDescription, _ := ioutil.ReadAll(configFile)
	var cd client.ClientDescriptor
	if err := json.Unmarshal([]byte(configDescription), &cd); err != nil {
		fmt.Printf("unable to decode the config file with err: %v\n", err)
		fmt.Printf("content is: %+v\n", cd)
	}

	// Setting defaults to empty config entries
	if *cd.Flags.FlagAddr == "" {
		*cd.Flags.FlagAddr = "http://localhost"
	}
	if *cd.Flags.FlagRequestor == "" {
		*cd.Flags.FlagRequestor = "9e-contestcli"
	}
	if *cd.Flags.FlagLogLevel == "" {
		*cd.Flags.FlagLogLevel = "debug"
	}

	// Create logLevel
	logLevel, err := logger.ParseLogLevel(*cd.Flags.FlagLogLevel)
	if err != nil {
		return err
	}

	// Create context
	ctx, cancel := xcontext.WithCancel(logrusctx.NewContext(logLevel, logging.DefaultOptions()...))
	defer cancel()

	// Initialize the plugin registry
	clientPluginRegistry := clientpluginregistry.NewClientPluginRegistry(ctx)
	clientplugins.Init(clientPluginRegistry, ctx.Logger())

	if err := cd.Validate(); err != nil {
		return err
	}

	// Creating a channel with a buffer size of 10, it's big enough
	webhookData := make(chan WebhookData, 10)

	// Starting go routine to run a webhooklistener
	go webhook(webhookData)

	// Iterate over every incoming webhook
	for nextWebhookData := range webhookData {
		// Iterate over all PreJobExecution plugins
		for _, eh := range cd.PreJobExecutionHooks {
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
			if _, err = bundlePreExecutionHook.PreJobExecutionHooks.Run(ctx, bundlePreExecutionHook.Parameters, cd, &http.HTTP{Addr: *cd.Flags.FlagAddr + *cd.Flags.FlagPortServer}); err != nil {
				return err
			}
		}
		// Run the job and receive the rundata
		var rundata map[int]client.RunData
		rundata, err := run(ctx, cd, &http.HTTP{Addr: *cd.Flags.FlagAddr + *cd.Flags.FlagPortServer}, stdout, nextWebhookData)
		if err != nil {
			fmt.Println("Running the job failed. Err: %w. You should probably check the connection and restart the test.", err)
			continue
		}
		// Iterate over all PostJobExecution plugins
		for _, eh := range cd.PostJobExecutionHooks {
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
			if _, err := bundlePostExecutionHook.PostJobExecutionHooks.Run(ctx, bundlePostExecutionHook.Parameters, cd, &http.HTTP{Addr: *cd.Flags.FlagAddr + *cd.Flags.FlagPortServer}, rundata); err != nil {
				return err
			}
		}
	}

	return nil
}
