// SPDX-License-Identifer: GPL-2.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/9elements/contest-client/pkg/client"
)

type versionCmd struct {
}

type startCmd struct {
	Config      string   `flag optional name:"config" help:"Path to the JSON config file." type:"path"`
	Address     string   `flag optional name:"address" help:"Address of the ConTest Server e.g. 'http://localhost'" default:"http://localhost"`
	Port        string   `flag optional name:"port" help:"ConTest Server Port to post to" default:"8080"`
	PortAPI     string   `flag optional name:"portapi" help:"Listening port of the CLI" default:"6001"`
	Requestor   string   `flag optional name:"requestor" help:"Name of the client as it will show up in the ConTest logs" default:"9e-contestcli"`
	Wait        bool     `flag optional name:"wait" help:"If wait is set, the CLI polls on the result of the ConTest server until it is finished" default:false`
	YAML        bool     `flag optional name:"ymal" help:"Specifies if the job description has been provided in .yaml format or not" default:true`
	JobWaitPoll int      `flag optional name:"jobwaitpoll" help:"Specifies the amount to wait before polling the result (again) default: 120"`
	LogLevel    string   `flag optional name:"loglevel" help:"Log leve can be: 'debug, error, panic'" default:"debug"`
	JobTemplate []string `flag optional name:"jobtemplate" help:"Specifies the jobs that should be run on each request"`
}

var cli struct {
	Debug   bool       `help:"Enable debug mode."`
	Start   startCmd   `cmd help:"Starts the listener"`
	Version versionCmd `cmd help:"Prints the version of the program"`
}

var GitCommit string

func (v *versionCmd) Run() error {
	fmt.Printf("Git Commit:\t%s\n", GitCommit)
	return nil
}

func (s *startCmd) Run() error {

	var clientConfig client.ClientDescriptor

	if s.Config != "" {
		// Open the configfile
		configFile, err := os.Open(s.Config)
		if err != nil {
			return fmt.Errorf("unable to open the config file: %w", err)
		}
		defer configFile.Close()

		// Parse and decode the json configfile
		configDescription, _ := ioutil.ReadAll(configFile)
		fmt.Printf("Configuration File: %v\n", configDescription)
		if err := json.Unmarshal(configDescription, &clientConfig); err != nil {
			return fmt.Errorf("unable to decode the config file: %w", err)
		}
	}

	fmt.Printf("Configuration: %v\n", clientConfig)

	if err := CLIMain(&clientConfig, os.Stdout); err != nil {
		return err
	}

	return nil
}
