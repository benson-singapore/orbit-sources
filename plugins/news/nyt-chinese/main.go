package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&NYTChinesePlugin{})
}

type NYTChinesePlugin struct{}

const (
	listBaseURL    = "https://m.cn.nytimes.com"
	articleBaseURL = "https://cn.nytimes.com"
	defaultUA      = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"
)

var sectionLabels = map[string]string{
	"home":        "首页",
	"world":       "国际",
	"china":       "中国",
	"business":    "商业与经济",
	"lens":        "镜头",
	"technology":  "科技",
	"science":     "科学",
	"health":      "健康",
	"education":   "教育",
	"culture":     "文化",
	"style":       "风尚",
	"travel":      "旅游",
	"real-estate": "房地产",
	"opinion":     "观点与评论",
}

var dateInURL = regexp.MustCompile(`/(\d{8})/`)

func (p *NYTChinesePlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/nyt-chinese/list":
		section := strings.TrimSpace(req.Params["section"])
		if section == "" {
			section = "home"
		}
		return fetchList(section)
	case req.Route == "/nyt-chinese/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(section string) (*sdk.FeedResult, error) {
	label, ok := sectionLabels[section]
	if !ok {
		return nil, fmt.Errorf("unknown section: %s", section)
	}

	fetchURL := listBaseURL + "/"
	if section != "home" {
		fetchURL = listBaseURL + "/" + section + "/"
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

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse list page: %w", err)
	}

	items := parseListItems(doc)
	if len(items) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	return &sdk.FeedResult{
		Title:       fmt.Sprintf("纽约时报中文网 - %s", label),
		Description: "纽约时报中文网面向全球华语读者的新闻报道",
		Items:       items,
	}, nil
}

func parseListItems(doc *goquery.Document) []sdk.FeedItem {
	seen := make(map[string]struct{})
	var items []sdk.FeedItem

	doc.Find("li.regular-item a, li.photospot-item a, li.site-title a").Each(func(_ int, link *goquery.Selection) {
		href, _ := link.Attr("href")
		href = strings.TrimSpace(href)
		if href == "" || !strings.Contains(href, "cn.nytimes.com/") {
			return
		}

		title := extractListTitle(link, href)
		if title == "" {
			return
		}

		articleURL := normalizeArticleURL(href)
		id := articleIDFromURL(articleURL)
		if id == "" {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}

		summary := strings.TrimSpace(link.Find("p.summary").First().Text())
		cover := strings.TrimSpace(link.Find("figure img").First().AttrOr("src", ""))

		items = append(items, sdk.FeedItem{
			ID:          id,
			Title:       title,
			URL:         articleURL,
			Summary:     summary,
			Cover:       cover,
			Image:       cover,
			PublishedAt: publishedAtFromURL(articleURL),
		})
	})

	return items
}

func extractListTitle(link *goquery.Selection, href string) string {
	title := strings.TrimSpace(link.Find("h2 span").First().Text())
	if title == "" {
		title = strings.TrimSpace(link.AttrOr("title", ""))
	}
	if title == "" {
		title = strings.TrimSpace(link.Find("figure img").First().AttrOr("alt", ""))
	}
	if title == "" {
		title = titleFromURL(href)
	}
	return title
}

func titleFromURL(raw string) string {
	id := articleIDFromURL(raw)
	if id == "" {
		return ""
	}
	parts := strings.Split(id, "/")
	slug := parts[len(parts)-1]
	slug = strings.ReplaceAll(slug, "-", " ")
	return strings.TrimSpace(slug)
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	articleURL := normalizeArticleURL(id)
	item, err := fetchArticleDetails(articleURL)
	if err != nil {
		return nil, err
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{*item},
	}, nil
}

