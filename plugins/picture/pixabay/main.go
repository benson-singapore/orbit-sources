package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	imageAPI = "https://pixabay.com/api/"
	videoAPI = "https://pixabay.com/api/videos/"
)

func main() {
	sdk.Run(&PixabayPlugin{})
}

type PixabayPlugin struct{}

var channelLabels = map[string]string{
	"editors-choice-photo": "编辑精选 · 照片",
	"latest-photo":         "最新照片",
	"popular-photo":        "热门照片",
	"illustration":         "插画",
	"vector":               "矢量图",
	"photo-nature":         "自然",
	"photo-animals":        "动物",
	"photo-travel":         "旅行",
	"photo-buildings":      "建筑",
	"photo-places":         "地点",
	"photo-food":           "美食",
	"photo-sports":         "运动",
	"photo-people":         "人物",
	"photo-backgrounds":    "背景",
	"photo-fashion":        "时尚",
	"photo-science":        "科学",
	"photo-computer":       "科技",
	"photo-business":       "商业",
	"photo-music":          "音乐",
	"photo-transportation": "交通",
	"photo-health":         "健康",
	"photo-education":      "教育",
	"photo-feelings":       "情感",
	"photo-religion":       "宗教",
	"photo-industry":       "工业",
	"photo-horizontal":     "横向照片",
	"photo-vertical":       "纵向照片",
	"editors-choice-video": "编辑精选 · 视频",
	"latest-video":         "最新视频",
	"popular-video":        "热门视频",
	"video-nature":         "自然视频",
	"video-travel":         "旅行视频",
	"video-animals":        "动物视频",
	"search":               "搜索",
}

var reservedParams = map[string]bool{
	"page":     true,
	"size":     true,
	"per_page": true,
	"query":    true,
	"media":    true,
}

