// Copyright (c) Facebook, Inc. and its affiliates.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package noop

import (
	"context"

	"github.com/9elements/contest-client/pkg/client"
	"github.com/facebookincubator/contest/pkg/transport"
)

// Name defines the name of the reporter used within the plugin registry
var Name = "noop"

// Noop is a reporter that does nothing. Probably only useful for testing.
type Noop struct{}

// ValidateRunParameters validates the parameters for the run reporter
func (n *Noop) ValidateParameters(params []byte) (interface{}, error) {
	return nil, nil
}

// Name returns the Name of the reporter
func (n *Noop) Name() string {
	return Name
}

// RunReport calculates the report to be associated with a job run.
func (n *Noop) Run(ctx context.Context, parameters interface{},
	cd client.ClientDescriptor, transport transport.Transport) (interface{}, error) {
	return nil, nil
}

// New builds a new TargetSuccessReporter
func New() client.PreJobExecutionHooks {
	return &Noop{}
}

// Load returns the name and factory which are needed to register the Reporter
func Load() (string, client.PreJobExecutionHooksFactory) {
	return Name, New
}
