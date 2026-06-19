package bluesky

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(baseURL string) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = baseURL
	cfg.Rate = 0
	return NewClient(cfg)
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	_, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 5
	cfg.BaseURL = srv.URL
	c := NewClient(cfg)

	_, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestGetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	_, err := c.get(context.Background(), srv.URL)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	cfg.Retries = 1
	cfg.BaseURL = srv.URL
	c := NewClient(cfg)

	_, err := c.GetProfile(context.Background(), "anyone")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("got %v, want ErrRateLimited", err)
	}
}

func TestBadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"InvalidRequest","message":"actor not valid"}`))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	_, err := c.GetProfile(context.Background(), "bad-handle")
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("got %v, want ErrBadRequest", err)
	}
}

func TestGetProfile(t *testing.T) {
	body := `{
		"handle": "alice.bsky.social",
		"displayName": "Alice",
		"did": "did:plc:abc123",
		"description": "test account",
		"followersCount": 42,
		"followsCount": 10,
		"postsCount": 5,
		"indexedAt": "2024-01-01T00:00:00Z"
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	p, err := c.GetProfile(context.Background(), "alice.bsky.social")
	if err != nil {
		t.Fatal(err)
	}
	if p.Handle != "alice.bsky.social" {
		t.Errorf("handle = %q, want alice.bsky.social", p.Handle)
	}
	if p.DisplayName != "Alice" {
		t.Errorf("display_name = %q, want Alice", p.DisplayName)
	}
	if p.FollowersCount != 42 {
		t.Errorf("followers_count = %d, want 42", p.FollowersCount)
	}
	if p.PostsCount != 5 {
		t.Errorf("posts_count = %d, want 5", p.PostsCount)
	}
	if p.URL != "https://bsky.app/profile/alice.bsky.social" {
		t.Errorf("url = %q, want https://bsky.app/profile/alice.bsky.social", p.URL)
	}
}

