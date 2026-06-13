package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) followersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "followers <handle>",
		Short: "List a user's followers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handle := args[0]
			n := a.effectiveLimit(20)
			a.progressf("fetching followers for %s...", handle)
			actors, err := a.client.GetFollowers(cmd.Context(), handle, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(actors, len(actors))
		},
	}
}
