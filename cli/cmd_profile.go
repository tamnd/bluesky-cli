package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) profileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile <handle>",
		Short: "Show a Bluesky user's profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handle := args[0]
			a.progressf("fetching profile for %s...", handle)
			p, err := a.client.GetProfile(cmd.Context(), handle)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(p)
		},
	}
}