func TestGetAuthorFeed(t *testing.T) {
	body := `{
		"feed": [
			{
				"post": {
					"uri": "at://did:plc:abc/app.bsky.feed.post/rkey1",
					"author": {"handle": "alice.bsky.social", "did": "did:plc:abc", "displayName": "Alice"},
					"record": {"text": "hello world"},
					"likeCount": 5,
					"repostCount": 2,
					"replyCount": 1,
					"indexedAt": "2024-01-01T00:00:00Z"
				}
			}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	posts, err := c.GetAuthorFeed(context.Background(), "alice.bsky.social", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("got %d posts, want 1", len(posts))
	}
	if posts[0].Text != "hello world" {
		t.Errorf("text = %q, want 'hello world'", posts[0].Text)
	}
	if posts[0].LikeCount != 5 {
		t.Errorf("like_count = %d, want 5", posts[0].LikeCount)
	}
	if posts[0].Rank != 1 {
		t.Errorf("rank = %d, want 1", posts[0].Rank)
	}
	if posts[0].Author != "alice.bsky.social" {
		t.Errorf("author = %q, want alice.bsky.social", posts[0].Author)
	}
	if posts[0].AuthorName != "Alice" {
		t.Errorf("author_name = %q, want Alice", posts[0].AuthorName)
	}
	if posts[0].URL != "https://bsky.app/profile/alice.bsky.social/post/rkey1" {
		t.Errorf("url = %q, unexpected", posts[0].URL)
	}
}

func TestGetPostsLimit(t *testing.T) {
	body := `{
		"feed": [
			{"post": {"uri": "at://did:plc:abc/app.bsky.feed.post/r1", "author": {"handle": "a"}, "record": {"text": "one"}, "indexedAt": "2024-01-01T00:00:00Z"}},
			{"post": {"uri": "at://did:plc:abc/app.bsky.feed.post/r2", "author": {"handle": "a"}, "record": {"text": "two"}, "indexedAt": "2024-01-01T00:00:00Z"}},
			{"post": {"uri": "at://did:plc:abc/app.bsky.feed.post/r3", "author": {"handle": "a"}, "record": {"text": "three"}, "indexedAt": "2024-01-01T00:00:00Z"}}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	posts, err := c.GetPosts(context.Background(), "a", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Errorf("got %d posts with limit=2, want 2", len(posts))
	}
}

func TestGetPostsPagination(t *testing.T) {
	page1 := `{
		"feed": [
			{"post": {"uri": "at://did:plc:x/app.bsky.feed.post/p1", "author": {"handle": "a"}, "record": {"text": "post1"}, "indexedAt": "2024-01-01T00:00:00Z"}},
			{"post": {"uri": "at://did:plc:x/app.bsky.feed.post/p2", "author": {"handle": "a"}, "record": {"text": "post2"}, "indexedAt": "2024-01-01T00:00:00Z"}}
		],
		"cursor": "page2cursor"
	}`
	page2 := `{
		"feed": [
			{"post": {"uri": "at://did:plc:x/app.bsky.feed.post/p3", "author": {"handle": "a"}, "record": {"text": "post3"}, "indexedAt": "2024-01-01T00:00:00Z"}}
		]
	}`
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("cursor") == "page2cursor" {
			_, _ = w.Write([]byte(page2))
		} else {
			_, _ = w.Write([]byte(page1))
		}
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	posts, err := c.GetPosts(context.Background(), "a", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 3 {
		t.Errorf("got %d posts across 2 pages, want 3", len(posts))
	}
	if calls != 2 {
		t.Errorf("made %d API calls, want 2", calls)
	}
}

func TestSearchActors(t *testing.T) {
	body := `{
		"actors": [
			{"handle": "alice.bsky.social", "displayName": "Alice", "did": "did:plc:a", "followersCount": 100},
			{"handle": "bob.bsky.social", "displayName": "Bob", "did": "did:plc:b", "followersCount": 200}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	actors, err := c.SearchActors(context.Background(), "alice", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(actors) != 2 {
		t.Fatalf("got %d actors, want 2", len(actors))
	}
	if actors[0].Handle != "alice.bsky.social" {
		t.Errorf("handle = %q, want alice.bsky.social", actors[0].Handle)
	}
	if actors[0].Followers != 100 {
		t.Errorf("followers = %d, want 100", actors[0].Followers)
	}
	if actors[1].Rank != 2 {
		t.Errorf("rank = %d, want 2", actors[1].Rank)
	}
	if actors[0].URL != "https://bsky.app/profile/alice.bsky.social" {
		t.Errorf("url = %q, unexpected", actors[0].URL)
	}
}

func TestGetPopularFeeds(t *testing.T) {
	body := `{
		"feeds": [
			{
				"uri": "at://did:plc:xyz/app.bsky.feed.generator/whats-hot",
				"displayName": "What's Hot",
				"description": "Popular posts",
				"creator": {"handle": "bsky.app"},
				"likeCount": 9999
			}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	feeds, err := c.GetPopularFeeds(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(feeds) != 1 {
		t.Fatalf("got %d feeds, want 1", len(feeds))
	}
	if feeds[0].DisplayName != "What's Hot" {
		t.Errorf("display_name = %q, want What's Hot", feeds[0].DisplayName)
	}
	if feeds[0].Creator != "bsky.app" {
		t.Errorf("creator = %q, want bsky.app", feeds[0].Creator)
	}
	if feeds[0].LikeCount != 9999 {
		t.Errorf("like_count = %d, want 9999", feeds[0].LikeCount)
	}
	if feeds[0].URL != "https://bsky.app/profile/bsky.app/feed/whats-hot" {
		t.Errorf("url = %q, unexpected", feeds[0].URL)
	}
}

func TestGetTrendingTopics(t *testing.T) {
	body := `{
		"topics": [
			{"topic": "golang", "link": "https://bsky.app/hashtag/golang"},
			{"topic": "ai", "link": "https://bsky.app/hashtag/ai"}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	topics, err := c.GetTrendingTopics(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 2 {
		t.Fatalf("got %d topics, want 2", len(topics))
	}
	if topics[0].Topic != "golang" {
		t.Errorf("topic = %q, want golang", topics[0].Topic)
	}
	if topics[0].Rank != 1 {
		t.Errorf("rank = %d, want 1", topics[0].Rank)
	}
	if topics[1].Link != "https://bsky.app/hashtag/ai" {
		t.Errorf("link = %q, unexpected", topics[1].Link)
	}
}

func TestGetThread(t *testing.T) {
	body := `{
		"thread": {
			"$type": "app.bsky.feed.defs#threadViewPost",
			"post": {
				"uri": "at://did:plc:xxx/app.bsky.feed.post/root",
				"author": {"handle": "alice.bsky.social"},
				"record": {"text": "Root post"},
				"likeCount": 100,
				"repostCount": 10,
				"replyCount": 2,
				"indexedAt": "2024-01-15T12:00:01.000Z"
			},
			"replies": [
				{
					"$type": "app.bsky.feed.defs#threadViewPost",
					"post": {
						"uri": "at://did:plc:yyy/app.bsky.feed.post/reply1",
						"author": {"handle": "bob.bsky.social"},
						"record": {"text": "Reply to root"},
						"likeCount": 5,
						"repostCount": 0,
						"replyCount": 0,
						"indexedAt": "2024-01-15T12:01:01.000Z"
					},
					"replies": []
				},
				{
					"$type": "app.bsky.feed.defs#blockedPost",
					"uri": "at://did:plc:blocked/app.bsky.feed.post/blocked",
					"blocked": true
				}
			]
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	posts, err := c.GetPostThread(context.Background(), "at://did:plc:xxx/app.bsky.feed.post/root", 6)
	if err != nil {
		t.Fatal(err)
	}
	// blockedPost must be skipped, so only 2 posts (root + one reply)
	if len(posts) != 2 {
		t.Fatalf("got %d thread posts, want 2 (blocked post should be skipped)", len(posts))
	}
	if posts[0].Depth != 0 {
		t.Errorf("root depth = %d, want 0", posts[0].Depth)
	}
	if posts[0].Author != "alice.bsky.social" {
		t.Errorf("root author = %q, want alice.bsky.social", posts[0].Author)
	}
	if posts[0].Text != "Root post" {
		t.Errorf("root text = %q, want 'Root post'", posts[0].Text)
	}
	if posts[1].Depth != 1 {
		t.Errorf("reply depth = %d, want 1", posts[1].Depth)
	}
	if posts[1].Author != "bob.bsky.social" {
		t.Errorf("reply author = %q, want bob.bsky.social", posts[1].Author)
	}
}

func TestGetFollowers(t *testing.T) {
	body := `{
		"followers": [
			{"handle": "bob.bsky.social", "displayName": "Bob", "did": "did:plc:b", "followersCount": 50}
		],
		"cursor": ""
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	followers, err := c.GetFollowers(context.Background(), "alice.bsky.social", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(followers) != 1 {
		t.Fatalf("got %d followers, want 1", len(followers))
	}
	if followers[0].Handle != "bob.bsky.social" {
		t.Errorf("handle = %q, want bob.bsky.social", followers[0].Handle)
	}
	if followers[0].Rank != 1 {
		t.Errorf("rank = %d, want 1", followers[0].Rank)
	}
}

func TestGetFollowing(t *testing.T) {
	body := `{
		"follows": [
			{"handle": "carol.bsky.social", "displayName": "Carol", "did": "did:plc:c", "followersCount": 75}
		],
		"cursor": ""
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	following, err := c.GetFollows(context.Background(), "alice.bsky.social", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(following) != 1 {
		t.Fatalf("got %d following, want 1", len(following))
	}
	if following[0].Handle != "carol.bsky.social" {
		t.Errorf("handle = %q, want carol.bsky.social", following[0].Handle)
	}
	if following[0].DisplayName != "Carol" {
		t.Errorf("display_name = %q, want Carol", following[0].DisplayName)
	}
}

func TestGetStarterPacks(t *testing.T) {
	body := `{
		"starterPacks": [
			{
				"uri": "at://did:plc:alice/app.bsky.graph.starterpack/mypack",
				"record": {"name": "Tech Pack", "description": "A tech pack"},
				"creator": {"handle": "alice.bsky.social"},
				"listItemCount": 25
			}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	packs, err := c.GetActorStarterPacks(context.Background(), "alice.bsky.social", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(packs) != 1 {
		t.Fatalf("got %d packs, want 1", len(packs))
	}
	if packs[0].Name != "Tech Pack" {
		t.Errorf("name = %q, want Tech Pack", packs[0].Name)
	}
	if packs[0].Creator != "alice.bsky.social" {
		t.Errorf("creator = %q, want alice.bsky.social", packs[0].Creator)
	}
	if packs[0].ListCount != 25 {
		t.Errorf("list_count = %d, want 25", packs[0].ListCount)
	}
	if packs[0].URL != "https://bsky.app/starter-pack/alice.bsky.social/mypack" {
		t.Errorf("url = %q, unexpected", packs[0].URL)
	}
}

func TestURIRKey(t *testing.T) {
	cases := []struct {
		uri  string
		want string
	}{
		{"at://did:plc:abc/app.bsky.feed.post/rkey1", "rkey1"},
		{"at://did:plc:abc/app.bsky.feed.generator/myFeed", "myFeed"},
		{"rkey-only", "rkey-only"},
	}
	for _, tc := range cases {
		got := uriRKey(tc.uri)
		if got != tc.want {
			t.Errorf("uriRKey(%q) = %q, want %q", tc.uri, got, tc.want)
		}
	}
}
