package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL     = "https://www.toutiao.com"
	feedAPI     = "https://www.toutiao.com/api/pc/feed/"
	hotBoardAPI = "https://www.toutiao.com/hot-event/hot-board/?origin=toutiao_pc"
	mobileInfo  = "https://m.toutiao.com/i%s/info/"
	feedAS      = "A1A5A5A5A5A5A5A5A5A5A5A5A5A5A5A5A5A5A5"
	feedCP      = "5A5A5A5A5A5A5A5A5A5A5A5A5A5A5A5A5A5A5"
)

func main() {
	sdk.Run(&ToutiaoPlugin{})
}

type ToutiaoPlugin struct{}

var categoryLabelMap = map[string]string{
	"news_hot":          "要闻",
	"news_tech":         "科技",
	"news_finance":      "财经",
	"news_sports":       "体育",
	"news_entertainment": "娱乐",
}

func (p *ToutiaoPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/toutiao/feed":
		return fetchCategoryFeed(req.Params)
	case req.Route == "/toutiao/hot-board":
		return fetchHotBoard()
	case req.Route == "/toutiao/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type feedResponse struct {
	HasMore bool        `json:"has_more"`
	Message string      `json:"message"`
	Data    []feedEntry `json:"data"`
}

type feedEntry struct {
	Title            string `json:"title"`
	Abstract         string `json:"abstract"`
	Source           string `json:"source"`
	ImageURL         string `json:"image_url"`
	MiddleImage      string `json:"middle_image"`
	SourceURL        string `json:"source_url"`
	ItemID           string `json:"item_id"`
	GroupID          string `json:"group_id"`
	BehotTime        int64  `json:"behot_time"`
	ChineseTag       string `json:"chinese_tag"`
	ArticleGenre     string `json:"article_genre"`
	HasVideo         bool   `json:"has_video"`
	VideoDurationStr string `json:"video_duration_str"`
	CommentsCount    int    `json:"comments_count"`
	VideoPlayCount   int    `json:"video_play_count"`
	IsFeedAd         bool   `json:"is_feed_ad"`
	MediaAvatarURL   string `json:"media_avatar_url"`
}

type hotBoardResponse struct {
	Data []hotBoardEntry `json:"data"`
}

type hotBoardEntry struct {
	ClusterIDStr string `json:"ClusterIdStr"`
	Title        string `json:"Title"`
	URL          string `json:"Url"`
	HotValue     string `json:"HotValue"`
	Label        string `json:"Label"`
	QueryWord    string `json:"QueryWord"`
	Image        struct {
		URL string `json:"url"`
	} `json:"Image"`
}

type mobileDetailResponse struct {
	Data mobileArticle `json:"data"`
}

