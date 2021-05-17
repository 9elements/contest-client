package contestcli

import (
	"github.com/9elements/contest-client/pkg/client"

	"github.com/facebookincubator/contest/pkg/xcontext"
)

func doPreJobExecutionHooks(ctx xcontext.Context, preHookBundle *client.PreHookExecutionBundle) error {
	if _, err := preHookBundle.PreJobExecutionHooks.Run(ctx, preHookBundle.Parameters); err != nil {
		return err
	}
	return nil
}

func doPostJobExecutionHooks(ctx xcontext.Context, postHookBundle *client.PostHookExecutionBundle) error {
	if _, err := postHookBundle.PostJobExecutionHooks.Run(ctx, postHookBundle.Parameters); err != nil {
		return err
	}
	return nil
}
