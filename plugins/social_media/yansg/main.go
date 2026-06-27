package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const baseURL = "https://www.yan.sg"

var (
	reRelNext = regexp.MustCompile(`(?is)<link\s+rel=['"]next['"]\s+href=['"]([^'"]+)['"]`)

	categoryMap = map[string]string{
		"sgnews":             "新加坡",
		"chinanews":          "中国",
		"southeastasianews":  "东南亚",
	}

	categoryPathMap = map[string]string{
		"sgnews":            "/category/news/sgnews/",
		"chinanews":         "/category/news/chinanews/",
		"southeastasianews": "/category/news/southeastasianews/",
	}
)

func main() {
	sdk.Run(&YansgPlugin{})
}

type YansgPlugin struct{}

type listEntry struct {
	URL         string
	Title       string
	Cover       string
	Author      string
	Summary     string
	PublishedAt string
}

func (p *YansgPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/yansg/list":
		category := strings.TrimSpace(req.Params["category"])
		if category == "" {
			category = strings.TrimSpace(req.ChannelID)
		}
		if category == "singapore" {
			category = "sgnews"
		} else if category == "china" {
			category = "chinanews"
		} else if category == "southeast-asia" {
			category = "southeastasianews"
		}
		if _, ok := categoryPathMap[category]; !ok {
			return nil, fmt.Errorf("unknown category: %s", category)
		}
		page := pageNum(req.Params)
		return fetchList(category, page)
	case req.Route == "/yansg/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(category string, page int) (*sdk.FeedResult, error) {
	listURL := categoryListURL(category, page)
	body, status, err := httpGet(listURL)
	if err != nil {
		return nil, fmt.Errorf("fetch list page failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("list page http status %d", status)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse list page: %w", err)
	}

	entries := parseListEntries(doc, page <= 1)
	if len(entries) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	items := make([]sdk.FeedItem, 0, len(entries))
	for _, entry := range entries {
		items = append(items, entryToFeedItem(entry))
	}

	label := categoryMap[category]
	if label == "" {
		label = category
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("新加坡眼 · %s", label),
		Description: "新加坡眼即时新闻",
		Items:       items,
	}

	if hasNextPage(string(body)) {
		result.HasMore = true
		result.Next = map[string]string{
			"page":     strconv.Itoa(page + 1),
			"category": category,
		}
	}

	return result, nil
}

func parseListEntries(doc *goquery.Document, includeGallery bool) []listEntry {
	seen := make(map[string]struct{})
	var entries []listEntry

	addEntry := func(entry listEntry) {
		entry.URL = normalizeURL(entry.URL)
		if entry.URL == "" || entry.Title == "" {
			return
		}
		if _, ok := seen[entry.URL]; ok {
			return
		}
		seen[entry.URL] = struct{}{}
		entries = append(entries, entry)
	}

	if includeGallery {
		doc.Find(".td_block_inner .td-module-thumb a").Each(func(_ int, s *goquery.Selection) {
			addEntry(listEntry{
				URL:   attrOr(s, "href"),
				Title: attrOr(s, "title"),
				Cover: s.Find("img.entry-thumb").AttrOr("src", ""),
			})
		})
	}

	doc.Find(".td-ss-main-content .td_module_16").Each(func(_ int, s *goquery.Selection) {
		link := s.Find("h3.entry-title a").First()
		if link.Length() == 0 {
			link = s.Find(".td-module-thumb a").First()
		}

		entry := listEntry{
			URL:         normalizeURL(attrOr(link, "href")),
			Title:       strings.TrimSpace(link.AttrOr("title", link.Text())),
			Cover:       s.Find("img.entry-thumb").First().AttrOr("src", ""),
			Author:      strings.TrimSpace(s.Find(".td-post-author-name a").First().Text()),
			Summary:     strings.TrimSpace(s.Find(".td-excerpt").First().Text()),
			PublishedAt: parseDateTime(s.Find("time.entry-date").First().AttrOr("datetime", "")),
		}
		addEntry(entry)
	})

	return entries
}

