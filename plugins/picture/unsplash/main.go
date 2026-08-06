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
	napiBase       = "https://unsplash.com/napi"
	clientVersion  = "03e5cd8bd8b353ee8fd3ba7f33657dba7b9d0d4a"
	defaultPerPage = 20
	maxPerPage     = 30
	userAgent      = "Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Mobile Safari/537.36"
)

func main() {
	sdk.Run(&UnsplashPlugin{})
}

type UnsplashPlugin struct{}

var channelLabels = map[string]string{
	"nature":                "自然",
	"animals":               "动物",
	"travel":                "旅行",
	"wallpapers":            "壁纸",
	"architecture-interior": "建筑与室内",
	"people":                "人物",
	"food-drink":            "美食",
	"film":                  "胶片",
	"textures-patterns":     "纹理与图案",
	"street-photography":    "街拍",
	"business-work":         "商业与工作",
	"fashion-beauty":        "时尚与美妆",
	"3d-renders":            "3D 渲染",
	"experimental":          "实验摄影",
	"arts-culture":          "艺术与文化",
	"current-events":        "时事",
	"spring":                "春天",
	"optimism":              "乐观",
	"search":                "搜索",
}

var reservedParams = map[string]bool{
	"page":     true,
	"size":     true,
	"per_page": true,
	"query":    true,
	"topic":    true,
}

func (p *UnsplashPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	cookie := strings.TrimSpace(req.Var("cookie"))
	if cookie == "" {
		return nil, fmt.Errorf("Unsplash Cookie required (login at unsplash.com and paste cookie in plugin settings)")
	}

	switch {
	case req.Route == "/unsplash/topics" || strings.HasPrefix(req.Route, "/unsplash/topics"):
		topic := strings.TrimSpace(req.Params["topic"])
		if topic == "" {
			topic = topicFromChannel(req.ChannelID)
		}
		if topic == "" {
			return nil, fmt.Errorf("missing topic parameter")
		}
		return fetchTopicPhotos(topic, req.Params, cookie, req.ChannelID)
	case req.Route == "/unsplash/search" || strings.HasPrefix(req.Route, "/unsplash/search"):
		query := strings.TrimSpace(req.Params["query"])
		if query == "" {
			return nil, fmt.Errorf("missing query parameter")
		}
		return fetchSearch(query, req.Params, cookie, req.ChannelID)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type unsplashPhoto struct {
	ID             string  `json:"id"`
	CreatedAt      string  `json:"created_at"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Description    *string `json:"description"`
	AltDescription string  `json:"alt_description"`
	Likes          int     `json:"likes"`
	Premium        bool    `json:"premium"`
	Plus           bool    `json:"plus"`
	Slug           string  `json:"slug"`
	URLs           struct {
		Regular string `json:"regular"`
		Small   string `json:"small"`
		Thumb   string `json:"thumb"`
		Full    string `json:"full"`
	} `json:"urls"`
	Links struct {
		HTML string `json:"html"`
	} `json:"links"`
	User struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"user"`
	TopicSubmissions map[string]struct {
		Status string `json:"status"`
	} `json:"topic_submissions"`
}

type searchResponse struct {
	Total      int             `json:"total"`
	TotalPages int             `json:"total_pages"`
	Results    []unsplashPhoto `json:"results"`
}

func fetchTopicPhotos(topic string, params map[string]string, cookie, channelID string) (*sdk.FeedResult, error) {
	page := pageNum(params)
	perPage := perPageNum(params)

	query := url.Values{
		"page":     {strconv.Itoa(page)},
		"per_page": {strconv.Itoa(perPage)},
	}
	for key, value := range params {
		if reservedParams[key] || strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}

	rawURL := fmt.Sprintf("%s/topics/%s/photos?%s", napiBase, url.PathEscape(topic), query.Encode())
	body, status, err := host.HTTPGet(rawURL, napiTopicHeaders(topic, cookie))
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	photos, err := parseTopicPhotos(body)
	if err != nil {
		return nil, err
	}

	title := channelLabels[channelID]
	if title == "" {
		title = channelLabels[topic]
	}
	if title == "" {
		title = fmt.Sprintf("Unsplash · %s", topic)
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: fmt.Sprintf("Unsplash 主题「%s」", topic),
		Items:       photosToItems(photos),
		HasMore:     len(photos) >= perPage,
	}, nil
}

func fetchSearch(queryText string, params map[string]string, cookie, channelID string) (*sdk.FeedResult, error) {
	page := pageNum(params)
	perPage := perPageNum(params)

	query := url.Values{
		"query":    {queryText},
		"page":     {strconv.Itoa(page)},
		"per_page": {strconv.Itoa(perPage)},
	}
	for key, value := range params {
		if reservedParams[key] || strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}

	rawURL := napiBase + "/search/photos?" + query.Encode()
	body, status, err := host.HTTPGet(rawURL, napiSearchHeaders(queryText, cookie))
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse search response failed: %w", err)
	}

	title := channelLabels[channelID]
	if title == "" {
		title = "Unsplash 搜索"
	}

	hasMore := len(resp.Results) >= perPage
	if resp.TotalPages > 0 {
		hasMore = page < resp.TotalPages
	}

	desc := fmt.Sprintf("关键词「%s」", queryText)
	if resp.Total > 0 {
		desc = fmt.Sprintf("%s · 共 %d 条结果", desc, resp.Total)
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: desc,
		Items:       photosToItems(resp.Results),
		HasMore:     hasMore,
	}, nil
}

func napiTopicHeaders(topic, cookie string) map[string]string {
	return napiHeaders(cookie, "https://unsplash.com/t/"+topic)
}

func napiSearchHeaders(queryText, cookie string) map[string]string {
	refererQuery := strings.ReplaceAll(queryText, " ", "-")
	return napiHeaders(cookie, "https://unsplash.com/s/photos/"+url.PathEscape(refererQuery))
}

// These headers mirror the browser request currently accepted by Unsplash's
// NAPI endpoint, including the mobile client fingerprint used by the cookie.
func napiHeaders(cookie, referer string) map[string]string {
	return map[string]string{
		"Accept":             "*/*",
		"Accept-Language":    "en-US",
		"Cache-Control":      "no-cache",
		"Client-Geo-Region":  "global",
		"Cookie":             cookie,
		"Priority":           "u=1, i",
		"Pragma":             "no-cache",
		"Referer":            referer,
		"Sec-CH-UA":          `"Not=A?Brand";v="99", "Google Chrome";v="151", "Chromium";v="151"`,
		"Sec-CH-UA-Mobile":   "?1",
		"Sec-CH-UA-Platform": `"Android"`,
		"Sec-Fetch-Dest":     "empty",
		"Sec-Fetch-Mode":     "cors",
		"Sec-Fetch-Site":     "same-origin",
		"User-Agent":         userAgent,
		"X-Client-Version":   clientVersion,
	}
}

func parseTopicPhotos(body []byte) ([]unsplashPhoto, error) {
	var photos []unsplashPhoto
	if err := json.Unmarshal(body, &photos); err != nil {
		return nil, fmt.Errorf("parse topic response failed: %w", err)
	}
	return photos, nil
}

func photosToItems(photos []unsplashPhoto) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, len(photos))
	for _, photo := range photos {
		imageURL := pickImageURL(photo)
		title := photoTitle(photo)
		author := strings.TrimSpace(photo.User.Name)
		if author == "" {
			author = photo.User.Username
		}

		items = append(items, sdk.FeedItem{
			ID:          "unsplash-" + photo.ID,
			Title:       title,
			URL:         photo.Links.HTML,
			Summary:     photoSummary(photo),
			Author:      author,
			Cover:       imageURL,
			Image:       imageURL,
			PublishedAt: photo.CreatedAt,
			Tags:        photoTags(photo),
		})
	}
	return items
}

