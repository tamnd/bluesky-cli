package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) feedsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "feeds",
		Short: "List popular Bluesky feed generators",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching popular feeds...")
			feeds, err := a.client.GetPopularFeeds(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(feeds, len(feeds))
		},
	}
}
