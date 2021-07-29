// Copyright (c) Facebook, Inc. and its affiliates.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"fmt"
	"os"

	"github.com/9elements/contest-client/contestcli"
)

// Unauthenticated, unencrypted sample HTTP client for ConTest.
// Requires the `httplistener` plugin for the API listener.
//
// Usage examples:
// List all the jobs:
//   ./contestcli list
//
// List all the failed jobs:
//   ./contestcli list -state JobStateFailed
//
// List all the failed jobs with tags "foo" and "bar":
//   ./contestcli list -state JobStateFailed -tags foo,bar

func main() {
	if err := contestcli.CLIMain(os.Args[0], os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
