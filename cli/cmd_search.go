package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search Bluesky users",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := args[0]
			n := a.effectiveLimit(20)
			a.progressf("searching users for %q...", q)
			actors, err := a.client.SearchActors(cmd.Context(), q, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(actors, len(actors))
		},
	}
}
