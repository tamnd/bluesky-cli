package bluesky

import (
	"fmt"
	"strings"
)

// Profile is the record emitted for a Bluesky user profile.
type Profile struct {
	Handle         string `json:"handle" kit:"id"`
	DisplayName    string `json:"display_name"`
	DID            string `json:"did"`
	Description    string `json:"description"`
	FollowersCount int    `json:"followers_count"`
	FollowsCount   int    `json:"follows_count"`
	PostsCount     int    `json:"posts_count"`
	IndexedAt      string `json:"indexed_at"`
	URL            string `json:"url"`
}

// Post is the record emitted for a post in an author feed.
type Post struct {
	Rank        int    `json:"rank"`
	URI         string `json:"uri" kit:"id"`
	Author      string `json:"author"`
	AuthorName  string `json:"author_name"`
	Text        string `json:"text"`
	LikeCount   int    `json:"like_count"`
	RepostCount int    `json:"repost_count"`
	ReplyCount  int    `json:"reply_count"`
	IndexedAt   string `json:"indexed_at"`
	URL         string `json:"url"`
}

// Actor is the record emitted for a search result, follower, or following.
type Actor struct {
	Rank        int    `json:"rank"`
	Handle      string `json:"handle" kit:"id"`
	DisplayName string `json:"display_name"`
	DID         string `json:"did"`
	Description string `json:"description"`
	Followers   int    `json:"followers"`
	URL         string `json:"url"`
}

// FeedGenerator is the record emitted for a popular feed generator.
type FeedGenerator struct {
	Rank        int    `json:"rank"`
	URI         string `json:"uri" kit:"id"`
	DisplayName string `json:"display_name"`
	Creator     string `json:"creator"`
	Description string `json:"description"`
	LikeCount   int    `json:"like_count"`
	URL         string `json:"url"`
}

// TrendingTopic is the record emitted for a trending topic.
type TrendingTopic struct {
	Rank  int    `json:"rank"`
	Topic string `json:"topic" kit:"id"`
	Link  string `json:"link"`
}

// ThreadPost is the record emitted for a post in a thread traversal (DFS order).
type ThreadPost struct {
	Depth       int    `json:"depth"`
	URI         string `json:"uri" kit:"id"`
	Author      string `json:"author"`
	Text        string `json:"text"`
	LikeCount   int    `json:"like_count"`
	RepostCount int    `json:"repost_count"`
	ReplyCount  int    `json:"reply_count"`
	IndexedAt   string `json:"indexed_at"`
	URL         string `json:"url"`
}

// StarterPack is the record emitted for a starter pack created by a user.
type StarterPack struct {
	Rank        int    `json:"rank"`
	URI         string `json:"uri" kit:"id"`
	Name        string `json:"name"`
	Creator     string `json:"creator"`
	Description string `json:"description"`
	ListCount   int    `json:"list_count"`
	URL         string `json:"url"`
}

// ─── wire types from the Bluesky AppView API ─────────────────────────────────

type wireProfileView struct {
	Handle         string `json:"handle"`
	DisplayName    string `json:"displayName"`
	DID            string `json:"did"`
	Description    string `json:"description"`
	FollowersCount int    `json:"followersCount"`
	FollowsCount   int    `json:"followsCount"`
	PostsCount     int    `json:"postsCount"`
	IndexedAt      string `json:"indexedAt"`
}

type wireFeedResponse struct {
	Feed   []wireFeedItem `json:"feed"`
	Cursor string         `json:"cursor"`
}

type wireFeedItem struct {
	Post wirePostView `json:"post"`
}

type wirePostView struct {
	URI         string     `json:"uri"`
	Author      wireAuthor `json:"author"`
	Record      wireRecord `json:"record"`
	LikeCount   int        `json:"likeCount"`
	RepostCount int        `json:"repostCount"`
	ReplyCount  int        `json:"replyCount"`
	IndexedAt   string     `json:"indexedAt"`
}

type wireAuthor struct {
	Handle      string `json:"handle"`
	DID         string `json:"did"`
	DisplayName string `json:"displayName"`
}

type wireRecord struct {
	Text string `json:"text"`
}

type wireSearchActorsResponse struct {
	Actors []wireActorView `json:"actors"`
	Cursor string          `json:"cursor"`
}

type wireActorView struct {
	Handle         string `json:"handle"`
	DisplayName    string `json:"displayName"`
	DID            string `json:"did"`
	Description    string `json:"description"`
	FollowersCount int    `json:"followersCount"`
}

type wireFollowersResponse struct {
	Followers []wireActorView `json:"followers"`
	Cursor    string          `json:"cursor"`
}

type wireFollowsResponse struct {
	Follows []wireActorView `json:"follows"`
	Cursor  string          `json:"cursor"`
}

type wirePopularFeedsResponse struct {
	Feeds  []wireFeedGeneratorView `json:"feeds"`
	Cursor string                  `json:"cursor"`
}

