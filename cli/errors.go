package cli

import (
	"errors"

	"github.com/tamnd/bluesky-cli/bluesky"
)

func mapFetchErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, bluesky.ErrNotFound) {
		return codeError(exitNoData, err)
	}
	if errors.Is(err, bluesky.ErrRateLimited) {
		return codeError(exitRateLimit, err)
	}
	return codeError(exitError, err)
}
