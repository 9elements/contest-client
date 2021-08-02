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

//define flags
var (
	flagSet    *flag.FlagSet
	flagConfig *string
)

//init the flags
func initFlags(cmd string) {
	flagSet = flag.NewFlagSet(cmd, flag.ContinueOnError)
	flagConfig = flagSet.StringP("config", "c", "clientconfig.json", "Path to the configuration file that describes the client")

	// define flag usage
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
	//init the flags
	initFlags(cmd)
	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	//open the configfile
	configFile, err := os.Open(*flagConfig)
	if err != nil {
		fmt.Println(err)
	}
	defer configFile.Close()

	//parse and decode the json configfile
	configDescription, _ := ioutil.ReadAll(configFile)
	var cd client.ClientDescriptor
	if err := json.Unmarshal([]byte(configDescription), &cd); err != nil {
		fmt.Printf("unable to decode the config file with err: %v\n", err)
		fmt.Printf("content is: %+v\n", cd)
	}

	//setting defaults to empty config entries
	if *cd.Flags.FlagAddr == "" {
		*cd.Flags.FlagAddr = "http://localhost:8080"
	}
	if *cd.Flags.FlagRequestor == "" {
		*cd.Flags.FlagRequestor = "9e-contestcli"
	}
	if *cd.Flags.FlagLogLevel == "" {
		*cd.Flags.FlagLogLevel = "debug"
	}

	//create logLevel
	logLevel, err := logger.ParseLogLevel(*cd.Flags.FlagLogLevel)
	if err != nil {
		return err
	}

	//create context
	ctx, cancel := xcontext.WithCancel(logrusctx.NewContext(logLevel, logging.DefaultOptions()...))
	defer cancel()

	//initialize the plugin registry
	clientPluginRegistry := clientpluginregistry.NewClientPluginRegistry(ctx)
	clientplugins.Init(clientPluginRegistry, ctx.Logger())

	if err := cd.Validate(); err != nil {
		return err
	}

	//creating a channel with a buffer size of 10, it's big enough
	webhookData := make(chan []string, 10)

	//starting go routine to run a webhooklistener
	go webhook(webhookData)

	//go over all PreJobExecutionHooks for every webhook run it and than go over all PostJobExecutionHooks
	for nextWebhookData := range webhookData {
		for _, eh := range cd.PreJobExecutionHooks {
			//validate the current plugin
			if err := eh.PreValidate(); err != nil {
				return err
			}
			//register the current plugin
			bundlePreExecutionHook, err := clientPluginRegistry.NewPreJobExecutionHookBundle(ctx, eh)
			if err != nil {
				return err
			}
			//run the plugin
			if _, err = bundlePreExecutionHook.PreJobExecutionHooks.Run(ctx, bundlePreExecutionHook.Parameters, cd, &http.HTTP{Addr: *cd.Flags.FlagAddr}); err != nil {
				return err
			}
		}
		//run the job and receive the
		var rundata map[int][2]string
		rundata, err := run(ctx, cd, &http.HTTP{Addr: *cd.Flags.FlagAddr}, stdout, nextWebhookData)
		if err != nil {
			return nil
		}

		for _, eh := range cd.PostJobExecutionHooks {
			//validate the current plugin
			if err := eh.PostValidate(); err != nil {
				return err
			}
			//register the current plugin
			bundlePostExecutionHook, err := clientPluginRegistry.NewPostJobExecutionHookBundle(ctx, eh)
			if err != nil {
				return err
			}
			//run the plugin
			if _, err := bundlePostExecutionHook.PostJobExecutionHooks.Run(ctx, bundlePostExecutionHook.Parameters, cd, &http.HTTP{Addr: *cd.Flags.FlagAddr}, rundata); err != nil {
				return err
			}
		}
	}

	return nil
}
