package sdk

// FetchRequest is passed to Plugin.Fetch from the host runtime.
type FetchRequest struct {
	ChannelID string            `json:"channelId"`
	Route     string            `json:"route"`
	Params    map[string]string `json:"params"`
	Vars      map[string]string `json:"vars,omitempty"`    // user-provided values from manifest variables
	Secrets   map[string]string `json:"secrets,omitempty"` // deprecated: same as Vars, kept for compatibility
}

// Var returns a user-configured variable by key.
// Checks Vars first, then Secrets for backward compatibility.
func (r *FetchRequest) Var(key string) string {
	if r.Vars != nil {
		if v := r.Vars[key]; v != "" {
			return v
		}
	}
	if r.Secrets != nil {
		return r.Secrets[key]
	}
	return ""
}

// FeedResult is the normalized feed payload returned to the host.
type FeedResult struct {
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Items       []FeedItem        `json:"items,omitempty"`
	Tree        []TreeNode        `json:"tree,omitempty"`
	HasMore     bool              `json:"hasMore,omitempty"`
	Next        map[string]string `json:"next,omitempty"`
}

// TreeNode is a hierarchical category node (e.g. navigation trees).
type TreeNode struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	URL      string     `json:"url,omitempty"`
	Site     string     `json:"site,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

// FeedItem is one entry in a feed result.
type FeedItem struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	URL          string        `json:"url"`
	Content      string        `json:"content,omitempty"`
	Summary      string        `json:"summary,omitempty"`
	Author       string        `json:"author,omitempty"`
	Cover        string        `json:"cover,omitempty"`
	Image        string        `json:"image,omitempty"`
	PublishedAt  string        `json:"published_at"`
	Tags         []string      `json:"tags,omitempty"`
	Kind         string        `json:"kind,omitempty"` // "short" | "long" (social notes)
	AuthorAvatar string        `json:"author_avatar,omitempty"`
	AuthorHandle string        `json:"author_handle,omitempty"`
	Stats        *SocialStats  `json:"stats,omitempty"`
	Media        []SocialMedia `json:"media,omitempty"`
	Quote        *SocialQuote  `json:"quote,omitempty"`
}

// SocialStats holds engagement counts for social feed items.
type SocialStats struct {
	Likes    int `json:"likes"`
	Replies  int `json:"replies"`
	Restacks int `json:"restacks"`
}

// SocialMedia is an image, video, or link attachment on a social note.
type SocialMedia struct {
	Type       string `json:"type"` // image | video | link
	URL        string `json:"url,omitempty"`
	Thumbnail  string `json:"thumbnail,omitempty"`
	Title      string `json:"title,omitempty"`
	PlaybackID string `json:"playback_id,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
}

// SocialQuote is a quoted/reposted note embedded in a social feed item.
type SocialQuote struct {
	ID           string `json:"id"`
	Author       string `json:"author"`
	AuthorAvatar string `json:"author_avatar,omitempty"`
	AuthorHandle string `json:"author_handle,omitempty"`
	Body         string `json:"body"`
	URL          string `json:"url,omitempty"`
}
