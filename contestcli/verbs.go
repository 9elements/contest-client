package contestcli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/facebookincubator/contest/pkg/api"
	"github.com/facebookincubator/contest/pkg/config"
	"github.com/facebookincubator/contest/pkg/event"
	"github.com/facebookincubator/contest/pkg/job"
	"github.com/facebookincubator/contest/pkg/transport"
	"github.com/facebookincubator/contest/pkg/types"
)

func run(flags client.Flags, transport transport.Transport, stdout io.Writer) error {
	verb := strings.ToLower(flagSet.Arg(0))
	if verb == "" {
		return fmt.Errorf("missing verb, see --help")
	}
	var resp interface{}
	var err error
	switch verb {
	case "start":
		var jobDesc []byte
		if flagSet.Arg(1) == "" {
			fmt.Fprintf(os.Stderr, "Reading from stdin...\n")
			jd, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read job descriptor: %w", err)
			}
			jobDesc = jd
		} else {
			jd, err := ioutil.ReadFile(flagSet.Arg(1))
			if err != nil {
				return fmt.Errorf("failed to read job descriptor: %w", err)
			}
			jobDesc = jd
		}

		jobDescFormat := config.JobDescFormatJSON
		if *flags.FlagYAML {
			jobDescFormat = config.JobDescFormatYAML
		}
		jobDescJSON, err := config.ParseJobDescriptor(jobDesc, jobDescFormat)
		if err != nil {
			return fmt.Errorf("failed to parse job descriptor: %w", err)
		}

		startResp, err := transport.Start(context.Background(), *flags.FlagRequestor, string(jobDescJSON))
		if err != nil {
			return err
		}
		resp = startResp

		// handle wait
		if *flags.FlagWait && startResp.Data.JobID != 0 {
			// print immediately if wait is used
			buffer := &bytes.Buffer{}
			encoder := json.NewEncoder(buffer)
			encoder.SetEscapeHTML(false)
			encoder.SetIndent("", " ")
			err = encoder.Encode(startResp)
			if err != nil {
				return fmt.Errorf("cannot re-encode api.Respose object: %v", err)
			}
			indentedJSON := buffer.String()
			fmt.Fprintf(stdout, "%s", string(indentedJSON))

			fmt.Fprintf(os.Stderr, "\nWaiting for job to complete...\n")
			resp, err = wait(context.Background(), startResp.Data.JobID, jobWaitPoll, *flags.FlagRequestor, transport)

			if *flags.FlagS3 {
				buffer := &bytes.Buffer{}
				encoder := json.NewEncoder(buffer)
				encoder.SetEscapeHTML(false)
				encoder.SetIndent("", " ")
				err = encoder.Encode(resp)
				if err != nil {
					return fmt.Errorf("cannot re-encode api.Respose object: %v", err)
				}
				err = pushResultsToS3(startResp.Data.JobID, buffer.String())
				if err != nil {
					return err
				}
			}

			if err != nil {
				return err
			}
		}
	case "stop":
		jobID, err := parseJob(flagSet.Arg(1))
		if err != nil {
			return err
		}
		resp, err = transport.Stop(context.Background(), *flags.FlagRequestor, types.JobID(jobID))
		if err != nil {
			return err
		}
	case "status":
		jobID, err := parseJob(flagSet.Arg(1))
		if err != nil {
			return err
		}
		resp, err = transport.Status(context.Background(), *flags.FlagRequestor, jobID)
		if err != nil {
			return err
		}
	case "retry":
		jobID, err := parseJob(flagSet.Arg(1))
		if err != nil {
			return err
		}
		resp, err = transport.Retry(context.Background(), *flags.FlagRequestor, jobID)
		if err != nil {
			return err
		}
	case "list":
		var states []job.State
		for _, sts := range *flagStates {
			st, err := job.EventNameToJobState(event.Name(sts))
			if err != nil {
				return err
			}
			states = append(states, st)
		}
		resp, err = transport.List(context.Background(), *flags.FlagRequestor, states, *flagTags)
		if err != nil {
			return err
		}
	case "version":
		resp, err = transport.Version(context.Background(), *flags.FlagRequestor)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid verb: '%s'", verb)
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", " ")
	err = encoder.Encode(resp)
	if err != nil {
		return fmt.Errorf("cannot re-encode api.Respose object: %v", err)
	}
	stdout.Write(buffer.Bytes())
	return nil
}

func wait(ctx context.Context, jobID types.JobID, jobWaitPoll time.Duration, requestor string, transport transport.Transport) (*api.StatusResponse, error) {
	// keep polling for status till job is completed, used when -wait is set
	for {
		resp, err := transport.Status(context.Background(), requestor, jobID)
		if err != nil {
			return nil, err
		}
		if resp.Err != nil {
			return nil, fmt.Errorf("server responded with an error: %s", resp.Err)
		}

		jobState := resp.Data.Status.State

		for _, eventName := range job.JobCompletionEvents {
			if string(jobState) == string(eventName) {
				return resp, nil
			}
		}
		// TODO use  time.Ticker instead of time.Sleep
		time.Sleep(jobWaitPoll)
	}
}

func parseJob(jobIDStr string) (types.JobID, error) {
	if jobIDStr == "" {
		return 0, errors.New("missing job ID")
	}
	var jobID types.JobID
	jobIDint, err := strconv.Atoi(jobIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid job ID: %s: %v", jobIDStr, err)
	}
	jobID = types.JobID(jobIDint)
	if jobID <= 0 {
		return 0, fmt.Errorf("invalid job ID: %s: it must be positive", jobIDStr)
	}
	return jobID, nil
}