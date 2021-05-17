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
	flagSet       *flag.FlagSet
	flagAddr      *string
	flagRequestor *string
	flagLogLevel  *string
	flagWait      *bool
	flagYAML      *bool
	flagS3        *bool
	flagStates    *[]string
	flagTags      *[]string
)

func initFlags(cmd string) {
	flagSet = flag.NewFlagSet(cmd, flag.ContinueOnError)
	flagAddr = flagSet.StringP("addr", "a", "http://localhost:8080", "ConTest server [scheme://]host:port[/basepath] to connect to")
	flagRequestor = flagSet.StringP("requestor", "r", defaultRequestor, "Identifier of the requestor of the API call")
	flagWait = flagSet.BoolP("wait", "w", false, "After starting a job, wait for it to finish, and exit 0 only if it is successful")
	flagYAML = flagSet.BoolP("yaml", "Y", true, "Parse job descriptor as YAML instead of JSON")
	flagS3 = flag.BoolP("s3", "s", false, "Upload Job Result to S3 Bucket")
	flagLogLevel = flagSet.String("logLevel", "debug", "A log level, possible values: debug, info, warning, error, panic, fatal")

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

	logLevel, err := logger.ParseLogLevel(*flagLogLevel)
	if err != nil {
		return err
	}

	ctx, cancel := xcontext.WithCancel(logrusctx.NewContext(logLevel, logging.DefaultOptions()...))

	defer cancel()

	clientPluginRegistry := clientpluginregistry.NewClientPluginRegistry(ctx)
	clientplugins.Init(clientPluginRegistry, ctx.Logger())

	//Open the configfile
	configFile, err := os.Open("clientconfig.json")
	if err != nil {
		fmt.Println(err)
	}
	defer configFile.Close()
	//parse and decode the json file
	configDescription, _ := ioutil.ReadAll(configFile)
	var cd client.ClientDescriptor
	if err := json.Unmarshal([]byte(configDescription), &cd); err != nil {
		fmt.Printf("unable to decode the config file\n")
	}

	if err := cd.Validate(); err != nil {
		return err
	}
	for _, eh := range cd.PreJobExecutionHooks {
		if err := eh.PreValidate(); err != nil {
			return err
		}
		bundlePreExecutionHook, err := clientPluginRegistry.NewPreJobExecutionHookBundle(ctx, eh)
		if err != nil {
			fmt.Errorf("could not create the bundle:\n", err)
			return err
		}
		if err := doPreJobExecutionHooks(ctx, bundlePreExecutionHook); err != nil {
			fmt.Errorf("could not execute the plugin:\n", err)
			return err
		}
	}

	if err := run(*flagRequestor, &http.HTTP{Addr: *flagAddr}, stdout); err != nil {
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
