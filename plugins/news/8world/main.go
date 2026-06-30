package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&EightWorldPlugin{})
}

type EightWorldPlugin struct{}

const (
	baseURL   = "https://www.8world.com"
	defaultUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var sectionPaths = map[string]string{
	"realtime":       "/realtime",
	"singapore":      "/singapore",
	"southeast-asia": "/southeast-asia",
	"greater-china":  "/greater-china",
	"world":          "/world",
	"finance":        "/finance",
	"sports":         "/sports",
}

var sectionLabels = map[string]string{
	"realtime":       "即时",
	"singapore":      "新加坡",
	"southeast-asia": "东南亚",
	"greater-china":  "中港台",
	"world":          "国际",
	"finance":        "财经",
	"sports":         "体育",
}

var reDrupalSettings = regexp.MustCompile(`data-drupal-selector="drupal-settings-json">(.*?)</script>`)
var reJSONLD = regexp.MustCompile(`<script type="application/ld\+json">(.*?)</script>`)
var reNumericID = regexp.MustCompile(`^\d+$`)

func (p *EightWorldPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/8world/list":
		return fetchList(req.Params)
	case req.Route == "/8world/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(params map[string]string) (*sdk.FeedResult, error) {
	section := strings.TrimSpace(params["section"])
	if section == "" {
		section = "singapore"
	}
	path, ok := sectionPaths[section]
	if !ok {
		return nil, fmt.Errorf("unknown section: %s", section)
	}

	page := parsePage(params["page"])
	fetchURL := baseURL + path
	if page > 0 {
		fetchURL = fmt.Sprintf("%s?page=%d", fetchURL, page)
	}

	body, status, err := host.HTTPGet(fetchURL, map[string]string{
		"User-Agent": defaultUA,
		"Accept":     "text/html,application/xhtml+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch list page failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("list page http status %d", status)
	}

	html := string(body)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse list page: %w", err)
	}

	label := sectionLabels[section]
	if label == "" {
		label = section
	}

	items := parseListArticles(doc)
	if len(items) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("8视界 - %s", label),
		Description: "新加坡最值得信赖的中文新闻平台",
		Items:       items,
	}

	if hasNextPage(doc) {
		result.HasMore = true
		result.Next = map[string]string{
			"page": strconv.Itoa(page + 1),
		}
	}

	return result, nil
}

func parseListArticles(doc *goquery.Document) []sdk.FeedItem {
	seen := make(map[string]struct{})
	var items []sdk.FeedItem

	doc.Find(`article.article.contour`).Each(func(_ int, card *goquery.Selection) {
		if isVideoListCard(card) {
			return
		}

		link := card.Find("a.article-link").First()
		href, _ := link.Attr("href")
		href = strings.TrimSpace(href)
		if href == "" || isVideoArticlePath(href) {
			return
		}

		title := strings.TrimSpace(link.Find("span").First().Text())
		if title == "" {
			title = strings.TrimSpace(link.Text())
		}
		if title == "" {
			if alt, ok := card.Find("img").First().Attr("alt"); ok {
				title = strings.TrimSpace(alt)
			}
		}
		if title == "" {
			return
		}

		articleURL := normalizeArticleURL(href)
		if _, exists := seen[articleURL]; exists {
			return
		}
		seen[articleURL] = struct{}{}

		cover, _ := card.Find("img.image, img").First().Attr("src")
		publishedAt := parseListTime(card.Find("time").First().AttrOr("datetime", ""))

		items = append(items, sdk.FeedItem{
			ID:          articleURL,
			Title:       title,
			URL:         articleURL,
			Cover:       strings.TrimSpace(cover),
			Image:       strings.TrimSpace(cover),
			PublishedAt: publishedAt,
		})
	})

	return items
}

func isVideoListCard(card *goquery.Selection) bool {
	media, _ := card.Attr("data-media")
	return strings.EqualFold(strings.TrimSpace(media), "Video")
}

func isVideoArticlePath(path string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	return strings.HasPrefix(path, "/videos/") || strings.HasPrefix(path, "/in-depth/")
}

