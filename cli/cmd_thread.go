package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) threadCmd() *cobra.Command {
	var depth int
	cmd := &cobra.Command{
		Use:   "thread <at-uri>",
		Short: "Show a post thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uri := args[0]
			a.progressf("fetching thread %s...", uri)
			posts, err := a.client.GetPostThread(cmd.Context(), uri, depth)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(posts, len(posts))
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 6, "maximum reply depth to traverse (0 = root only)")
	return cmd
}
