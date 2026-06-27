package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	feedAPI        = "https://substack.com/api/v1/reader/feed"
	profileFeedAPI = "https://substack.com/api/v1/reader/feed/profile/"
	noteURLPrefix  = "https://substack.com/note/c-"
	maxScanPages   = 5
)

func main() {
	sdk.Run(&SubstackPlugin{})
}

type SubstackPlugin struct{}

func (p *SubstackPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/substack/feed":
		cursor := strings.TrimSpace(req.Params["cursor"])
		return fetchFeed(cursor)
	case strings.HasPrefix(req.Route, "/substack/profile"):
		userID, handle, err := resolveProfileParams(req.Params)
		if err != nil {
			return nil, err
		}
		cursor := strings.TrimSpace(req.Params["cursor"])
		return fetchProfileFeed(userID, handle, cursor)
	case req.Route == "/substack/note/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchNoteByID(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type readerFeedResponse struct {
	Items      []readerFeedItem `json:"items"`
	NextCursor string           `json:"nextCursor"`
}

type readerFeedItem struct {
	Type    string         `json:"type"`
	Context feedContext    `json:"context"`
	Comment *substackNote  `json:"comment"`
}

type feedContext struct {
	Timestamp string `json:"timestamp"`
}

type substackNote struct {
	ID            int              `json:"id"`
	Name          string           `json:"name"`
	Handle        string           `json:"handle"`
	PhotoURL      string           `json:"photo_url"`
	Body          string           `json:"body"`
	BodyJSON      json.RawMessage  `json:"body_json"`
	ReactionCount int              `json:"reaction_count"`
	ChildrenCount int              `json:"children_count"`
	Restacks      int              `json:"restacks"`
	Attachments   []substackAttach `json:"attachments"`
	Date          string           `json:"date"`
}

type substackAttach struct {
	Type         string           `json:"type"`
	ImageURL     string           `json:"imageUrl"`
	ImageWidth   int              `json:"imageWidth"`
	ImageHeight  int              `json:"imageHeight"`
	LinkMetadata *linkMetadata    `json:"linkMetadata"`
	MediaUpload  *mediaUpload     `json:"mediaUpload"`
	Comment      *substackNote    `json:"comment"`
	Post         *substackPostRef `json:"post"`
}

type linkMetadata struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Image    string `json:"image"`
	Host     string `json:"host"`
}

