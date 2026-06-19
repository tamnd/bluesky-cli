// Package bluesky is the library behind the bsky command: the HTTP client,
// request shaping, and typed data models for the Bluesky AT Protocol public
// AppView API. No authentication is needed; the API at public.api.bsky.app
// is open and returns JSON for every endpoint this library uses.
//
// bsky is an independent tool and is not affiliated with, endorsed by, or
// sponsored by Bluesky Social PBC or the AT Protocol project. It reads only
// publicly accessible data at a polite rate.
package bluesky

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://public.api.bsky.app/xrpc"

// Host is the Bluesky web frontend hostname. It is used by the kit domain to
// claim ownership of bsky.app URLs pasted into a multi-domain host.
const Host = "bsky.app"

// DefaultUserAgent identifies the client to the Bluesky API.
const DefaultUserAgent = "bsky/dev (+https://github.com/tamnd/bluesky-cli)"

// Sentinel errors returned by the library.
var (
	ErrNotFound    = errors.New("not found")
	ErrRateLimited = errors.New("rate limited")
	ErrBadRequest  = errors.New("bad request")
)

// Config holds Client constructor parameters.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults for an interactive CLI.
func DefaultConfig() Config {
	return Config{
		BaseURL:   defaultBaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      50 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Bluesky public AppView API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client configured by cfg.
func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		baseURL:    base,
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// get fetches rawURL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, true, ErrRateLimited
	}
	if resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode == http.StatusBadRequest {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		// Extract the XRPC message if present.
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(b, &e)
		if e.Message != "" {
			return nil, false, fmt.Errorf("%w: %s", ErrBadRequest, e.Message)
		}
		return nil, false, ErrBadRequest
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// getJSON fetches rawURL and JSON-decodes the response into v.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(body)) == "null" {
		return ErrNotFound
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// endpoint builds a full URL for the given XRPC method and query params.
func (c *Client) endpoint(method string, params url.Values) string {
	u := c.baseURL + "/" + method
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return u
}

// ─── public methods ───────────────────────────────────────────────────────────

// GetProfile fetches the profile for a handle or DID.
func (c *Client) GetProfile(ctx context.Context, actor string) (Profile, error) {
	params := url.Values{"actor": {actor}}
	var w wireProfileView
	if err := c.getJSON(ctx, c.endpoint("app.bsky.actor.getProfile", params), &w); err != nil {
		return Profile{}, fmt.Errorf("profile %q: %w", actor, err)
	}
	return profileFromWire(w), nil
}

// GetPosts returns up to limit recent posts by actor.
func (c *Client) GetPosts(ctx context.Context, actor string, limit int) ([]Post, error) {
	return c.GetAuthorFeed(ctx, actor, limit)
}

// GetAuthorFeed returns up to limit recent posts by actor.
func (c *Client) GetAuthorFeed(ctx context.Context, actor string, limit int) ([]Post, error) {
	if limit <= 0 {
		limit = 20
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}

	var out []Post
	cursor := ""
	for {
		params := url.Values{
			"actor": {actor},
			"limit": {fmt.Sprintf("%d", pageSize)},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		var resp wireFeedResponse
		if err := c.getJSON(ctx, c.endpoint("app.bsky.feed.getAuthorFeed", params), &resp); err != nil {
			return out, fmt.Errorf("author feed %q: %w", actor, err)
		}
		for _, item := range resp.Feed {
			out = append(out, postFromWire(item.Post, len(out)+1))
			if len(out) >= limit {
				return out, nil
			}
		}
		if resp.Cursor == "" || len(resp.Feed) == 0 {
			break
		}
		cursor = resp.Cursor
	}
	return out, nil
}

// SearchActors searches for actors (users) matching q, up to limit results.
func (c *Client) SearchActors(ctx context.Context, q string, limit int) ([]Actor, error) {
	if limit <= 0 {
		limit = 20
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}

	var out []Actor
	cursor := ""
	for {
		params := url.Values{
			"q":     {q},
			"limit": {fmt.Sprintf("%d", pageSize)},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		var resp wireSearchActorsResponse
		if err := c.getJSON(ctx, c.endpoint("app.bsky.actor.searchActors", params), &resp); err != nil {
			return out, fmt.Errorf("search actors %q: %w", q, err)
		}
		for _, a := range resp.Actors {
			out = append(out, actorFromWire(a, len(out)+1))
			if len(out) >= limit {
				return out, nil
			}
		}
		if resp.Cursor == "" || len(resp.Actors) == 0 {
			break
		}
		cursor = resp.Cursor
	}
	return out, nil
}

// GetPopularFeeds returns up to limit popular feed generators.
func (c *Client) GetPopularFeeds(ctx context.Context, limit int) ([]FeedGenerator, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	var resp wirePopularFeedsResponse
	if err := c.getJSON(ctx, c.endpoint("app.bsky.unspecced.getPopularFeedGenerators", params), &resp); err != nil {
		return nil, fmt.Errorf("popular feeds: %w", err)
	}
	out := make([]FeedGenerator, 0, len(resp.Feeds))
	for i, f := range resp.Feeds {
		out = append(out, feedGenFromWire(f, i+1))
	}
	return out, nil
}

// GetTrendingTopics returns up to limit trending topics.
func (c *Client) GetTrendingTopics(ctx context.Context, limit int) ([]TrendingTopic, error) {
	if limit <= 0 {
		limit = 10
	}
	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	var resp wireTrendingResponse
	if err := c.getJSON(ctx, c.endpoint("app.bsky.unspecced.getTrendingTopics", params), &resp); err != nil {
		return nil, fmt.Errorf("trending topics: %w", err)
	}
	out := make([]TrendingTopic, 0, len(resp.Topics))
	for i, t := range resp.Topics {
		out = append(out, trendingFromWire(t, i+1))
	}
	return out, nil
}

// GetPostThread returns the thread rooted at uri, flattened in DFS order.
// depth controls how many reply levels are fetched (0 = root post only).
func (c *Client) GetPostThread(ctx context.Context, uri string, depth int) ([]ThreadPost, error) {
	params := url.Values{
		"uri":   {uri},
		"depth": {fmt.Sprintf("%d", depth)},
	}
	var resp wireThreadResponse
	if err := c.getJSON(ctx, c.endpoint("app.bsky.feed.getPostThread", params), &resp); err != nil {
		return nil, fmt.Errorf("thread %q: %w", uri, err)
	}
	var out []ThreadPost
	walkThread(&resp.Thread, 0, &out)
	return out, nil
}

func walkThread(node *wireThreadNode, depth int, out *[]ThreadPost) {
	if node == nil {
		return
	}
	// Skip blocked or not-found thread nodes.
	if node.Type == "app.bsky.feed.defs#blockedPost" ||
		node.Type == "app.bsky.feed.defs#notFoundPost" {
		return
	}
	*out = append(*out, threadPostFromWire(node.Post, depth))
	for i := range node.Replies {
		walkThread(&node.Replies[i], depth+1, out)
	}
}

// GetFollowers returns up to limit followers of actor.
func (c *Client) GetFollowers(ctx context.Context, actor string, limit int) ([]Actor, error) {
	if limit <= 0 {
		limit = 20
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}

	var out []Actor
	cursor := ""
	for {
		params := url.Values{
			"actor": {actor},
			"limit": {fmt.Sprintf("%d", pageSize)},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		var resp wireFollowersResponse
		if err := c.getJSON(ctx, c.endpoint("app.bsky.graph.getFollowers", params), &resp); err != nil {
			return out, fmt.Errorf("followers %q: %w", actor, err)
		}
		for _, a := range resp.Followers {
			out = append(out, actorFromWire(a, len(out)+1))
			if len(out) >= limit {
				return out, nil
			}
		}
		if resp.Cursor == "" || len(resp.Followers) == 0 {
			break
		}
		cursor = resp.Cursor
	}
	return out, nil
}

// GetFollows returns up to limit accounts that actor follows.
func (c *Client) GetFollows(ctx context.Context, actor string, limit int) ([]Actor, error) {
	if limit <= 0 {
		limit = 20
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}

	var out []Actor
	cursor := ""
	for {
		params := url.Values{
			"actor": {actor},
			"limit": {fmt.Sprintf("%d", pageSize)},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		var resp wireFollowsResponse
		if err := c.getJSON(ctx, c.endpoint("app.bsky.graph.getFollows", params), &resp); err != nil {
			return out, fmt.Errorf("following %q: %w", actor, err)
		}
		for _, a := range resp.Follows {
			out = append(out, actorFromWire(a, len(out)+1))
			if len(out) >= limit {
				return out, nil
			}
		}
		if resp.Cursor == "" || len(resp.Follows) == 0 {
			break
		}
		cursor = resp.Cursor
	}
	return out, nil
}

// GetActorStarterPacks returns up to limit starter packs created by actor.
func (c *Client) GetActorStarterPacks(ctx context.Context, actor string, limit int) ([]StarterPack, error) {
	if limit <= 0 {
		limit = 20
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}

	var out []StarterPack
	cursor := ""
	for {
		params := url.Values{
			"actor": {actor},
			"limit": {fmt.Sprintf("%d", pageSize)},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		var resp wireStarterPacksResponse
		if err := c.getJSON(ctx, c.endpoint("app.bsky.graph.getActorStarterPacks", params), &resp); err != nil {
			return out, fmt.Errorf("starter packs %q: %w", actor, err)
		}
		for _, s := range resp.StarterPacks {
			out = append(out, starterPackFromWire(s, len(out)+1))
			if len(out) >= limit {
				return out, nil
			}
		}
		if resp.Cursor == "" || len(resp.StarterPacks) == 0 {
			break
		}
		cursor = resp.Cursor
	}
	return out, nil
}
