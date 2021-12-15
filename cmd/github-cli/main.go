// SPDX-License-Identifer: GPL-2.0-or-later

package main

import (
	"github.com/alecthomas/kong"
)

const (
	programName = "ConTest Github Listener"
	programDesc = "HTTP Listener on Github webhooks for ConTest"
)

var (
	gitcommit string
	gittag    string
)

func main() {
	ctx := kong.Parse(&cli,
		kong.Name(programName),
		kong.Description(programDesc),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