type mediaUpload struct {
	MuxPlaybackID string `json:"mux_playback_id"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
}

type substackPostRef struct {
	Title        string `json:"title"`
	CanonicalURL string `json:"canonical_url"`
	CoverImage   string `json:"cover_image"`
}

func fetchFeed(cursor string) (*sdk.FeedResult, error) {
	return parseReaderFeed(cursor, feedAPI, "Substack Notes", "For You")
}

func fetchProfileFeed(userID, handle, cursor string) (*sdk.FeedResult, error) {
	title := "Substack Notes"
	desc := "@" + strings.TrimPrefix(strings.TrimSpace(handle), "@")
	if desc == "@" {
		desc = "user " + userID
	} else {
		title = desc
	}
	return parseReaderFeed(cursor, profileFeedAPI+profileFeedSlug(userID, handle), title, desc)
}

func parseReaderFeed(cursor, apiURL, title, description string) (*sdk.FeedResult, error) {
	raw, err := requestReaderFeed(apiURL, cursor)
	if err != nil {
		return nil, err
	}
	var resp readerFeedResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	items := make([]sdk.FeedItem, 0, len(resp.Items))
	for _, entry := range resp.Items {
		if entry.Type != "comment" || entry.Comment == nil {
			continue
		}
		item, ok := mapNoteToFeedItem(entry.Comment, entry.Context.Timestamp)
		if !ok {
			continue
		}
		items = append(items, item)
	}

	result := &sdk.FeedResult{
		Title:       title,
		Description: description,
		Items:       items,
		HasMore:     strings.TrimSpace(resp.NextCursor) != "",
	}
	if result.HasMore {
		result.Next = map[string]string{"cursor": resp.NextCursor}
	}
	return result, nil
}

func resolveProfileParams(params map[string]string) (userID, handle string, err error) {
	userID = strings.TrimSpace(params["userId"])
	handle = strings.TrimSpace(params["handle"])
	if profile := strings.TrimSpace(params["profile"]); profile != "" && userID == "" {
		userID, handle = parseProfileSlug(profile)
	}
	if userID == "" {
		return "", "", fmt.Errorf("missing userId parameter")
	}
	return userID, handle, nil
}

func parseProfileSlug(slug string) (userID, handle string) {
	slug = strings.TrimPrefix(strings.TrimSpace(slug), "@")
	if i := strings.Index(slug, "-"); i > 0 {
		prefix := slug[:i]
		if _, err := strconv.Atoi(prefix); err == nil {
			return prefix, slug[i+1:]
		}
	}
	return slug, ""
}

func profileFeedSlug(userID, handle string) string {
	handle = strings.TrimPrefix(strings.TrimSpace(handle), "@")
	if handle != "" {
		return userID + "-" + handle
	}
	return userID
}

func fetchNoteByID(id string) (*sdk.FeedResult, error) {
	cursor := ""
	for page := 0; page < maxScanPages; page++ {
		raw, err := requestFeed(cursor)
		if err != nil {
			return nil, err
		}
		var resp readerFeedResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("parse feed: %w", err)
		}
		for _, entry := range resp.Items {
			if entry.Type != "comment" || entry.Comment == nil {
				continue
			}
			if strconv.Itoa(entry.Comment.ID) != id {
				continue
			}
			item, ok := mapNoteToFeedItem(entry.Comment, entry.Context.Timestamp)
			if !ok {
				return nil, fmt.Errorf("note %s has no content", id)
			}
			return &sdk.FeedResult{Title: item.Title, Items: []sdk.FeedItem{item}}, nil
		}
		cursor = strings.TrimSpace(resp.NextCursor)
		if cursor == "" {
			break
		}
	}
	return nil, fmt.Errorf("note not found: %s", id)
}

func requestFeed(cursor string) ([]byte, error) {
	return requestReaderFeed(feedAPI, cursor)
}

func requestReaderFeed(apiURL, cursor string) ([]byte, error) {
	cursor = normalizeFeedCursor(cursor)
	feedURL := apiURL
	if cursor != "" {
		sep := "?"
		if strings.Contains(apiURL, "?") {
			sep = "&"
		}
		feedURL = apiURL + sep + "cursor=" + url.QueryEscape(cursor)
	}
	body, status, err := host.HTTPGet(feedURL, map[string]string{
		"User-Agent": "Mozilla/5.0",
		"Accept":     "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("substack feed returned status %d", status)
	}
	return body, nil
}

func mapNoteToFeedItem(note *substackNote, contextTS string) (sdk.FeedItem, bool) {
	body := strings.TrimSpace(note.Body)
	if body == "" && len(note.BodyJSON) == 0 {
		return sdk.FeedItem{}, false
	}

	media, quote := parseAttachments(note.Attachments)
	kind := classifyKind(body, media, quote)

	publishedAt := strings.TrimSpace(contextTS)
	if publishedAt == "" {
		publishedAt = strings.TrimSpace(note.Date)
	}

	content := string(note.BodyJSON)
	if content == "" {
		content = body
	}

	summary := body
	if kind == "long" && utf8.RuneCountInString(body) > 280 {
		summary = truncateRunes(body, 280) + "…"
	}

	item := sdk.FeedItem{
		ID:           strconv.Itoa(note.ID),
		Title:        noteTitle(body),
		URL:          noteURLPrefix + strconv.Itoa(note.ID),
		Summary:      summary,
		Content:      content,
		Author:       strings.TrimSpace(note.Name),
		AuthorAvatar: strings.TrimSpace(note.PhotoURL),
		AuthorHandle: strings.TrimSpace(note.Handle),
		PublishedAt:  publishedAt,
		Kind:         kind,
		Stats: &sdk.SocialStats{
			Likes:    note.ReactionCount,
			Replies:  note.ChildrenCount,
			Restacks: note.Restacks,
		},
		Media: media,
		Quote: quote,
	}

	if len(media) > 0 {
		for _, m := range media {
			if m.Type == "image" && m.URL != "" {
				item.Image = m.URL
				item.Cover = m.URL
				break
			}
			if m.Thumbnail != "" {
				item.Image = m.Thumbnail
				item.Cover = m.Thumbnail
				break
			}
		}
	}

	return item, true
}

func parseAttachments(attachments []substackAttach) ([]sdk.SocialMedia, *sdk.SocialQuote) {
	var media []sdk.SocialMedia
	var quote *sdk.SocialQuote

	for _, att := range attachments {
		switch att.Type {
		case "image":
			if att.ImageURL == "" {
				continue
			}
			media = append(media, sdk.SocialMedia{
				Type:     "image",
				URL:      att.ImageURL,
				Width:    att.ImageWidth,
				Height:   att.ImageHeight,
				Thumbnail: att.ImageURL,
			})
		case "video":
			if att.MediaUpload == nil || att.MediaUpload.MuxPlaybackID == "" {
				continue
			}
			playbackID := att.MediaUpload.MuxPlaybackID
			media = append(media, sdk.SocialMedia{
				Type:       "video",
				PlaybackID: playbackID,
				URL:        "https://stream.mux.com/" + playbackID + ".m3u8",
				Width:      att.MediaUpload.Width,
				Height:     att.MediaUpload.Height,
			})
		case "link":
			if att.LinkMetadata == nil {
				continue
			}
			lm := att.LinkMetadata
			media = append(media, sdk.SocialMedia{
				Type:      "link",
				URL:       lm.URL,
				Title:     lm.Title,
				Thumbnail: firstNonEmpty(lm.Image, lm.URL),
			})
		case "comment":
			if att.Comment == nil {
				continue
			}
			c := att.Comment
			quote = &sdk.SocialQuote{
				ID:           strconv.Itoa(c.ID),
				Author:       strings.TrimSpace(c.Name),
				AuthorAvatar: strings.TrimSpace(c.PhotoURL),
				AuthorHandle: strings.TrimSpace(c.Handle),
				Body:         strings.TrimSpace(c.Body),
				URL:          noteURLPrefix + strconv.Itoa(c.ID),
			}
		case "post":
			if att.Post == nil {
				continue
			}
			p := att.Post
			media = append(media, sdk.SocialMedia{
				Type:      "link",
				URL:       p.CanonicalURL,
				Title:     p.Title,
				Thumbnail: p.CoverImage,
			})
		}
	}

	return media, quote
}

func classifyKind(body string, media []sdk.SocialMedia, quote *sdk.SocialQuote) string {
	if quote != nil {
		return "long"
	}
	linkCount := 0
	for _, m := range media {
		if m.Type == "link" {
			linkCount++
		}
	}
	if linkCount > 0 || len(media) > 1 {
		return "long"
	}
	if utf8.RuneCountInString(body) > 280 {
		return "long"
	}
	return "short"
}

func noteTitle(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "Note"
	}
	line := body
	if idx := strings.IndexAny(body, "\n"); idx >= 0 {
		line = body[:idx]
	}
	line = strings.TrimSpace(line)
	if utf8.RuneCountInString(line) > 80 {
		return truncateRunes(line, 80) + "…"
	}
	return line
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	count := 0
	for _, r := range s {
		if count >= max {
			break
		}
		b.WriteRune(r)
		count++
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// normalizeFeedCursor drops bogus page-number cursors from legacy pagination clients.
func normalizeFeedCursor(cursor string) string {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return ""
	}
	if _, err := strconv.Atoi(cursor); err == nil {
		return ""
	}
	return cursor
}
