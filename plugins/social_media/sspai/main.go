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
	baseURL     = "https://sspai.com"
	cdnBase     = "https://cdnfile.sspai.com"
	defaultSize = 10
)

func main() {
	sdk.Run(&SspaiPlugin{})
}

type SspaiPlugin struct{}

var listTypeMap = map[string]struct {
	apiPath string
	label   string
}{
	"index":  {apiPath: "/api/v1/article/index/page/get", label: "推荐"},
	"matrix": {apiPath: "/api/v1/article/matrix/page/get", label: "全部"},
	"hot":    {apiPath: "/api/v1/article/hot/page/get", label: "热门"},
}

func (p *SspaiPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/sspai/list":
		listType := req.Params["type"]
		if listType == "" {
			listType = "index"
		}
		return fetchList(listType, req.Params)
	case req.Route == "/sspai/detail/:id":
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
	Error int           `json:"error"`
	Msg   string        `json:"msg"`
	Data  []articleItem `json:"data"`
	Total int           `json:"total"`
}

type articleItem struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Banner       string `json:"banner"`
	Summary      string `json:"summary"`
	CommentCount int    `json:"comment_count"`
	LikeCount    int    `json:"like_count"`
	ReleasedTime int64  `json:"released_time"`
	Author       struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	} `json:"author"`
	Tags []struct {
		Title string `json:"title"`
	} `json:"tags"`
}

type detailResponse struct {
	Error int `json:"error"`
	Data  struct {
		ID           int    `json:"id"`
		Title        string `json:"title"`
		Body         string `json:"body"`
		Summary      string `json:"summary"`
		Banner       string `json:"banner"`
		ReleasedTime int64  `json:"released_time"`
		Author       struct {
			Nickname string `json:"nickname"`
			Avatar   string `json:"avatar"`
		} `json:"author"`
		ArticleCount struct {
			CommentCount int `json:"comment_count"`
			LikeCount    int `json:"like_count"`
		} `json:"article_count"`
	} `json:"data"`
}

func fetchList(listType string, params map[string]string) (*sdk.FeedResult, error) {
	cfg, ok := listTypeMap[listType]
	if !ok {
		return nil, fmt.Errorf("unknown list type: %s", listType)
	}

	pageNum := parsePositiveInt(params["page"], 1)
	pageSize := parsePositiveInt(params["size"], defaultSize)
	if pageSize > 30 {
		pageSize = 30
	}
	offset := (pageNum - 1) * pageSize

	url := fmt.Sprintf(
		"%s%s?limit=%d&offset=%d&created_at=%d",
		baseURL, cfg.apiPath, pageSize, offset, time.Now().Unix(),
	)

	body, status, err := host.HTTPGet(url, defaultHeaders())
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}
	if resp.Error != 0 {
		return nil, fmt.Errorf("api error %d: %s", resp.Error, resp.Msg)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("empty list data")
	}

	items := make([]sdk.FeedItem, 0, len(resp.Data))
	for _, article := range resp.Data {
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
		Title:       fmt.Sprintf("少数派 · %s", cfg.label),
		Description: "少数派高质量科技生活内容",
		Items:       items,
	}

	if resp.Total > 0 {
		hasMore := offset+len(resp.Data) < resp.Total
		result.HasMore = hasMore
		if hasMore {
			result.Next = map[string]string{
				"page": strconv.Itoa(pageNum + 1),
			}
		}
	} else if len(resp.Data) == pageSize {
		result.HasMore = true
		result.Next = map[string]string{
			"page": strconv.Itoa(pageNum + 1),
		}
	}

	return result, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	url := fmt.Sprintf(
		"%s/api/v1/article/info/get?id=%s&support_webp=true&view=second",
		baseURL, id,
	)

	body, status, err := host.HTTPGet(url, defaultHeaders())
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var resp detailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse detail response: %w", err)
	}
	if resp.Error != 0 {
		return nil, fmt.Errorf("api error %d", resp.Error)
	}
	if resp.Data.ID == 0 || strings.TrimSpace(resp.Data.Title) == "" {
		return nil, fmt.Errorf("article not found")
	}

	article := resp.Data
	cover := normalizeImageURL(article.Banner)
	content := buildArticleContent(cover, article.Body)

	item := sdk.FeedItem{
		ID:          strconv.Itoa(article.ID),
		Title:       strings.TrimSpace(article.Title),
		URL:         articleURL(article.ID),
		Summary:     strings.TrimSpace(article.Summary),
		Author:      strings.TrimSpace(article.Author.Nickname),
		Cover:       cover,
		Image:       cover,
		Content:     content,
		PublishedAt: unixToRFC3339(article.ReleasedTime),
	}

	if article.ArticleCount.LikeCount > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("点赞 %d", article.ArticleCount.LikeCount))
	}
	if article.ArticleCount.CommentCount > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("评论 %d", article.ArticleCount.CommentCount))
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func articleToFeedItem(article articleItem) sdk.FeedItem {
	cover := normalizeImageURL(article.Banner)

	item := sdk.FeedItem{
		ID:          strconv.Itoa(article.ID),
		Title:       strings.TrimSpace(article.Title),
		URL:         articleURL(article.ID),
		Summary:     strings.TrimSpace(article.Summary),
		Author:      strings.TrimSpace(article.Author.Nickname),
		Cover:       cover,
		Image:       cover,
		PublishedAt: unixToRFC3339(article.ReleasedTime),
	}

	if article.LikeCount > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("点赞 %d", article.LikeCount))
	}
	if article.CommentCount > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("评论 %d", article.CommentCount))
	}
	for _, tag := range article.Tags {
		if title := strings.TrimSpace(tag.Title); title != "" {
			item.Tags = append(item.Tags, title)
		}
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

func articleURL(id int) string {
	return fmt.Sprintf("%s/post/%d", baseURL, id)
}

func normalizeImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return cdnBase + "/" + strings.TrimPrefix(raw, "/")
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
		"Accept":     "application/json",
		"Referer":    baseURL + "/",
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
	}
}
