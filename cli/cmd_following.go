package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) followingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "following <handle>",
		Short: "List the accounts a user follows",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handle := args[0]
			n := a.effectiveLimit(20)
			a.progressf("fetching following for %s...", handle)
			actors, err := a.client.GetFollows(cmd.Context(), handle, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(actors, len(actors))
		},
	}
}
