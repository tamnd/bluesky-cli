package bluesky

import (
	"context"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes bluesky as a kit Domain. A blank import from a multi-domain
// host such as ant is enough to enable the driver:
//
//	import _ "github.com/tamnd/bluesky-cli/bluesky"
//
// The same Domain drives the standalone bsky binary (cli/root.go via kit.NewApp),
// so the binary and any host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the Bluesky driver. It carries no state; the per-run client is
// built by the factory Register hands to kit.
type Domain struct{}

// Info returns the scheme, claimed hostnames, and binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "bluesky",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "bluesky",
			Short:  "A command line for Bluesky.",
			Long: `A command line for Bluesky (bsky.app). Browse feeds, profiles, threads, and trending topics on the AT Protocol network. No API key required.

bsky is an independent tool and is not affiliated with, endorsed by, or sponsored by Bluesky Social PBC or the AT Protocol project.`,
			Site: "https://bsky.app",
			Repo: "https://github.com/tamnd/bluesky-cli",
		},
	}
}

// Register installs the client factory and all nine operations onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// feeds — popular feed generators (no argument needed)
	kit.Handle(app, kit.OpMeta{
		Name:    "feeds",
		Group:   "read",
		List:    true,
		Summary: "List popular feed generators",
		URIType: "feed",
	}, listFeeds)

	// followers — users who follow a handle
	kit.Handle(app, kit.OpMeta{
		Name:    "followers",
		Group:   "read",
		List:    true,
		Summary: "List followers of a user",
		URIType: "actor",
		Args:    []kit.Arg{{Name: "handle", Help: "handle or DID"}},
	}, listFollowers)

	// following — accounts a handle follows
	kit.Handle(app, kit.OpMeta{
		Name:    "following",
		Group:   "read",
		List:    true,
		Summary: "List accounts a user follows",
		URIType: "actor",
		Args:    []kit.Arg{{Name: "handle", Help: "handle or DID"}},
	}, listFollowing)

	// profile — single user profile
	kit.Handle(app, kit.OpMeta{
		Name:     "profile",
		Group:    "read",
		Single:   true,
		Resolver: true,
		Summary:  "Fetch a user profile",
		URIType:  "profile",
		Args:     []kit.Arg{{Name: "handle", Help: "handle or DID"}},
	}, getProfile)

	// search — search for users
	kit.Handle(app, kit.OpMeta{
		Name:    "search",
		Group:   "read",
		List:    true,
		Summary: "Search for users",
		URIType: "actor",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}},
	}, searchActors)

	// starter-packs — starter packs created by a user
	kit.Handle(app, kit.OpMeta{
		Name:    "starter-packs",
		Group:   "read",
		List:    true,
		Summary: "List starter packs by a user",
		URIType: "starter-pack",
		Args:    []kit.Arg{{Name: "handle", Help: "handle or DID"}},
	}, listStarterPacks)

	// thread — post thread by AT URI
	kit.Handle(app, kit.OpMeta{
		Name:    "thread",
		Group:   "read",
		List:    true,
		Summary: "Fetch a post thread",
		URIType: "post",
		Args:    []kit.Arg{{Name: "uri", Help: "AT URI of the root post"}},
	}, getThread)

	// trending — trending topics
	kit.Handle(app, kit.OpMeta{
		Name:    "trending",
		Group:   "read",
		List:    true,
		Summary: "List trending topics",
		URIType: "trending",
	}, listTrending)

	// user — recent posts by a user
	kit.Handle(app, kit.OpMeta{
		Name:    "user",
		Group:   "read",
		List:    true,
		Summary: "List recent posts by a user",
		URIType: "post",
		Args:    []kit.Arg{{Name: "handle", Help: "handle or DID"}},
	}, listUserPosts)
}

// newClient builds the bluesky client from the kit Config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	bc := DefaultConfig()
	if cfg.UserAgent != "" {
		bc.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		bc.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		bc.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		bc.Timeout = cfg.Timeout
	}
	return NewClient(bc), nil
}

// --- input structs ---

type handleIn struct {
	Handle string  `kit:"arg" help:"handle or DID"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type queryIn struct {
	Query  string  `kit:"arg" help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type uriIn struct {
	URI    string  `kit:"arg" help:"AT URI of the root post"`
	Client *Client `kit:"inject"`
}

type noArgIn struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listFeeds(ctx context.Context, in noArgIn, emit func(*FeedGenerator) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	feeds, err := in.Client.GetPopularFeeds(ctx, limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range feeds {
		if err := emit(&feeds[i]); err != nil {
			return err
		}
	}
	return nil
}

func listFollowers(ctx context.Context, in handleIn, emit func(*Actor) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	actors, err := in.Client.GetFollowers(ctx, in.Handle, limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range actors {
		if err := emit(&actors[i]); err != nil {
			return err
		}
	}
	return nil
}

func listFollowing(ctx context.Context, in handleIn, emit func(*Actor) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	actors, err := in.Client.GetFollows(ctx, in.Handle, limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range actors {
		if err := emit(&actors[i]); err != nil {
			return err
		}
	}
	return nil
}

func getProfile(ctx context.Context, in handleIn, emit func(*Profile) error) error {
	p, err := in.Client.GetProfile(ctx, in.Handle)
	if err != nil {
		return mapErr(err)
	}
	return emit(&p)
}

func searchActors(ctx context.Context, in queryIn, emit func(*Actor) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	actors, err := in.Client.SearchActors(ctx, in.Query, limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range actors {
		if err := emit(&actors[i]); err != nil {
			return err
		}
	}
	return nil
}

func listStarterPacks(ctx context.Context, in handleIn, emit func(*StarterPack) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	packs, err := in.Client.GetActorStarterPacks(ctx, in.Handle, limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range packs {
		if err := emit(&packs[i]); err != nil {
			return err
		}
	}
	return nil
}

func getThread(ctx context.Context, in uriIn, emit func(*ThreadPost) error) error {
	posts, err := in.Client.GetPostThread(ctx, in.URI, 6)
	if err != nil {
		return mapErr(err)
	}
	for i := range posts {
		if err := emit(&posts[i]); err != nil {
			return err
		}
	}
	return nil
}

func listTrending(ctx context.Context, in noArgIn, emit func(*TrendingTopic) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	topics, err := in.Client.GetTrendingTopics(ctx, limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range topics {
		if err := emit(&topics[i]); err != nil {
			return err
		}
	}
	return nil
}

func listUserPosts(ctx context.Context, in handleIn, emit func(*Post) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	posts, err := in.Client.GetAuthorFeed(ctx, in.Handle, limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range posts {
		if err := emit(&posts[i]); err != nil {
			return err
		}
	}
	return nil
}

// mapErr converts a library error into the appropriate kit error kind.
func mapErr(err error) error {
	switch {
	case err == ErrNotFound:
		return errs.NotFound("%s", err.Error())
	case err == ErrRateLimited:
		return errs.RateLimited("%s", err.Error())
	}
	return err
}
