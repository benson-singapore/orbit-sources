package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const maxResults = 50

func main() {
	sdk.Run(&YouTubePlugin{})
}

type YouTubePlugin struct{}

func (p *YouTubePlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	apiKey := req.Var("apiKey")
	if apiKey == "" {
		return nil, fmt.Errorf("YouTube API key required (configure variable apiKey in plugin settings)")
	}

	lastID := strings.TrimSpace(req.Params["lastId"])

	switch {
	case strings.HasPrefix(req.Route, "/youtube/channel"):
		channelID := req.Params["channelId"]
		if channelID == "" {
			return nil, fmt.Errorf("channelId param required")
		}
		return fetchChannel(channelID, lastID, apiKey)
	case strings.HasPrefix(req.Route, "/youtube/user"):
		username := req.Params["username"]
		if username == "" {
			return nil, fmt.Errorf("username param required")
		}
		return fetchUserByUsername(username, lastID, apiKey)
	case strings.HasPrefix(req.Route, "/youtube/playlist"):
		playlistID := req.Params["playlistId"]
		if playlistID == "" {
			return nil, fmt.Errorf("playlistId param required")
		}
		return fetchPlaylist(playlistID, lastID, apiKey)
	case strings.HasPrefix(req.Route, "/youtube/search"):
		query := req.Params["query"]
		if query == "" {
			return nil, fmt.Errorf("query param required")
		}
		return fetchSearch(query, lastID, apiKey)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

// YouTube Data API v3 response structures
type YouTubeAPIResponse struct {
	Items         []YouTubeVideo `json:"items"`
	NextPageToken string         `json:"nextPageToken,omitempty"`
}

type YouTubeVideo struct {
	ID             interface{}         `json:"id"`
	Snippet        VideoSnippet        `json:"snippet"`
	Statistics     VideoStatistics     `json:"statistics,omitempty"`
	ContentDetails VideoContentDetails `json:"contentDetails,omitempty"`
}

type VideoSnippet struct {
	PublishedAt  string      `json:"publishedAt"`
	ChannelID    string      `json:"channelId"`
	Title        string      `json:"title"`
	Description  string      `json:"description"`
	Thumbnails   Thumbnails  `json:"thumbnails"`
	ChannelTitle string      `json:"channelTitle"`
	ResourceID   *ResourceID `json:"resourceId,omitempty"`
}

type ResourceID struct {
	VideoID string `json:"videoId"`
}

type Thumbnails struct {
	High Thumbnail `json:"high"`
}

type Thumbnail struct {
	URL string `json:"url"`
}

type VideoStatistics struct {
	ViewCount    string `json:"viewCount"`
	LikeCount    string `json:"likeCount"`
	DislikeCount string `json:"dislikeCount"`
	CommentCount string `json:"commentCount"`
}

type VideoContentDetails struct {
	Duration string `json:"duration"`
}

type ChannelListResponse struct {
	Items []ChannelItem `json:"items"`
}

type ChannelItem struct {
	ID             string                `json:"id"`
	ContentDetails ChannelContentDetails `json:"contentDetails"`
	Statistics     ChannelStatistics     `json:"statistics,omitempty"`
}

type ChannelContentDetails struct {
	RelatedPlaylists RelatedPlaylists `json:"relatedPlaylists"`
}

type RelatedPlaylists struct {
	Uploads string `json:"uploads"`
}

type ChannelStatistics struct {
	SubscriberCount string `json:"subscriberCount"`
	ViewCount       string `json:"viewCount"`
	VideoCount      string `json:"videoCount"`
}

type listKind int

const (
	listPlaylist listKind = iota
	listSearch
)

func fetchChannel(channelID, lastID, apiKey string) (*sdk.FeedResult, error) {
	channelURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/channels?part=contentDetails&id=%s&key=%s", channelID, apiKey)
	body, status, err := host.HTTPGet(channelURL, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch channel info: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("fetch channel info: http status %d", status)
	}

	var channelResp ChannelListResponse
	if err := json.Unmarshal(body, &channelResp); err != nil {
		return nil, fmt.Errorf("parse channel response: %w", err)
	}
	if len(channelResp.Items) == 0 {
		return nil, fmt.Errorf("channel not found")
	}

	uploadsPlaylistID := channelResp.Items[0].ContentDetails.RelatedPlaylists.Uploads
	return fetchPlaylist(uploadsPlaylistID, lastID, apiKey)
}

func fetchUserByUsername(username, lastID, apiKey string) (*sdk.FeedResult, error) {
	if strings.HasPrefix(username, "@") {
		username = username[1:]
	}

	channelID, err := getChannelIDFromUsername(username)
	if err != nil {
		return nil, fmt.Errorf("get channel id from username: %w", err)
	}

	return fetchChannel(channelID, lastID, apiKey)
}

func getChannelIDFromUsername(username string) (string, error) {
	pageURL := fmt.Sprintf("https://www.youtube.com/@%s", username)
	body, status, err := host.HTTPGet(pageURL, map[string]string{
		"Accept": "text/html",
	})
	if err != nil {
		return "", fmt.Errorf("fetch user page: %w", err)
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("fetch user page: http status %d", status)
	}

	re := regexp.MustCompile(`"externalId":"(UC[A-Za-z0-9_-]{21}[AQgw])"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("channel id not found in page")
	}

	return matches[1], nil
}

func fetchPlaylist(playlistID, lastID, apiKey string) (*sdk.FeedResult, error) {
	title := "YouTube Playlist"
	return fetchVideoList(lastID, apiKey, listPlaylist, playlistID, title)
}

func fetchSearch(query, lastID, apiKey string) (*sdk.FeedResult, error) {
	title := fmt.Sprintf("YouTube Search: %s", query)
	return fetchVideoList(lastID, apiKey, listSearch, query, title)
}

func fetchVideoList(lastID, apiKey string, kind listKind, resourceID, title string) (*sdk.FeedResult, error) {
	fetchPage := func(pageToken string) (*YouTubeAPIResponse, error) {
		return requestListPage(kind, resourceID, pageToken, apiKey)
	}

	items, hasMore, err := paginateAfterLastID(lastID, fetchPage)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no videos found")
	}

	statsMap, err := fetchVideoStats(items, apiKey)
	if err != nil {
		return nil, err
	}

	return buildFeedResult(items, statsMap, title, apiKey, hasMore)
}

// paginateAfterLastID walks YouTube pageToken pages until it can return the next batch after lastID.
func paginateAfterLastID(lastID string, fetchPage func(pageToken string) (*YouTubeAPIResponse, error)) ([]YouTubeVideo, bool, error) {
	pageToken := ""

	if lastID != "" {
		found := false
		for {
			resp, err := fetchPage(pageToken)
			if err != nil {
				return nil, false, err
			}

			for i, item := range resp.Items {
				videoID := extractVideoID(item)
				if videoID == "" {
					continue
				}
				if videoID != lastID {
					continue
				}

				found = true
				remaining := resp.Items[i+1:]
				if len(remaining) > 0 {
					return remaining, resp.NextPageToken != "", nil
				}
				if resp.NextPageToken == "" {
					return nil, false, nil
				}
				pageToken = resp.NextPageToken
				break
			}

			if found {
				break
			}
			if resp.NextPageToken == "" {
				return nil, false, fmt.Errorf("cursor video not found: %s", lastID)
			}
			pageToken = resp.NextPageToken
		}
	}

	resp, err := fetchPage(pageToken)
	if err != nil {
		return nil, false, err
	}
	return resp.Items, resp.NextPageToken != "", nil
}

func requestListPage(kind listKind, resourceID, pageToken, apiKey string) (*YouTubeAPIResponse, error) {
	var apiURL string
	switch kind {
	case listPlaylist:
		apiURL = fmt.Sprintf(
			"https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=%s&maxResults=%d&key=%s",
			resourceID, maxResults, apiKey,
		)
	case listSearch:
		apiURL = fmt.Sprintf(
			"https://www.googleapis.com/youtube/v3/search?part=snippet&q=%s&type=video&maxResults=%d&key=%s",
			url.QueryEscape(resourceID), maxResults, apiKey,
		)
	default:
		return nil, fmt.Errorf("unknown list kind")
	}

	if pageToken != "" {
		apiURL += "&pageToken=" + url.QueryEscape(pageToken)
	}

	body, status, err := host.HTTPGet(apiURL, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch list page: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("fetch list page: http status %d, body: %s", status, string(body))
	}

	var resp YouTubeAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}
	return &resp, nil
}

func fetchVideoStats(videos []YouTubeVideo, apiKey string) (map[string]VideoStatistics, error) {
	videoIDs := make([]string, 0, len(videos))
	for _, item := range videos {
		if videoID := extractVideoID(item); videoID != "" {
			videoIDs = append(videoIDs, videoID)
		}
	}

	statsMap := make(map[string]VideoStatistics)
	contentMap := make(map[string]VideoContentDetails)
	if len(videoIDs) == 0 {
		return statsMap, nil
	}

	videosURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?part=statistics,contentDetails&id=%s&key=%s",
		strings.Join(videoIDs, ","), apiKey,
	)
	statsBody, statsStatus, err := host.HTTPGet(videosURL, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch video stats: %w", err)
	}
	if statsStatus < 200 || statsStatus >= 300 {
		return statsMap, nil
	}

	var statsResp YouTubeAPIResponse
	if err := json.Unmarshal(statsBody, &statsResp); err != nil {
		return statsMap, nil
	}
	for _, video := range statsResp.Items {
		videoID := extractVideoID(video)
		if videoID == "" {
			continue
		}
		statsMap[videoID] = video.Statistics
		contentMap[videoID] = video.ContentDetails
	}

	for i, item := range videos {
		videoID := extractVideoID(item)
		if details, ok := contentMap[videoID]; ok {
			videos[i].ContentDetails = details
		}
	}

	return statsMap, nil
}

func extractVideoID(video YouTubeVideo) string {
	if video.Snippet.ResourceID != nil && video.Snippet.ResourceID.VideoID != "" {
		return video.Snippet.ResourceID.VideoID
	}
	return extractVideoIDFromInterface(video.ID)
}

func extractVideoIDFromInterface(id interface{}) string {
	switch v := id.(type) {
	case string:
		return v
	case map[string]interface{}:
		if videoID, ok := v["videoId"].(string); ok {
			return videoID
		}
	}
	return ""
}

func buildFeedResult(videos []YouTubeVideo, statsMap map[string]VideoStatistics, title, apiKey string, hasMore bool) (*sdk.FeedResult, error) {
	items := make([]sdk.FeedItem, 0, len(videos))
	channelCache := make(map[string]string)

	for _, video := range videos {
		videoID := extractVideoID(video)
		if videoID == "" {
			continue
		}

		publishedAt := parseDate(video.Snippet.PublishedAt)
		thumbnail := video.Snippet.Thumbnails.High.URL
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

		tags := buildTagsFromStats(statsMap[videoID])
		if video.ContentDetails.Duration != "" {
			if durationFormatted := formatDuration(video.ContentDetails.Duration); durationFormatted != "" {
				tags = append(tags, fmt.Sprintf("时长:%s", durationFormatted))
			}
		}

		channelID := video.Snippet.ChannelID
		if channelID != "" {
			if subCount, exists := channelCache[channelID]; exists {
				if subCount != "" {
					tags = append(tags, fmt.Sprintf("频道关注:%s", subCount))
				}
			} else if apiKey != "" {
				subCount := fetchChannelSubscriberCount(channelID, apiKey)
				channelCache[channelID] = subCount
				if subCount != "" {
					tags = append(tags, fmt.Sprintf("频道关注:%s", subCount))
				}
			}
		}

		content := truncateDescription(video.Snippet.Description, 200)

		items = append(items, sdk.FeedItem{
			ID:          videoID,
			Title:       video.Snippet.Title,
			URL:         videoURL,
			Summary:     video.Snippet.Title,
			Content:     content,
			Author:      video.Snippet.ChannelTitle,
			Cover:       thumbnail,
			Image:       thumbnail,
			PublishedAt: publishedAt,
			Tags:        tags,
		})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no valid videos in response")
	}

	result := &sdk.FeedResult{
		Title:       title,
		Description: fmt.Sprintf("YouTube videos (%d items)", len(items)),
		Items:       items,
		HasMore:     hasMore,
	}
	if hasMore {
		result.Next = map[string]string{
			"lastId": items[len(items)-1].ID,
		}
	}
	return result, nil
}

func parseDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format(time.RFC3339)
}

func fetchChannelSubscriberCount(channelID, apiKey string) string {
	if channelID == "" || apiKey == "" {
		return ""
	}

	channelURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/channels?part=statistics&id=%s&key=%s",
		channelID, apiKey)

	body, status, err := host.HTTPGet(channelURL, map[string]string{
		"Accept": "application/json",
	})
	if err != nil || status < 200 || status >= 300 {
		return ""
	}

	var channelResp ChannelListResponse
	if err := json.Unmarshal(body, &channelResp); err != nil {
		return ""
	}
	if len(channelResp.Items) == 0 {
		return ""
	}

	return channelResp.Items[0].Statistics.SubscriberCount
}

func buildTagsFromStats(stats VideoStatistics) []string {
	tags := []string{}

	if stats.ViewCount != "" {
		tags = append(tags, fmt.Sprintf("观看:%s", stats.ViewCount))
	}
	if stats.LikeCount != "" {
		tags = append(tags, fmt.Sprintf("点赞:%s", stats.LikeCount))
	}
	if stats.DislikeCount != "" && stats.DislikeCount != "0" {
		tags = append(tags, fmt.Sprintf("点踩:%s", stats.DislikeCount))
	}
	if stats.CommentCount != "" {
		tags = append(tags, fmt.Sprintf("评论:%s", stats.CommentCount))
	}

	return tags
}

func formatDuration(isoDuration string) string {
	if isoDuration == "" {
		return ""
	}
	if !strings.HasPrefix(isoDuration, "PT") {
		return isoDuration
	}

	duration := isoDuration[2:]
	var hours, minutes, seconds int

	if idx := strings.Index(duration, "H"); idx != -1 {
		fmt.Sscanf(duration[:idx], "%d", &hours)
		duration = duration[idx+1:]
	}
	if idx := strings.Index(duration, "M"); idx != -1 {
		fmt.Sscanf(duration[:idx], "%d", &minutes)
		duration = duration[idx+1:]
	}
	if idx := strings.Index(duration, "S"); idx != -1 {
		fmt.Sscanf(duration[:idx], "%d", &seconds)
	}

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func truncateDescription(desc string, maxLen int) string {
	if len(desc) <= maxLen {
		return desc
	}
	truncated := desc[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}