type mobileArticle struct {
	GID          string `json:"gid"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	BizTag       string `json:"biz_tag"`
	Source       string `json:"source"`
	DetailSource string `json:"detail_source"`
	PublishTime  string `json:"publish_time"`
	URL          string `json:"url"`
	PosterURL    string `json:"poster_url"`
	CommentCount int    `json:"comment_count"`
	DiggCount    int    `json:"digg_count"`
	MediaUser    struct {
		ScreenName string `json:"screen_name"`
		AvatarURL  string `json:"avatar_url"`
	} `json:"media_user"`
}

func fetchCategoryFeed(params map[string]string) (*sdk.FeedResult, error) {
	category := strings.TrimSpace(params["category"])
	if category == "" {
		return nil, fmt.Errorf("missing category parameter")
	}

	label := categoryLabelMap[category]
	if label == "" {
		label = category
	}

	query := url.Values{
		"category":           {category},
		"max_behot_time":     {"0"},
		"max_behot_time_tmp": {"0"},
		"tadrequire":         {"true"},
		"as":                 {feedAS},
		"cp":                 {feedCP},
	}

	body, status, err := host.HTTPGet(feedAPI+"?"+query.Encode(), defaultHeaders())
	if err != nil {
		return nil, fmt.Errorf("fetch feed failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("feed http status %d", status)
	}

	var resp feedResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse feed response: %w", err)
	}
	if resp.Message != "success" || len(resp.Data) == 0 {
		return nil, fmt.Errorf("empty feed data")
	}

	items := make([]sdk.FeedItem, 0, len(resp.Data))
	for _, entry := range resp.Data {
		if entry.IsFeedAd {
			continue
		}
		item := feedEntryToItem(entry)
		if item.Title == "" || item.ID == "" {
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no valid items in feed")
	}

	return &sdk.FeedResult{
		Title:       fmt.Sprintf("今日头条 · %s", label),
		Description: "今日头条分类资讯",
		Items:       items,
	}, nil
}

func fetchHotBoard() (*sdk.FeedResult, error) {
	body, status, err := host.HTTPGet(hotBoardAPI, defaultHeaders())
	if err != nil {
		return nil, fmt.Errorf("fetch hot board failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("hot board http status %d", status)
	}

	var resp hotBoardResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse hot board response: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("empty hot board data")
	}

	items := make([]sdk.FeedItem, 0, len(resp.Data))
	for _, entry := range resp.Data {
		item := hotBoardEntryToItem(entry)
		if item.Title == "" || item.ID == "" {
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no valid hot board items")
	}

	return &sdk.FeedResult{
		Title:       "今日头条 · 热榜",
		Description: "今日头条实时热搜",
		Items:       items,
	}, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	detailURL := fmt.Sprintf(mobileInfo, id)
	body, status, err := host.HTTPGet(detailURL, mobileHeaders())
	if err != nil {
		return nil, fmt.Errorf("fetch detail failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("detail http status %d", status)
	}

	var resp mobileDetailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse detail response: %w", err)
	}

	article := resp.Data
	articleID := strings.TrimSpace(article.GID)
	if articleID == "" {
		articleID = id
	}
	title := strings.TrimSpace(article.Title)
	if title == "" {
		return nil, fmt.Errorf("article not found")
	}

	author := strings.TrimSpace(article.MediaUser.ScreenName)
	if author == "" {
		author = strings.TrimSpace(article.Source)
	}
	if author == "" {
		author = strings.TrimSpace(article.DetailSource)
	}

	cover := strings.TrimSpace(article.PosterURL)
	content := strings.TrimSpace(article.Content)
	itemURL := articleURL(articleID, article.URL)
	if strings.TrimSpace(article.BizTag) == "热榜事件" {
		itemURL = trendingURL(articleID, article.URL)
	}
	if content == "" {
		content = fmt.Sprintf(
			`<p>该热榜话题暂无独立正文，请<a href="%s" target="_blank" rel="noopener">在今日头条查看相关资讯</a>。</p>`,
			itemURL,
		)
	} else if cover != "" && !strings.Contains(content, cover) {
		content = fmt.Sprintf(`<img src="%s" style="max-width:100%%;border-radius:8px;margin-bottom:1rem;"/>`, cover) + "\n" + content
	}

	summary := textSummary(content, 200)
	if summary == "" {
		summary = title
	}

	item := sdk.FeedItem{
		ID:           articleID,
		Title:        title,
		URL:          itemURL,
		Summary:      summary,
		Content:      content,
		Author:       author,
		Cover:        cover,
		Image:        cover,
		AuthorAvatar: strings.TrimSpace(article.MediaUser.AvatarURL),
		PublishedAt:  parsePublishTime(article.PublishTime),
	}
	if article.DiggCount > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("点赞 %d", article.DiggCount))
	}
	if article.CommentCount > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("评论 %d", article.CommentCount))
	}
	if tag := strings.TrimSpace(article.BizTag); tag != "" {
		item.Tags = append(item.Tags, tag)
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func feedEntryToItem(entry feedEntry) sdk.FeedItem {
	id := strings.TrimSpace(entry.ItemID)
	if id == "" {
		id = strings.TrimSpace(entry.GroupID)
	}

	cover := normalizeImageURL(entry.MiddleImage)
	if cover == "" {
		cover = normalizeImageURL(entry.ImageURL)
	}

	summary := strings.TrimSpace(entry.Abstract)
	if summary == "" {
		summary = strings.TrimSpace(entry.Title)
	}

	item := sdk.FeedItem{
		ID:           id,
		Title:        strings.TrimSpace(entry.Title),
		URL:          articleURL(id, entry.SourceURL),
		Summary:      summary,
		Author:       strings.TrimSpace(entry.Source),
		Cover:        cover,
		Image:        cover,
		AuthorAvatar: strings.TrimSpace(entry.MediaAvatarURL),
		PublishedAt:  unixToRFC3339(entry.BehotTime),
	}

	if tag := strings.TrimSpace(entry.ChineseTag); tag != "" {
		item.Tags = append(item.Tags, tag)
	}
	if entry.HasVideo || entry.ArticleGenre == "video" {
		item.Kind = "long"
		if entry.VideoDurationStr != "" {
			item.Tags = append(item.Tags, entry.VideoDurationStr)
		}
	}
	if entry.CommentsCount > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("评论 %d", entry.CommentsCount))
	}
	if entry.VideoPlayCount > 0 {
		item.Tags = append(item.Tags, fmt.Sprintf("播放 %d", entry.VideoPlayCount))
	}

	return item
}

func hotBoardEntryToItem(entry hotBoardEntry) sdk.FeedItem {
	id := strings.TrimSpace(entry.ClusterIDStr)
	cover := strings.TrimSpace(entry.Image.URL)

	item := sdk.FeedItem{
		ID:          id,
		Title:       strings.TrimSpace(entry.Title),
		URL:         trendingURL(id, entry.URL),
		Summary:     strings.TrimSpace(entry.QueryWord),
		Cover:       cover,
		Image:       cover,
		PublishedAt: time.Now().Format(time.RFC3339),
	}

	if entry.HotValue != "" {
		item.Tags = append(item.Tags, fmt.Sprintf("热度 %s", entry.HotValue))
	}
	if label := strings.TrimSpace(entry.Label); label != "" {
		item.Tags = append(item.Tags, label)
	}

	return item
}

func articleURL(id, rawURL string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return fmt.Sprintf("%s/article/%s/", baseURL, id)
	}
	return normalizePageURL(rawURL)
}

func trendingURL(id, rawURL string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return fmt.Sprintf("%s/trending/%s/", baseURL, id)
	}
	return normalizePageURL(rawURL)
}

func normalizePageURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return baseURL + "/"
	}
	if strings.HasPrefix(rawURL, "http") {
		return rawURL
	}
	if strings.HasPrefix(rawURL, "/") {
		return baseURL + rawURL
	}
	return baseURL + "/" + rawURL
}

func normalizeImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	return raw
}

func parsePublishTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now().Format(time.RFC3339)
	}
	ts, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || ts <= 0 {
		return time.Now().Format(time.RFC3339)
	}
	return time.Unix(ts, 0).Format(time.RFC3339)
}

func unixToRFC3339(ts int64) string {
	if ts <= 0 {
		return time.Now().Format(time.RFC3339)
	}
	return time.Unix(ts, 0).Format(time.RFC3339)
}

func textSummary(html string, maxLen int) string {
	text := strings.TrimSpace(stripHTML(html))
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "…"
}

func stripHTML(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func defaultHeaders() map[string]string {
	return map[string]string{
		"Accept":     "application/json, text/plain, */*",
		"Referer":    baseURL + "/",
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

func mobileHeaders() map[string]string {
	return map[string]string{
		"Accept":     "application/json, text/plain, */*",
		"Referer":    baseURL + "/",
		"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
	}
}
