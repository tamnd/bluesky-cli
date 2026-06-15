package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) postsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "posts <handle>",
		Short: "Show a user's recent posts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handle := args[0]
			n := a.effectiveLimit(20)
			a.progressf("fetching posts for %s...", handle)
			posts, err := a.client.GetPosts(cmd.Context(), handle, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(posts, len(posts))
		},
	}
}