func fetchDetail(rawID string) (*sdk.FeedResult, error) {
	articleURL := normalizeURL(rawID)
	if articleURL == "" {
		return nil, fmt.Errorf("invalid article id")
	}

	body, status, err := httpGet(articleURL)
	if err != nil {
		return nil, fmt.Errorf("fetch article page failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("article http status %d", status)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse article page: %w", err)
	}

	title := strings.TrimSpace(doc.Find("h1.entry-title").First().Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find("meta[property=\"og:title\"]").AttrOr("content", ""))
	}
	if title == "" {
		return nil, fmt.Errorf("no title found")
	}

	author := strings.TrimSpace(doc.Find(".td-post-title .td-post-author-name a").First().Text())
	if author == "" {
		author = strings.TrimSpace(doc.Find(".td-module-meta-info .td-post-author-name a").First().Text())
	}

	publishedAt := parseDateTime(doc.Find(".td-post-title time.entry-date").First().AttrOr("datetime", ""))
	if publishedAt == "" {
		publishedAt = parseDateTime(doc.Find("meta[property=\"article:published_time\"]").AttrOr("content", ""))
	}
	if publishedAt == "" {
		publishedAt = time.Now().Format(time.RFC3339)
	}

	cover := strings.TrimSpace(doc.Find("meta[property=\"og:image\"]").AttrOr("content", ""))
	if cover == "" {
		cover = strings.TrimSpace(doc.Find(".td-post-content img").First().AttrOr("src", ""))
	}

	summary := strings.TrimSpace(doc.Find("meta[property=\"og:description\"]").AttrOr("content", ""))

	contentHTML, _ := doc.Find(".td-post-content").First().Html()
	contentHTML = cleanArticleHTML(contentHTML)
	content := buildArticleContent(cover, summary, contentHTML)

	item := sdk.FeedItem{
		ID:          articleURL,
		Title:       title,
		URL:         articleURL,
		Summary:     summary,
		Author:      author,
		Cover:       cover,
		Image:       cover,
		Content:     content,
		PublishedAt: publishedAt,
	}

	if views := strings.TrimSpace(doc.Find(".td-post-views").First().Text()); views != "" {
		item.Tags = append(item.Tags, strings.TrimSpace(views))
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func cleanArticleHTML(html string) string {
	if strings.TrimSpace(html) == "" {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	doc.Find(".td-a-rec, .mp_profile_iframe_wrp, ul.list-paddingleft-1").Remove()

	doc.Find("section").Each(func(_ int, s *goquery.Selection) {
		style, _ := s.Attr("style")
		style = strings.ToLower(style)
		text := strings.TrimSpace(s.Text())

		switch {
		case strings.Contains(style, "text-align: center") && strings.Contains(text, "相关阅读"):
			s.Remove()
		case strings.Contains(style, "-webkit-tap-highlight-color"):
			s.Remove()
		case strings.Contains(style, "text-align: center") && strings.Contains(text, "新加坡眼"):
			s.Remove()
		}
	})

	result, _ := doc.Html()
	return strings.TrimSpace(result)
}

func buildArticleContent(cover, summary, content string) string {
	var sb strings.Builder
	if cover != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s" style="max-width:100%%;border-radius:8px;margin-bottom:1rem;"/>`, cover))
		sb.WriteString("\n")
	}
	if summary != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>%s</strong></p>\n", summary))
	}
	if content != "" {
		sb.WriteString(content)
	}
	return sb.String()
}

func entryToFeedItem(entry listEntry) sdk.FeedItem {
	publishedAt := entry.PublishedAt
	if publishedAt == "" {
		publishedAt = time.Now().Format(time.RFC3339)
	}

	return sdk.FeedItem{
		ID:          entry.URL,
		Title:       entry.Title,
		URL:         entry.URL,
		Summary:     entry.Summary,
		Author:      entry.Author,
		Cover:       entry.Cover,
		Image:       entry.Cover,
		PublishedAt: publishedAt,
	}
}

func categoryListURL(category string, page int) string {
	path := categoryPathMap[category]
	if page <= 1 {
		return baseURL + path
	}
	return fmt.Sprintf("%s%s/page/%d/", baseURL, strings.TrimSuffix(path, "/"), page)
}

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return baseURL + raw
}

func parseDateTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, raw); err == nil {
			return t.Format(time.RFC3339)
		}
	}
	return ""
}

func pageNum(params map[string]string) int {
	raw := strings.TrimSpace(params["page"])
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func hasNextPage(htmlBody string) bool {
	return reRelNext.MatchString(htmlBody)
}

func httpGet(url string) ([]byte, int, error) {
	return host.HTTPGet(url, map[string]string{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
		"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	})
}

func attrOr(s *goquery.Selection, key string) string {
	if s == nil || s.Length() == 0 {
		return ""
	}
	return strings.TrimSpace(s.AttrOr(key, ""))
}