func fetchArticleDetails(articleURL string) (*sdk.FeedItem, error) {
	body, status, err := host.HTTPGet(articleURL, map[string]string{
		"User-Agent": defaultUA,
		"Accept":     "text/html,application/xhtml+xml",
	})
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

	title := strings.TrimSpace(doc.Find("meta[property=\"og:title\"]").AttrOr("content", ""))
	if title == "" {
		title = strings.TrimSpace(doc.Find("h1").First().Text())
	}
	if title == "" {
		return nil, fmt.Errorf("no title found")
	}

	publishedAt := parsePublishedAt(
		doc.Find("meta[property=\"article:published_time\"]").AttrOr("content", ""),
		doc.Find("time").AttrOr("datetime", ""),
		articleURL,
	)

	cover := strings.TrimSpace(doc.Find("meta[property=\"og:image\"]").AttrOr("content", ""))
	summary := strings.TrimSpace(doc.Find("meta[property=\"og:description\"]").AttrOr("content", ""))
	if summary == "" {
		summary = strings.TrimSpace(doc.Find("meta[name=\"description\"]").AttrOr("content", ""))
	}

	content := renderArticleContent(doc)
	item := &sdk.FeedItem{
		ID:          articleIDFromURL(articleURL),
		Title:       title,
		URL:         articleURL,
		Summary:     summary,
		Cover:       cover,
		Image:       cover,
		Content:     content,
		PublishedAt: publishedAt,
	}
	if item.ID == "" {
		item.ID = articleURL
	}
	return item, nil
}

func renderArticleContent(doc *goquery.Document) string {
	body := doc.Find(".article-body").First()
	if body.Length() == 0 {
		return ""
	}

	var sb strings.Builder
	body.Find(".article-paragraph").Each(func(_ int, para *goquery.Selection) {
		figure := para.Find("figure").First()
		if figure.Length() > 0 {
			img := figure.Find("img").First()
			src := strings.TrimSpace(img.AttrOr("src", ""))
			if src == "" {
				src = strings.TrimSpace(img.AttrOr("data-src", ""))
			}
			if src == "" {
				return
			}
			alt := escapeHTML(strings.TrimSpace(img.AttrOr("alt", "")))
			caption := strings.TrimSpace(figure.Find("figcaption span").First().Text())
			sb.WriteString(fmt.Sprintf("<figure><img src=\"%s\" alt=\"%s\" style=\"max-width:100%%;\"/>", src, alt))
			if caption != "" {
				sb.WriteString(fmt.Sprintf("<figcaption>%s</figcaption>", escapeHTML(caption)))
			}
			sb.WriteString("</figure>\n")
			return
		}

		text := strings.TrimSpace(para.Text())
		if text == "" {
			return
		}

		if para.Find("a").Length() > 0 {
			inner, _ := para.Html()
			sb.WriteString(fmt.Sprintf("<p>%s</p>\n", inner))
			return
		}

		sb.WriteString(fmt.Sprintf("<p>%s</p>\n", escapeHTML(text)))
	})

	return sb.String()
}

func normalizeArticleURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	if strings.HasPrefix(raw, "http") {
		if u, err := url.Parse(raw); err == nil {
			u.RawQuery = ""
			u.Fragment = ""
			path := strings.TrimSuffix(u.Path, "/") + "/"
			return u.Scheme + "://" + u.Host + path
		}
		return raw
	}

	raw = strings.TrimPrefix(raw, "/")
	return articleBaseURL + "/" + raw + "/"
}

func articleIDFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		raw = u.Path
	}

	raw = strings.Trim(raw, "/")
	parts := strings.Split(raw, "/")
	if len(parts) < 3 {
		return ""
	}

	// {section}/{YYYYMMDD}/{slug}
	if len(parts[0]) < 2 || len(parts[1]) != 8 {
		return ""
	}
	for _, ch := range parts[1] {
		if ch < '0' || ch > '9' {
			return ""
		}
	}

	return strings.Join(parts, "/")
}

func publishedAtFromURL(articleURL string) string {
	matches := dateInURL.FindStringSubmatch(articleURL)
	if len(matches) < 2 {
		return ""
	}
	if t, err := time.Parse("20060102", matches[1]); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	return ""
}

func parsePublishedAt(values ...string) string {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		for _, format := range formats {
			if t, err := time.Parse(format, raw); err == nil {
				return t.UTC().Format(time.RFC3339)
			}
		}
	}

	for _, raw := range values {
		if ts := publishedAtFromURL(raw); ts != "" {
			return ts
		}
	}

	return time.Now().UTC().Format(time.RFC3339)
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return replacer.Replace(s)
}
