package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) trendingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trending",
		Short: "Show trending topics on Bluesky",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching trending topics...")
			topics, err := a.client.GetTrendingTopics(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(topics, len(topics))
		},
	}
}