func photoTitle(photo unsplashPhoto) string {
	if photo.Description != nil {
		if title := strings.TrimSpace(*photo.Description); title != "" {
			return title
		}
	}
	if title := strings.TrimSpace(photo.AltDescription); title != "" {
		return title
	}
	if photo.Slug != "" {
		return photo.Slug
	}
	return "Unsplash Photo"
}

func photoSummary(photo unsplashPhoto) string {
	parts := []string{
		fmt.Sprintf("%dx%d", photo.Width, photo.Height),
		fmt.Sprintf("赞 %d", photo.Likes),
	}
	if photo.Premium || photo.Plus {
		parts = append(parts, "Unsplash+")
	}
	return strings.Join(parts, " · ")
}

func photoTags(photo unsplashPhoto) []string {
	tags := make([]string, 0, len(photo.TopicSubmissions))
	for topic, submission := range photo.TopicSubmissions {
		if submission.Status == "approved" {
			tags = append(tags, topic)
		}
	}
	return tags
}

func pickImageURL(photo unsplashPhoto) string {
	if u := strings.TrimSpace(photo.URLs.Regular); u != "" {
		return u
	}
	if u := strings.TrimSpace(photo.URLs.Small); u != "" {
		return u
	}
	if u := strings.TrimSpace(photo.URLs.Full); u != "" {
		return u
	}
	return strings.TrimSpace(photo.URLs.Thumb)
}

func topicFromChannel(channelID string) string {
	if strings.HasPrefix(channelID, "topic-") {
		return strings.TrimPrefix(channelID, "topic-")
	}
	return channelID
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
	if err != nil || size < 1 {
		return defaultPerPage
	}
	if size > maxPerPage {
		return maxPerPage
	}
	return size
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
