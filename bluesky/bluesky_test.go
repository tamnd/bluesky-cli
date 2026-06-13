package bluesky

import (
	"context"
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
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
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
	if p.URL == "" {
		t.Error("url should not be empty")
	}
}

func TestGetAuthorFeed(t *testing.T) {
	body := `{
		"feed": [
			{
				"post": {
					"uri": "at://did:plc:abc/app.bsky.feed.post/rkey1",
					"author": {"handle": "alice.bsky.social", "did": "did:plc:abc"},
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