func hasNextPage(doc *goquery.Document) bool {
	return doc.Find(`.pager__link--next, .pager__item--next a[rel="next"]`).Length() > 0
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	articleURL, err := resolveArticleURL(id)
	if err != nil {
		return nil, err
	}
	if isVideoArticlePath(strings.TrimPrefix(articleURL, baseURL)) {
		return nil, fmt.Errorf("video article is not supported")
	}

	body, status, err := host.HTTPGet(articleURL, map[string]string{
		"User-Agent": defaultUA,
		"Accept":     "text/html,application/xhtml+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch article failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("article http status %d", status)
	}

	html := string(body)
	if isVideoArticlePage(html) {
		return nil, fmt.Errorf("video article is not supported")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse article page: %w", err)
	}

	item, err := pageToFeedItem(articleURL, html, doc)
	if err != nil {
		return nil, err
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{*item},
	}, nil
}

func isVideoArticlePage(html string) bool {
	if strings.Contains(html, `data-video-id="`) && strings.Contains(html, "<video-js") {
		return true
	}
	if match := reDrupalSettings.FindStringSubmatch(html); len(match) > 1 {
		var settings map[string]json.RawMessage
		if err := json.Unmarshal([]byte(match[1]), &settings); err == nil {
			if raw, ok := settings["videoad"]; ok && len(raw) > 0 && string(raw) != "null" && string(raw) != "[]" {
				return true
			}
		}
	}
	return false
}

type jsonLDGraph struct {
	Context any `json:"@context"`
	Graph   []jsonLDItem `json:"@graph"`
}

type jsonLDItem struct {
	Type        string   `json:"@type"`
	Headline    string   `json:"headline"`
	Description string   `json:"description"`
	DatePub     string   `json:"datePublished"`
	Image       any      `json:"image"`
}

func pageToFeedItem(articleURL, html string, doc *goquery.Document) (*sdk.FeedItem, error) {
	meta := extractNewsArticleJSONLD(html)

	title := strings.TrimSpace(doc.Find("h1 span, h1").First().Text())
	if title == "" {
		title = meta.Headline
	}
	if title == "" {
		title = doc.Find(`meta[property="og:title"]`).AttrOr("content", "")
	}
	if title == "" {
		return nil, fmt.Errorf("no title found")
	}

	summary := meta.Description
	if summary == "" {
		summary = doc.Find(`meta[property="og:description"]`).AttrOr("content", "")
	}

	publishedAt := meta.DatePub
	if publishedAt == "" {
		publishedAt = parseListTime(doc.Find("time").First().AttrOr("datetime", ""))
	}
	if publishedAt == "" {
		publishedAt = time.Now().UTC().Format(time.RFC3339)
	}

	cover := firstImage(meta.Image)
	if cover == "" {
		cover = doc.Find(`meta[property="og:image"]`).AttrOr("content", "")
	}

	contentHTML := extractArticleContent(doc)
	content := buildContent(cover, summary, contentHTML)

	return &sdk.FeedItem{
		ID:          articleURL,
		Title:       title,
		URL:         articleURL,
		Summary:     summary,
		Cover:       cover,
		Image:       cover,
		Content:     content,
		PublishedAt: publishedAt,
	}, nil
}

func extractNewsArticleJSONLD(html string) jsonLDItem {
	var best jsonLDItem
	for _, match := range reJSONLD.FindAllStringSubmatch(html, -1) {
		if len(match) < 2 {
			continue
		}

		var graph jsonLDGraph
		if err := json.Unmarshal([]byte(match[1]), &graph); err == nil && len(graph.Graph) > 0 {
			for _, item := range graph.Graph {
				if item.Type == "NewsArticle" {
					return item
				}
			}
			continue
		}

		var item jsonLDItem
		if err := json.Unmarshal([]byte(match[1]), &item); err == nil && item.Type == "NewsArticle" {
			return item
		}
	}
	return best
}

func firstImage(raw any) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		for _, entry := range v {
			if s, ok := entry.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func extractArticleContent(doc *goquery.Document) string {
	content := doc.Find(".article-content").First()
	if content.Length() == 0 {
		return ""
	}

	clone := content.Clone()
	clone.Find("img[src*='news_mailing_graphic'], img[src*='news_app_graphic'], img[src*='news_telegram_graphic']").Remove()

	html, err := clone.Html()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(html)
}

func buildContent(cover, summary, contentHTML string) string {
	var sb strings.Builder
	if cover != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s" style="max-width:100%%;margin-bottom:1rem;"/>`+"\n", cover))
	}
	if summary != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>%s</strong></p>\n", escapeHTML(summary)))
	}
	if contentHTML != "" {
		sb.WriteString(contentHTML)
	}
	return sb.String()
}

func normalizeArticleURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if strings.HasPrefix(raw, "http") {
		return raw
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return baseURL + raw
}

func resolveArticleURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("missing id parameter")
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw, nil
	}

	if strings.HasPrefix(raw, "/") {
		return baseURL + raw, nil
	}

	if reNumericID.MatchString(raw) {
		return baseURL + "/node/" + raw, nil
	}

	if strings.Contains(raw, "/") {
		return normalizeArticleURL(raw), nil
	}

	return "", fmt.Errorf("invalid article id: %s", raw)
}

func parseListTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	formats := []string{
		time.RFC3339,
		"02/01/2006 15:04",
		"2006-01-02T15:04:05Z07:00",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, raw); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return ""
}

func parsePage(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(s)
}
