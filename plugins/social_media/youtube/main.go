package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

// PaginationCache stores pagination tokens for different routes/queries
type PaginationCache struct {
	CacheKey   string            // Route key: "channel:{id}", "playlist:{id}", "search:{query}"
	PageTokens map[int]string    // Map of page number -> pageToken
	Mutex      sync.RWMutex     // Thread-safe access
}

// Global cache for all pagination types
var paginationCaches = make(map[string]*PaginationCache)
var cacheMutex sync.RWMutex

// getPaginationCache returns or creates a cache for the given key
func getPaginationCache(key string) *PaginationCache {
	cacheMutex.RLock()
	if cache, exists := paginationCaches[key]; exists {
		cacheMutex.RUnlock()
		return cache
	}
	cacheMutex.RUnlock()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Double-check after acquiring write lock
	if cache, exists := paginationCaches[key]; exists {
		return cache
	}

	cache := &PaginationCache{
		CacheKey:   key,
		PageTokens: make(map[int]string),
	}
	paginationCaches[key] = cache
	return cache
}

func main() {
	sdk.Run(&YouTubePlugin{})
}

type YouTubePlugin struct{}

func (p *YouTubePlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	apiKey := req.Var("apiKey")
	if apiKey == "" {
		return nil, fmt.Errorf("YouTube API key required (configure variable apiKey in plugin settings)")
	}

	switch {
	case strings.HasPrefix(req.Route, "/youtube/channel"):
		channelID := req.Params["channelId"]
		if channelID == "" {
			return nil, fmt.Errorf("channelId param required")
		}
		page := req.Params["page"]
		if page == "" {
			page = "1"
		}
		return fetchChannel(channelID, page, apiKey)
	case strings.HasPrefix(req.Route, "/youtube/user"):
		username := req.Params["username"]
		if username == "" {
			return nil, fmt.Errorf("username param required")
		}
		page := req.Params["page"]
		if page == "" {
			page = "1"
		}
		return fetchUserByUsername(username, page, apiKey)
	case strings.HasPrefix(req.Route, "/youtube/playlist"):
		playlistID := req.Params["playlistId"]
		if playlistID == "" {
			return nil, fmt.Errorf("playlistId param required")
		}
		page := req.Params["page"]
		if page == "" {
			page = "1"
		}
		return fetchPlaylist(playlistID, page, apiKey)
	case strings.HasPrefix(req.Route, "/youtube/search"):
		query := req.Params["query"]
		if query == "" {
			return nil, fmt.Errorf("query param required")
		}
		page := req.Params["page"]
		if page == "" {
			page = "1"
		}
		return fetchSearch(query, page, apiKey)
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
	ID             interface{}      `json:"id"` // Can be string or object
	Snippet        VideoSnippet     `json:"snippet"`
	Statistics     VideoStatistics  `json:"statistics,omitempty"`
	ContentDetails VideoContentDetails `json:"contentDetails,omitempty"`
}

type VideoSnippet struct {
	PublishedAt  string           `json:"publishedAt"`
	ChannelID    string           `json:"channelId"`
	Title        string           `json:"title"`
	Description  string           `json:"description"`
	Thumbnails   Thumbnails       `json:"thumbnails"`
	ChannelTitle string           `json:"channelTitle"`
	ResourceID   *ResourceID      `json:"resourceId,omitempty"` // For playlist items
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
	ViewCount     string `json:"viewCount"`
	LikeCount     string `json:"likeCount"`
	DislikeCount  string `json:"dislikeCount"`
	CommentCount  string `json:"commentCount"`
}

type VideoContentDetails struct {
	Duration string `json:"duration"` // ISO 8601 format, e.g., PT10M30S
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

func fetchChannel(channelID string, page string, apiKey string) (*sdk.FeedResult, error) {
	// Step 1: Get the "uploads" playlist ID for the channel
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

	// Step 2: Fetch videos from uploads playlist with pagination
	return fetchPlaylist(uploadsPlaylistID, page, apiKey)
}

func fetchUserByUsername(username string, page string, apiKey string) (*sdk.FeedResult, error) {
	// Normalize username: remove @ prefix if present
	if strings.HasPrefix(username, "@") {
		username = username[1:]
	}

	// Extract channel ID from user's channel page
	channelID, err := getChannelIDFromUsername(username)
	if err != nil {
		return nil, fmt.Errorf("get channel id from username: %w", err)
	}

	// Use channel ID to fetch feed with pagination
	return fetchChannel(channelID, page, apiKey)
}

func getChannelIDFromUsername(username string) (string, error) {
	// Fetch the channel page HTML
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

	// Extract channel ID from HTML using regex
	// Pattern: "externalId":"UC[A-Za-z0-9_-]{21}[AQgw]"
	re := regexp.MustCompile(`"externalId":"(UC[A-Za-z0-9_-]{21}[AQgw])"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("channel id not found in page")
	}

	return matches[1], nil
}

func fetchPlaylist(playlistID string, page string, apiKey string) (*sdk.FeedResult, error) {
	// Parse page number
	pageNum := 1
	if page != "" {
		fmt.Sscanf(page, "%d", &pageNum)
	}
	if pageNum < 1 {
		pageNum = 1
	}

	// Get cache for this playlist
	cacheKey := fmt.Sprintf("playlist:%s", playlistID)
	cache := getPaginationCache(cacheKey)

	// Get pageToken from cache
	pageToken, err := getPageTokenFromCache(cache, pageNum, playlistID, apiKey, "playlist")
	if err != nil {
		return nil, fmt.Errorf("get page token: %w", err)
	}

	// Use playlistItems.list (1 unit cost) instead of search.list (100 units)
	playlistURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=%s&maxResults=50&key=%s", playlistID, apiKey)
	
	if pageToken != "" {
		playlistURL += fmt.Sprintf("&pageToken=%s", url.QueryEscape(pageToken))
	}

	body, status, err := host.HTTPGet(playlistURL, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch playlist: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("fetch playlist: http status %d, body: %s", status, string(body))
	}

	var apiResp YouTubeAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse playlist response: %w", err)
	}

	if len(apiResp.Items) == 0 {
		return nil, fmt.Errorf("empty playlist")
	}

	// Update cache with nextPageToken
	if apiResp.NextPageToken != "" {
		cache.Mutex.Lock()
		cache.PageTokens[pageNum+1] = apiResp.NextPageToken
		cache.Mutex.Unlock()
	}

	// Extract video IDs
	videoIDs := make([]string, 0, len(apiResp.Items))
	for _, item := range apiResp.Items {
		if item.Snippet.ResourceID != nil && item.Snippet.ResourceID.VideoID != "" {
			videoIDs = append(videoIDs, item.Snippet.ResourceID.VideoID)
		}
	}

	// Fetch video statistics (views, likes, duration, etc.) in batch - 1 unit for up to 50 videos
	videosURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?part=statistics,contentDetails&id=%s&key=%s", strings.Join(videoIDs, ","), apiKey)
	statsBody, statsStatus, err := host.HTTPGet(videosURL, map[string]string{
		"Accept": "application/json",
	})
	
	// Build stats map
	statsMap := make(map[string]VideoStatistics)
	if err == nil && statsStatus >= 200 && statsStatus < 300 {
		var statsResp YouTubeAPIResponse
		if json.Unmarshal(statsBody, &statsResp) == nil {
			for _, video := range statsResp.Items {
				videoID := extractVideoIDFromInterface(video.ID)
				if videoID != "" {
					statsMap[videoID] = video.Statistics
				}
			}
		}
	}

	// Build title with pagination info
	title := fmt.Sprintf("YouTube Playlist (Page %d)", pageNum)
	if apiResp.NextPageToken != "" {
		title += " [more available]"
	}

	return parseAPIResponse(apiResp.Items, statsMap, title, apiKey)
}

// getPageTokenFromCache retrieves pageToken for a given page, fetching intervening pages if needed
func getPageTokenFromCache(cache *PaginationCache, pageNum int, resourceID string, apiKey string, routeType string) (string, error) {
	cache.Mutex.Lock()
	defer cache.Mutex.Unlock()

	// Page 1 always has no token
	if pageNum == 1 {
		return "", nil
	}

	// Check if we already have this token
	if token, exists := cache.PageTokens[pageNum]; exists {
		return token, nil
	}

	// If we don't have the token, we need to fetch pages sequentially
	// Find the highest page we have cached
	highestPage := 1
	for page := range cache.PageTokens {
		if page > highestPage {
			highestPage = page
		}
	}

	// Release lock during HTTP calls to avoid blocking
	cache.Mutex.Unlock()

	for currentPage := highestPage + 1; currentPage <= pageNum; currentPage++ {
		// Get token for current page
		currentPageToken := ""
		if currentPage > 1 {
			cache.Mutex.Lock()
			if token, exists := cache.PageTokens[currentPage]; exists {
				currentPageToken = token
			}
			cache.Mutex.Unlock()
		}

		// Fetch this page to get the next page token
		err := fetchPageForTokenByType(cache, currentPage, currentPageToken, resourceID, apiKey, routeType)
		if err != nil {
			cache.Mutex.Lock()
			return "", err
		}
	}

	cache.Mutex.Lock()
	// Now we should have the token for the requested page
	if token, exists := cache.PageTokens[pageNum]; exists {
		return token, nil
	}

	return "", nil
}

// fetchPageForTokenByType fetches a page and stores the nextPageToken for the next page
func fetchPageForTokenByType(cache *PaginationCache, pageNum int, pageToken string, resourceID string, apiKey string, routeType string) error {
	var apiURL string

	switch routeType {
	case "playlist":
		apiURL = fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=%s&maxResults=50&key=%s", 
			resourceID, apiKey)
	case "search":
		apiURL = fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&q=%s&type=video&maxResults=50&key=%s", 
			url.QueryEscape(resourceID), apiKey)
	default:
		return fmt.Errorf("unknown route type: %s", routeType)
	}

	if pageToken != "" {
		apiURL += fmt.Sprintf("&pageToken=%s", url.QueryEscape(pageToken))
	}

	body, status, err := host.HTTPGet(apiURL, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return fmt.Errorf("fetch page %d: %w", pageNum, err)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("fetch page %d: http status %d", pageNum, status)
	}

	var apiResp YouTubeAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("parse page %d response: %w", pageNum, err)
	}

	// Store the nextPageToken for page+1
	if apiResp.NextPageToken != "" {
		cache.Mutex.Lock()
		cache.PageTokens[pageNum+1] = apiResp.NextPageToken
		cache.Mutex.Unlock()
	}

	return nil
}

func fetchSearch(query string, page string, apiKey string) (*sdk.FeedResult, error) {
	// Parse page number
	pageNum := 1
	if page != "" {
		fmt.Sscanf(page, "%d", &pageNum)
	}
	if pageNum < 1 {
		pageNum = 1
	}

	// Get cache for this search query
	cacheKey := fmt.Sprintf("search:%s", query)
	cache := getPaginationCache(cacheKey)

	// Get pageToken from cache (will fetch missing pages if needed)
	pageToken, err := getPageTokenFromCache(cache, pageNum, query, apiKey, "search")
	if err != nil {
		return nil, fmt.Errorf("get page token: %w", err)
	}

	// Use search.list (100 units cost) to search for videos
	searchURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&q=%s&type=video&maxResults=50&key=%s", 
		url.QueryEscape(query), apiKey)
	
	// Add pageToken if provided (for pagination)
	if pageToken != "" {
		searchURL += fmt.Sprintf("&pageToken=%s", url.QueryEscape(pageToken))
	}
	
	body, status, err := host.HTTPGet(searchURL, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch search results: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("fetch search results: http status %d, body: %s", status, string(body))
	}

	var apiResp YouTubeAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	if len(apiResp.Items) == 0 {
		return nil, fmt.Errorf("no search results found")
	}

	// Update cache with nextPageToken for this page
	if apiResp.NextPageToken != "" {
		cache.Mutex.Lock()
		cache.PageTokens[pageNum+1] = apiResp.NextPageToken
		cache.Mutex.Unlock()
	}

	// Extract video IDs
	videoIDs := make([]string, 0, len(apiResp.Items))
	for _, item := range apiResp.Items {
		videoID := extractVideoIDFromInterface(item.ID)
		if videoID != "" {
			videoIDs = append(videoIDs, videoID)
		}
	}

	// Fetch video statistics (views, likes, duration, etc.) in batch - 1 unit for up to 50 videos
	statsMap := make(map[string]VideoStatistics)
	if len(videoIDs) > 0 {
		videosURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?part=statistics,contentDetails&id=%s&key=%s", 
			strings.Join(videoIDs, ","), apiKey)
		statsBody, statsStatus, err := host.HTTPGet(videosURL, map[string]string{
			"Accept": "application/json",
		})
		
		if err == nil && statsStatus >= 200 && statsStatus < 300 {
			var statsResp YouTubeAPIResponse
			if json.Unmarshal(statsBody, &statsResp) == nil {
				for _, video := range statsResp.Items {
					videoID := extractVideoIDFromInterface(video.ID)
					if videoID != "" {
						statsMap[videoID] = video.Statistics
					}
				}
			}
		}
	}

	// Build title with pagination info
	title := fmt.Sprintf("YouTube Search: %s (Page %d)", query, pageNum)
	if apiResp.NextPageToken != "" {
		title += " [more available]"
	}
	
	return parseAPIResponse(apiResp.Items, statsMap, title, apiKey)
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

func parseAPIResponse(videos []YouTubeVideo, statsMap map[string]VideoStatistics, title string, apiKey string) (*sdk.FeedResult, error) {
	items := make([]sdk.FeedItem, 0, len(videos))
	
	// Cache for channel subscriber counts to avoid duplicate API calls
	channelCache := make(map[string]string)
	
	for _, video := range videos {
		// Get video ID
		var videoID string
		if video.Snippet.ResourceID != nil && video.Snippet.ResourceID.VideoID != "" {
			videoID = video.Snippet.ResourceID.VideoID
		} else {
			videoID = extractVideoIDFromInterface(video.ID)
		}
		
		if videoID == "" {
			continue
		}

		// Parse published date
		publishedAt := parseDate(video.Snippet.PublishedAt)

		// Get thumbnail URL
		thumbnail := video.Snippet.Thumbnails.High.URL

		// Get video URL
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

		// Build tags from statistics and content details
		tags := buildTagsFromStats(statsMap[videoID])
		
		// Add duration tag if available
		if video.ContentDetails.Duration != "" {
			durationFormatted := formatDuration(video.ContentDetails.Duration)
			if durationFormatted != "" {
				tags = append(tags, fmt.Sprintf("时长:%s", durationFormatted))
			}
		}
		
		// Add channel subscriber count if available
		channelID := video.Snippet.ChannelID
		if channelID != "" {
			// Check cache first
			if subCount, exists := channelCache[channelID]; exists {
				if subCount != "" {
					tags = append(tags, fmt.Sprintf("频道关注:%s", subCount))
				}
			} else {
				// Fetch channel info (only if not cached and if we have API key)
				if apiKey != "" {
					subCount := fetchChannelSubscriberCount(channelID, apiKey)
					channelCache[channelID] = subCount
					if subCount != "" {
						tags = append(tags, fmt.Sprintf("频道关注:%s", subCount))
					}
				}
			}
		}

		// Truncate description
		content := truncateDescription(video.Snippet.Description, 200)

		item := sdk.FeedItem{
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
		}

		items = append(items, item)
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: fmt.Sprintf("YouTube videos (%d items)", len(items)),
		Items:       items,
	}, nil
}

func parseDate(dateStr string) string {
	// Parse RFC3339 format: 2024-01-15T10:30:00Z
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format(time.RFC3339)
}

// fetchChannelSubscriberCount retrieves the subscriber count for a YouTube channel
func fetchChannelSubscriberCount(channelID string, apiKey string) string {
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

	// View count
	if stats.ViewCount != "" {
		tags = append(tags, fmt.Sprintf("观看:%s", stats.ViewCount))
	}

	// Like count
	if stats.LikeCount != "" {
		tags = append(tags, fmt.Sprintf("点赞:%s", stats.LikeCount))
	}

	// Dislike count (deprecated in YouTube API but may still be available)
	if stats.DislikeCount != "" && stats.DislikeCount != "0" {
		tags = append(tags, fmt.Sprintf("点踩:%s", stats.DislikeCount))
	}

	// Comment count
	if stats.CommentCount != "" {
		tags = append(tags, fmt.Sprintf("评论:%s", stats.CommentCount))
	}

	return tags
}

// formatDuration converts ISO 8601 duration to human-readable format
// e.g., PT10M30S -> "10:30", PT1H5M30S -> "1:05:30"
func formatDuration(isoDuration string) string {
	if isoDuration == "" {
		return ""
	}

	// Remove PT prefix
	if !strings.HasPrefix(isoDuration, "PT") {
		return isoDuration
	}

	duration := isoDuration[2:]
	var hours, minutes, seconds int

	// Parse hours
	if idx := strings.Index(duration, "H"); idx != -1 {
		fmt.Sscanf(duration[:idx], "%d", &hours)
		duration = duration[idx+1:]
	}

	// Parse minutes
	if idx := strings.Index(duration, "M"); idx != -1 {
		fmt.Sscanf(duration[:idx], "%d", &minutes)
		duration = duration[idx+1:]
	}

	// Parse seconds
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
	// Truncate to maxLen and avoid cutting in the middle of a word if possible
	truncated := desc[:maxLen]
	// Find last space to avoid cutting words
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}
