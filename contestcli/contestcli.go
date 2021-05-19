package contestcli

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

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

const (
	defaultRequestor = "9e-contestcli"
	jobWaitPoll      = 30 * time.Second
)

var (
	flagSet    *flag.FlagSet
	flagConfig *string
	flagStates *[]string
	flagTags   *[]string
)

func initFlags(cmd string) {
	flagSet = flag.NewFlagSet(cmd, flag.ContinueOnError)
	flagConfig = flagSet.StringP("config", "c", "clientconfig.json", "Path to the configuration file that describes the client")

	// Flags for the "list" command.
	flagStates = flagSet.StringSlice("states", []string{}, "List of job states for the list command. A job must be in any of the specified states to match.")
	flagTags = flagSet.StringSlice("tags", []string{}, "List of tags for the list command. A job must have all the tags to match.")

	flagSet.Usage = func() {
		fmt.Fprintf(flagSet.Output(),
			`Usage:

  contestcli [flags] command

Commands:
  start [file]
        start a new job using the job description from the specified file
        or passed via stdin.
        when used with -wait flag, stdout will have two JSON outputs
        for job start and completion status separated with newline
  stop int
        stop a job by job ID
  status int
        get the status of a job by job ID
  retry int
        retry a job by job ID
  list [--states=JobStateStarted,...] [--tags=foo,...]
        list jobs by state and/or tags
  version
        request the API version to the server

Flags:
`)
		flagSet.PrintDefaults()
	}
}

func CLIMain(cmd string, args []string, stdout io.Writer) error {
	initFlags(cmd)
	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	//Open the configfile
	configFile, err := os.Open(*flagConfig)
	if err != nil {
		fmt.Println(err)
	}
	defer configFile.Close()
	//parse and decode the json file
	configDescription, _ := ioutil.ReadAll(configFile)
	var cd client.ClientDescriptor
	if err := json.Unmarshal([]byte(configDescription), &cd); err != nil {
		fmt.Printf("unable to decode the config file with err: %v\n", err)
		fmt.Printf("content is: %+v\n", cd)
	}
	if *cd.Flags.FlagAddr == "" {
		*cd.Flags.FlagAddr = "http://localhost:8080"
	}
	if *cd.Flags.FlagRequestor == "" {
		*cd.Flags.FlagRequestor = "9e-contestcli"
	}
	if *cd.Flags.FlagLogLevel == "" {
		*cd.Flags.FlagLogLevel = "debug"
	}

	logLevel, err := logger.ParseLogLevel(*cd.Flags.FlagLogLevel)
	if err != nil {
		return err
	}

	ctx, cancel := xcontext.WithCancel(logrusctx.NewContext(logLevel, logging.DefaultOptions()...))

	defer cancel()

	clientPluginRegistry := clientpluginregistry.NewClientPluginRegistry(ctx)
	clientplugins.Init(clientPluginRegistry, ctx.Logger())

	if err := cd.Validate(); err != nil {
		return err
	}
	for _, eh := range cd.PreJobExecutionHooks {
		if err := eh.PreValidate(); err != nil {
			return err
		}
		bundlePreExecutionHook, err := clientPluginRegistry.NewPreJobExecutionHookBundle(ctx, eh)
		if err != nil {
			return err
		}
		if err := doPreJobExecutionHooks(ctx, bundlePreExecutionHook); err != nil {
			return err
		}
	}

	if err := run(cd.Flags, &http.HTTP{Addr: *cd.Flags.FlagAddr}, stdout); err != nil {
		return err
	}

	for _, eh := range cd.PostJobExecutionHooks {
		if err := eh.PostValidate(); err != nil {
			return err
		}
		bundlePostExecutionHook, err := clientPluginRegistry.NewPostJobExecutionHookBundle(ctx, eh)
		if err != nil {
			return err
		}
		if err := doPostJobExecutionHooks(ctx, bundlePostExecutionHook); err != nil {
			return err
		}
	}
	return nil
}
