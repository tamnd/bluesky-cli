package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) starterPacksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "starter-packs <handle>",
		Short: "List starter packs created by a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handle := args[0]
			n := a.effectiveLimit(20)
			a.progressf("fetching starter packs for %s...", handle)
			packs, err := a.client.GetActorStarterPacks(cmd.Context(), handle, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(packs, len(packs))
		},
	}
}