func (p *PixabayPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	apiKey := req.Var("apiKey")
	if apiKey == "" {
		return nil, fmt.Errorf("Pixabay API key required (configure variable apiKey in plugin settings)")
	}
	lang := req.Var("lang")
	if lang == "" {
		lang = "zh"
	}
	safesearch := req.Var("safesearch")
	if safesearch == "" {
		safesearch = "true"
	}

	switch {
	case req.Route == "/pixabay/images" || strings.HasPrefix(req.Route, "/pixabay/images"):
		return fetchImages(req.Params, apiKey, lang, safesearch, req.ChannelID)
	case req.Route == "/pixabay/videos" || strings.HasPrefix(req.Route, "/pixabay/videos"):
		return fetchVideos(req.Params, apiKey, lang, safesearch, req.ChannelID)
	case req.Route == "/pixabay/search" || strings.HasPrefix(req.Route, "/pixabay/search"):
		media := strings.TrimSpace(req.Params["media"])
		if media == "video" || media == "videos" {
			return fetchVideos(req.Params, apiKey, lang, safesearch, req.ChannelID)
		}
		return fetchImages(req.Params, apiKey, lang, safesearch, req.ChannelID)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type pixabayImageResponse struct {
	Total     int           `json:"total"`
	TotalHits int           `json:"totalHits"`
	Hits      []pixabayImage `json:"hits"`
}

type pixabayImage struct {
	ID            int    `json:"id"`
	PageURL       string `json:"pageURL"`
	Type          string `json:"type"`
	Tags          string `json:"tags"`
	PreviewURL    string `json:"previewURL"`
	WebformatURL  string `json:"webformatURL"`
	LargeImageURL string `json:"largeImageURL"`
	ImageWidth    int    `json:"imageWidth"`
	ImageHeight   int    `json:"imageHeight"`
	Views         int    `json:"views"`
	Downloads     int    `json:"downloads"`
	Likes         int    `json:"likes"`
	Comments      int    `json:"comments"`
	User          string `json:"user"`
	Name          string `json:"name"`
}

type pixabayVideoResponse struct {
	Total     int           `json:"total"`
	TotalHits int           `json:"totalHits"`
	Hits      []pixabayVideo `json:"hits"`
}

type pixabayVideo struct {
	ID       int    `json:"id"`
	PageURL  string `json:"pageURL"`
	Type     string `json:"type"`
	Tags     string `json:"tags"`
	Duration int    `json:"duration"`
	Videos   struct {
		Large  videoRendition `json:"large"`
		Medium videoRendition `json:"medium"`
		Small  videoRendition `json:"small"`
		Tiny   videoRendition `json:"tiny"`
	} `json:"videos"`
	Views     int    `json:"views"`
	Downloads int    `json:"downloads"`
	Likes     int    `json:"likes"`
	Comments  int    `json:"comments"`
	User      string `json:"user"`
	Name      string `json:"name"`
}

type videoRendition struct {
	URL       string `json:"url"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbnail string `json:"thumbnail"`
}

func fetchImages(params map[string]string, apiKey, lang, safesearch, channelID string) (*sdk.FeedResult, error) {
	query := buildQuery(params, apiKey, lang, safesearch)
	rawURL := imageAPI + "?" + query.Encode()

	body, status, err := host.HTTPGet(rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	var resp pixabayImageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	items := make([]sdk.FeedItem, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		imageURL := pickImageURL(hit)

		title := strings.TrimSpace(hit.Name)
		if title == "" {
			title = firstTag(hit.Tags)
		}
		if title == "" {
			title = fmt.Sprintf("Pixabay #%d", hit.ID)
		}

		items = append(items, sdk.FeedItem{
			ID:      fmt.Sprintf("pixabay-image-%d", hit.ID),
			Title:   title,
			URL:     hit.PageURL,
			Summary: imageSummary(hit),
			Author:  hit.User,
			Cover:   imageURL,
			Image:   imageURL,
			Tags:    splitTags(hit.Tags),
		})
	}

	page := pageNum(params)
	perPage := perPageNum(params)
	title := channelLabels[channelID]
	if title == "" {
		title = "Pixabay 图片"
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: fmt.Sprintf("共 %d 条可访问结果（总计 %d）", resp.TotalHits, resp.Total),
		Items:       items,
		HasMore:     page*perPage < resp.TotalHits,
	}, nil
}

func fetchVideos(params map[string]string, apiKey, lang, safesearch, channelID string) (*sdk.FeedResult, error) {
	query := buildQuery(params, apiKey, lang, safesearch)
	rawURL := videoAPI + "?" + query.Encode()

	body, status, err := host.HTTPGet(rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	var resp pixabayVideoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	items := make([]sdk.FeedItem, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		rendition := pickVideoRendition(hit.Videos.Medium, hit.Videos.Large, hit.Videos.Small, hit.Videos.Tiny)
		thumb := rendition.Thumbnail
		if thumb == "" {
			thumb = pickVideoRendition(hit.Videos.Small, hit.Videos.Tiny, hit.Videos.Medium, hit.Videos.Large).Thumbnail
		}

		title := strings.TrimSpace(hit.Name)
		if title == "" {
			title = firstTag(hit.Tags)
		}
		if title == "" {
			title = fmt.Sprintf("Pixabay Video #%d", hit.ID)
		}

		items = append(items, sdk.FeedItem{
			ID:      fmt.Sprintf("pixabay-video-%d", hit.ID),
			Title:   title,
			URL:     hit.PageURL,
			Content: rendition.URL,
			Summary: videoSummary(hit),
			Author:  hit.User,
			Cover:   thumb,
			Image:   thumb,
			Tags:    splitTags(hit.Tags),
		})
	}

	page := pageNum(params)
	perPage := perPageNum(params)
	title := channelLabels[channelID]
	if title == "" {
		title = "Pixabay 视频"
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: fmt.Sprintf("共 %d 条可访问结果（总计 %d）", resp.TotalHits, resp.Total),
		Items:       items,
		HasMore:     page*perPage < resp.TotalHits,
	}, nil
}

func buildQuery(params map[string]string, apiKey, lang, safesearch string) url.Values {
	query := url.Values{
		"key":        {apiKey},
		"lang":       {lang},
		"safesearch": {safesearch},
		"page":       {strconv.Itoa(pageNum(params))},
		"per_page":   {strconv.Itoa(perPageNum(params))},
	}

	if q := strings.TrimSpace(params["query"]); q != "" {
		query.Set("q", q)
	}

	for key, value := range params {
		if reservedParams[key] || strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}

	return query
}

func pageNum(params map[string]string) int {
	page, err := strconv.Atoi(strings.TrimSpace(params["page"]))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func perPageNum(params map[string]string) int {
	raw := strings.TrimSpace(params["per_page"])
	if raw == "" {
		raw = strings.TrimSpace(params["size"])
	}
	size, err := strconv.Atoi(raw)
	if err != nil || size < 3 {
		return 50
	}
	if size > 200 {
		return 200
	}
	return size
}

func pickImageURL(hit pixabayImage) string {
	if u := strings.TrimSpace(hit.LargeImageURL); u != "" {
		return u
	}
	if u := strings.TrimSpace(hit.WebformatURL); u != "" {
		return u
	}
	return strings.TrimSpace(hit.PreviewURL)
}

func imageSummary(hit pixabayImage) string {
	parts := []string{
		fmt.Sprintf("%s · %dx%d", hit.Type, hit.ImageWidth, hit.ImageHeight),
		fmt.Sprintf("浏览 %d · 下载 %d · 赞 %d", hit.Views, hit.Downloads, hit.Likes),
	}
	if hit.Comments > 0 {
		parts = append(parts, fmt.Sprintf("评论 %d", hit.Comments))
	}
	return strings.Join(parts, " · ")
}

func videoSummary(hit pixabayVideo) string {
	parts := []string{
		fmt.Sprintf("%s · %ds", hit.Type, hit.Duration),
		fmt.Sprintf("浏览 %d · 下载 %d · 赞 %d", hit.Views, hit.Downloads, hit.Likes),
	}
	if hit.Comments > 0 {
		parts = append(parts, fmt.Sprintf("评论 %d", hit.Comments))
	}
	return strings.Join(parts, " · ")
}

func splitTags(tags string) []string {
	raw := strings.Split(tags, ",")
	out := make([]string, 0, len(raw))
	for _, tag := range raw {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			out = append(out, tag)
		}
	}
	return out
}

func firstTag(tags string) string {
	for _, tag := range splitTags(tags) {
		return tag
	}
	return ""
}

func pickVideoRendition(options ...videoRendition) videoRendition {
	for _, option := range options {
		if strings.TrimSpace(option.URL) != "" {
			return option
		}
	}
	return videoRendition{}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
