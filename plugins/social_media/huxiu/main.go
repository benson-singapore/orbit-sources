package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL        = "https://www.huxiu.com"
	listAPI        = "https://api-ms-article.huxiu.com/v1/channel/pcArticleList"
	detailAPI      = "https://api-web-article.huxiu.com/web/article/detail"
	defaultSize    = 12
	platform       = "www"
)

func main() {
	sdk.Run(&HuxiuPlugin{})
}

type HuxiuPlugin struct{}

var channelLabelMap = map[string]string{
	"105": "前沿资讯",
	"107": "国际热点",
	"114": "出海",
	"22":  "游戏娱乐",
}

func (p *HuxiuPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/huxiu/list":
		return fetchList(req.Params)
	case req.Route == "/huxiu/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type listResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Name     string        `json:"name"`
		LastID   int64         `json:"last_id"`
		Datalist []articleItem `json:"datalist"`
	} `json:"data"`
}

type articleItem struct {
	AID       int    `json:"aid"`
	Title     string `json:"title"`
	PicPath   string `json:"pic_path"`
	URL       string `json:"url"`
	Dateline  int64  `json:"dateline"`
	UserInfo  struct {
		Username string `json:"username"`
	} `json:"user_info"`
	CountInfo struct {
		FavoriteNum     int `json:"favorite_num"`
		TotalCommentNum int `json:"total_comment_num"`
	} `json:"count_info"`
}

type detailResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AID        string `json:"aid"`
		Title      string `json:"title"`
		PicPath    string `json:"pic_path"`
		URL        string `json:"url"`
		Author     string `json:"author"`
		Summary    string `json:"summary"`
		Content    string `json:"content"`
		Dateline   string `json:"dateline"`
		Fdateline  string `json:"fdateline"`
		FavNum     int    `json:"fav_num"`
		CommentNum int    `json:"total_comment_num"`
	} `json:"data"`
}

func fetchList(params map[string]string) (*sdk.FeedResult, error) {
	channelID := strings.TrimSpace(params["channel_id"])
	if channelID == "" {
		return nil, fmt.Errorf("missing channel_id parameter")
	}

	lastID := strings.TrimSpace(params["lastId"])
	pageSize := parsePositiveInt(params["size"], defaultSize)
	if pageSize > 30 {
		pageSize = 30
	}

	formBody := fmt.Sprintf(
		"platform=%s&channel_id=%s&last_id=%s&pagesize=%d",
		platform, channelID, lastID, pageSize,
	)

	body, status, err := host.HTTPPost(listAPI, defaultHeaders(), formBody)
	if err != nil {
		return nil, fmt.Errorf("http post failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("api returned success=false")
	}
	if len(resp.Data.Datalist) == 0 {
		return nil, fmt.Errorf("empty list data")
	}

	label := channelLabelMap[channelID]
	if label == "" {
		label = strings.TrimSpace(resp.Data.Name)
	}
	if label == "" {
		label = channelID
	}

	items := make([]sdk.FeedItem, 0, len(resp.Data.Datalist))
	for _, article := range resp.Data.Datalist {
		item := articleToFeedItem(article)
		if item.Title == "" {
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no valid items in list")
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("虎嗅 · %s", label),
		Description: "虎嗅网商业科技资讯",
		Items:       items,
	}

	if len(resp.Data.Datalist) >= pageSize && resp.Data.LastID > 0 {
		result.HasMore = true
		result.Next = map[string]string{
			"lastId": strconv.FormatInt(resp.Data.LastID, 10),
		}
	}

	return result, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	formBody := fmt.Sprintf("platform=%s&aid=%s", platform, id)

	body, status, err := host.HTTPPost(detailAPI, defaultHeaders(), formBody)
	if err != nil {
		return nil, fmt.Errorf("http post failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var resp detailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse detail response: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("api returned success=false")
	}

	article := resp.Data
	aid := strings.TrimSpace(article.AID)
	if aid == "" || strings.TrimSpace(article.Title) == "" {
		return nil, fmt.Errorf("article not found")
	}

	cover := strings.TrimSpace(article.PicPath)
	content := buildArticleContent(cover, article.Content)
	publishedAt := formatPublishedAt(parseUnixString(article.Dateline), article.Fdateline)

	item := sdk.FeedItem{
		ID:          aid,
		Title:       strings.TrimSpace(article.Title),
		URL:         articleURL(aid, article.URL),
		Summary:     strings.TrimSpace(article.Summary),
		Author:      strings.TrimSpace(article.Author),
		Cover:       cover,
		Image:       cover,
		Content:     content,
		PublishedAt: publishedAt,
	}

	if article.FavNum > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("收藏 %d", article.FavNum))
	}
	if article.CommentNum > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("评论 %d", article.CommentNum))
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func articleToFeedItem(article articleItem) sdk.FeedItem {
	cover := strings.TrimSpace(article.PicPath)

	item := sdk.FeedItem{
		ID:          strconv.Itoa(article.AID),
		Title:       strings.TrimSpace(article.Title),
		URL:         articleURL(strconv.Itoa(article.AID), article.URL),
		Author:      strings.TrimSpace(article.UserInfo.Username),
		Cover:       cover,
		Image:       cover,
		PublishedAt: unixToRFC3339(article.Dateline),
	}

	if article.CountInfo.FavoriteNum > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("收藏 %d", article.CountInfo.FavoriteNum))
	}
	if article.CountInfo.TotalCommentNum > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("评论 %d", article.CountInfo.TotalCommentNum))
	}

	return item
}

func buildArticleContent(cover, body string) string {
	var sb strings.Builder
	if cover != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s" style="max-width:100%%;border-radius:8px;margin-bottom:1rem;"/>`, cover))
		sb.WriteString("\n")
	}
	if body != "" {
		sb.WriteString(body)
	}
	return sb.String()
}

func articleURL(aid string, rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL != "" {
		if strings.Contains(rawURL, "m.huxiu.com") {
			return fmt.Sprintf("%s/article/%s.html?type=text", baseURL, aid)
		}
		return rawURL
	}
	return fmt.Sprintf("%s/article/%s.html?type=text", baseURL, aid)
}

func parseUnixString(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func formatPublishedAt(dateline int64, fdateline string) string {
	if dateline > 0 {
		return unixToRFC3339(dateline)
	}
	fdateline = strings.TrimSpace(fdateline)
	if fdateline == "" {
		return time.Now().Format(time.RFC3339)
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04", fdateline, time.Local); err == nil {
		return t.Format(time.RFC3339)
	}
	return time.Now().Format(time.RFC3339)
}

func unixToRFC3339(ts int64) string {
	if ts <= 0 {
		return time.Now().Format(time.RFC3339)
	}
	return time.Unix(ts, 0).Format(time.RFC3339)
}

func parsePositiveInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func defaultHeaders() map[string]string {
	return map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/x-www-form-urlencoded",
		"Origin":       baseURL,
		"Referer":      baseURL + "/",
		"User-Agent":   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
	}
}
