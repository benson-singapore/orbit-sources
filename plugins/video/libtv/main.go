package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&LibTVPlugin{})
}

type LibTVPlugin struct{}

const apiURL = "https://api.liblib.tv/api/community/project/template/feed/stream"

var channelLabelMap = map[string]string{
	"3036": "AI漫剧精卫计划",
	"3040": "广告导演请就位",
	"1000": "精选画布",
	"1200": "专业影视",
	"1800": "短剧漫剧",
	"1100": "商业广告",
	"1300": "动漫游戏",
	"1500": "教育生活",
	"1600": "TV工具箱",
}

// API Request - tagId must be int
type APIRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	TagID    int `json:"tagId"`
}

type APIResponse struct {
	Code int         `json:"code"`
	Data ResponseData `json:"data"`
}

type ResponseData struct {
	List    []Project `json:"list"`
	HasMore bool      `json:"hasMore"`
	Total   int       `json:"total"`
}

type Project struct {
	TemplateUUID string    `json:"templateUuid"`
	ProjectUUID  string    `json:"projectUuid"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CoverURL     string    `json:"coverUrl"`
	FinalOutput  string    `json:"finalOutput"`
	Nickname     string    `json:"nickname"`
	Avatar       string    `json:"avatar"`
	LikeCount    int       `json:"likeCount"`
	PublishAt    string    `json:"publishAt"`
	CreateAt     string    `json:"createAt"`
}

func (p *LibTVPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/libtv/feed/:tagId":
		tagID := req.Params["tagId"]
		if tagID == "" {
			tagID = "3036"
		}
		pageStr := req.Params["page"]
		page := 1
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}
		return fetchFeed(tagID, page)
	case req.Route == "/libtv/detail/:id":
		// id is the video URL passed from feed item
		videoURL := strings.TrimSpace(req.Params["id"])
		if videoURL == "" {
			return nil, fmt.Errorf("missing video URL parameter")
		}
		return fetchDetail(videoURL)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchFeed(tagIDStr string, page int) (*sdk.FeedResult, error) {
	label := channelLabelMap[tagIDStr]
	if label == "" {
		label = tagIDStr
	}

	// Convert tagId to int
	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tagId: %s", tagIDStr)
	}

	// Build request payload - tagId as int
	reqBody := APIRequest{
		Page:     page,
		PageSize: 20,
		TagID:    tagID,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Make API request
	body, status, err := host.HTTPPost(apiURL, map[string]string{
		"Content-Type":   "application/json",
		"Accept":         "application/json, text/plain, */*",
		"Origin":         "https://www.liblib.tv",
		"Referer":        "https://www.liblib.tv/",
		"User-Agent":     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		"X-Language":     "zh",
	}, string(reqJSON))

	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("api returned status %d", status)
	}

	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("api error code %d", resp.Code)
	}

	var items []sdk.FeedItem
	for _, proj := range resp.Data.List {
		if proj.TemplateUUID == "" || proj.Name == "" {
			continue
		}

		publishedAt := parseTime(proj.CreateAt)

		items = append(items, sdk.FeedItem{
			ID:            proj.FinalOutput, // Use video URL as ID for detail routing
			Title:         proj.Name,
			URL:           proj.FinalOutput,
			Cover:         proj.CoverURL,
			Image:         proj.CoverURL,
			Summary:       truncate(proj.Description, 200),
			Content:       proj.Description,
			PublishedAt:   publishedAt,
			Author:        proj.Nickname,
			AuthorAvatar:  proj.Avatar,
			Tags:          []string{label},
		})
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("LibLib TV - %s", label),
		Description: label,
		Items:       items,
		HasMore:     resp.Data.HasMore,
	}

	if resp.Data.HasMore && page > 0 {
		result.Next = map[string]string{"page": strconv.Itoa(page + 1)}
	}

	return result, nil
}

func fetchDetail(videoURL string) (*sdk.FeedResult, error) {
	// Build HTML5 video player with the video URL
	videoHTML := buildVideoPlayer(videoURL)

	item := sdk.FeedItem{
		ID:      videoURL,
		Title:   "Video Player",
		URL:     videoURL,
		Content: videoHTML,
	}

	return &sdk.FeedResult{
		Title:       "Video Detail",
		Description: "Video player",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func buildVideoPlayer(videoURL string) string {
	// Standard MP4 player with proper video source
	return fmt.Sprintf(`
<div style="position: relative; width: 100%%; padding-bottom: 56.25%%; height: 0; background: #000;">
  <video id="video-player"
         style="position: absolute; top: 0; left: 0; width: 100%%; height: 100%%;"
         controls controlsList="nodownload"
         preload="metadata">
    <source src="%s" type="video/mp4">
    Your browser does not support the video tag. 
    <a href="%s" target="_blank">Download video</a>
  </video>
</div>
`, videoURL, videoURL)
}

func parseTime(timeStr string) string {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return time.Now().Format(time.RFC3339)
	}

	// Expected format: "2026年06月15日 18:10"
	formats := []string{
		"2006年01月02日 15:04",
		"2006年01月02日 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format(time.RFC3339)
		}
	}

	return time.Now().Format(time.RFC3339)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