type wireFeedGeneratorView struct {
	URI         string     `json:"uri"`
	DisplayName string     `json:"displayName"`
	Description string     `json:"description"`
	LikeCount   int        `json:"likeCount"`
	Creator     wireAuthor `json:"creator"`
}

type wireTrendingResponse struct {
	Topics []wireTrendingTopic `json:"topics"`
}

type wireTrendingTopic struct {
	Topic string `json:"topic"`
	Link  string `json:"link"`
}

type wireThreadResponse struct {
	Thread wireThreadNode `json:"thread"`
}

type wireThreadNode struct {
	Type    string           `json:"$type"`
	Post    wirePostView     `json:"post"`
	Replies []wireThreadNode `json:"replies"`
}

type wireStarterPacksResponse struct {
	StarterPacks []wireStarterPackView `json:"starterPacks"`
	Cursor       string                `json:"cursor"`
}

type wireStarterPackView struct {
	URI           string                `json:"uri"`
	Record        wireStarterPackRecord `json:"record"`
	Creator       wireAuthor            `json:"creator"`
	ListItemCount int                   `json:"listItemCount"`
}

type wireStarterPackRecord struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ─── conversion helpers ───────────────────────────────────────────────────────

func profileFromWire(w wireProfileView) Profile {
	return Profile{
		Handle:         w.Handle,
		DisplayName:    w.DisplayName,
		DID:            w.DID,
		Description:    w.Description,
		FollowersCount: w.FollowersCount,
		FollowsCount:   w.FollowsCount,
		PostsCount:     w.PostsCount,
		IndexedAt:      w.IndexedAt,
		URL:            profileURL(w.Handle),
	}
}

func postFromWire(p wirePostView, rank int) Post {
	return Post{
		Rank:        rank,
		URI:         p.URI,
		Author:      p.Author.Handle,
		AuthorName:  p.Author.DisplayName,
		Text:        p.Record.Text,
		LikeCount:   p.LikeCount,
		RepostCount: p.RepostCount,
		ReplyCount:  p.ReplyCount,
		IndexedAt:   p.IndexedAt,
		URL:         postURL(p.Author.Handle, uriRKey(p.URI)),
	}
}

func actorFromWire(a wireActorView, rank int) Actor {
	return Actor{
		Rank:        rank,
		Handle:      a.Handle,
		DisplayName: a.DisplayName,
		DID:         a.DID,
		Description: a.Description,
		Followers:   a.FollowersCount,
		URL:         profileURL(a.Handle),
	}
}

func feedGenFromWire(f wireFeedGeneratorView, rank int) FeedGenerator {
	return FeedGenerator{
		Rank:        rank,
		URI:         f.URI,
		DisplayName: f.DisplayName,
		Creator:     f.Creator.Handle,
		Description: f.Description,
		LikeCount:   f.LikeCount,
		URL:         feedURL(f.Creator.Handle, uriRKey(f.URI)),
	}
}

func trendingFromWire(t wireTrendingTopic, rank int) TrendingTopic {
	return TrendingTopic{
		Rank:  rank,
		Topic: t.Topic,
		Link:  t.Link,
	}
}

func threadPostFromWire(p wirePostView, depth int) ThreadPost {
	return ThreadPost{
		Depth:       depth,
		URI:         p.URI,
		Author:      p.Author.Handle,
		Text:        p.Record.Text,
		LikeCount:   p.LikeCount,
		RepostCount: p.RepostCount,
		ReplyCount:  p.ReplyCount,
		IndexedAt:   p.IndexedAt,
		URL:         postURL(p.Author.Handle, uriRKey(p.URI)),
	}
}

func starterPackFromWire(s wireStarterPackView, rank int) StarterPack {
	return StarterPack{
		Rank:        rank,
		URI:         s.URI,
		Name:        s.Record.Name,
		Creator:     s.Creator.Handle,
		Description: s.Record.Description,
		ListCount:   s.ListItemCount,
		URL:         starterPackURL(s.Creator.Handle, uriRKey(s.URI)),
	}
}

// ─── URL helpers ─────────────────────────────────────────────────────────────

func profileURL(handle string) string {
	return fmt.Sprintf("https://bsky.app/profile/%s", handle)
}

func postURL(handle, rkey string) string {
	return fmt.Sprintf("https://bsky.app/profile/%s/post/%s", handle, rkey)
}

func feedURL(handle, rkey string) string {
	return fmt.Sprintf("https://bsky.app/profile/%s/feed/%s", handle, rkey)
}

func starterPackURL(handle, rkey string) string {
	return fmt.Sprintf("https://bsky.app/starter-pack/%s/%s", handle, rkey)
}

// uriRKey extracts the record key (rkey) from an AT URI.
// AT URIs have the form at://did/collection/rkey.
func uriRKey(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) < 1 {
		return uri
	}
	return parts[len(parts)-1]
}
