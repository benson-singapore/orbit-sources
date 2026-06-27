package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&ZaobaoPlugin{})
}

type ZaobaoPlugin struct{}

const baseURL = "https://www.zaobao.com"

var sectionMap = map[string]string{
	"china":     "/realtime/china",
	"singapore": "/realtime/singapore",
	"world":     "/realtime/world",
}

var sectionLabelMap = map[string]string{
	"china":     "中国",
	"singapore": "新加坡",
	"world":     "国际",
}

func (p *ZaobaoPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/zaobao/realtime/:section":
		section := req.Params["section"]
		if section == "" {
			section = "china"
		}
		return fetchRealtimeList(section)
	case req.Route == "/zaobao/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchRealtimeList(section string) (*sdk.FeedResult, error) {
	sectionPath, ok := sectionMap[section]
	if !ok {
		return nil, fmt.Errorf("unknown section: %s", section)
	}

	label := sectionLabelMap[section]
	if label == "" {
		label = section
	}

	items, err := fetchRealtimeNews(sectionPath)
	if err != nil {
		return nil, err
	}

	return &sdk.FeedResult{
		Title:       fmt.Sprintf("联合早报 - %s - 即时", label),
		Description: "新加坡、中国、亚洲和国际的即时、评论、商业、体育、生活、科技与多媒体新闻",
		Items:       items,
	}, nil
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

func fetchRealtimeNews(sectionPath string) ([]sdk.FeedItem, error) {
	url := baseURL + sectionPath

	// Fetch list page
	body, status, err := host.HTTPGet(url, map[string]string{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
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

	seen := make(map[string]struct{})
	var items []sdk.FeedItem

	appendItem := func(title, href string) {
		href = strings.TrimSpace(href)
		title = strings.TrimSpace(title)
		if href == "" || title == "" {
			return
		}

		articleURL := normalizeArticleURL(href)
		if _, exists := seen[articleURL]; exists {
			return
		}
		seen[articleURL] = struct{}{}

		items = append(items, sdk.FeedItem{
			ID:          articleURL,
			Title:       title,
			URL:         articleURL,
			PublishedAt: publishedAtFromURL(articleURL),
		})
	}

	doc.Find(".card-listing .card").Each(func(_ int, card *goquery.Selection) {
		link := card.Find(".content-header a").First()
		href, _ := link.Attr("href")
		appendItem(link.Text(), href)
	})

	if len(items) == 0 {
		doc.Find("[data-testid=\"article-list\"] article").Each(func(_ int, article *goquery.Selection) {
			link := article.Find("a.article-link").First()
			href, _ := link.Attr("href")
			appendItem(link.Text(), href)
		})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	return items, nil
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

func publishedAtFromURL(articleURL string) string {
	urlPart := strings.TrimPrefix(articleURL, baseURL)
	datePattern := regexp.MustCompile(`(\d{8})`)
	if matches := datePattern.FindStringSubmatch(urlPart); len(matches) > 1 {
		if t, err := time.Parse("20060102", matches[1]); err == nil {
			return t.Format(time.RFC3339)
		}
	}
	return time.Now().Format(time.RFC3339)
}

func fetchArticleDetails(articleURL string) (*sdk.FeedItem, error) {
	// Fetch article page
	body, status, err := host.HTTPGet(articleURL, map[string]string{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
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

	// Extract title
	title := doc.Find("h1").First().Text()
	if title == "" {
		title = doc.Find("meta[property=\"og:title\"]").AttrOr("content", "")
	}
	if title == "" {
		return nil, fmt.Errorf("no title found")
	}

	// Extract published date - try multiple sources
	pubDateStr := doc.Find("meta[property=\"article:published_time\"]").AttrOr("content", "")
	if pubDateStr == "" {
		pubDateStr = doc.Find("meta[name=\"publish_date\"]").AttrOr("content", "")
	}
	if pubDateStr == "" {
		pubDateStr = doc.Find("time").AttrOr("datetime", "")
	}
	if pubDateStr == "" {
		pubDateStr = doc.Find("meta[property=\"og:publish_time\"]").AttrOr("content", "")
	}

	var publishedAt string
	if pubDateStr != "" {
		// Try parsing different formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02",
			"2006-01-02 15:04:05",
			"20060102",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, pubDateStr); err == nil {
				publishedAt = t.Format(time.RFC3339)
				break
			}
		}
	}

	// Fallback: try extracting date from URL (e.g., story20260608-9172851 or 20260609)
	if publishedAt == "" {
		publishedAt = publishedAtFromURL(articleURL)
	}

	// Extract cover image
	coverImage := doc.Find("meta[property=\"og:image\"]").AttrOr("content", "")

	// Extract summary
	summary := doc.Find("meta[property=\"og:description\"]").AttrOr("content", "")

	// Extract article body HTML
	var contentHTML string
	articleBody := doc.Find(".articleBody, .article-body, [data-testid=\"article-body\"]").First()
	if articleBody.Length() > 0 {
		contentHTML, _ = articleBody.Html()
	} else {
		// Fallback: get main content
		contentHTML, _ = doc.Find("main, article, .content").First().Html()
	}

	// Clean up HTML (remove ads, buttons, etc.)
	cleanHTML := cleanArticleHTML(contentHTML)

	// Build description with image and content
	description := buildDescription(coverImage, summary, cleanHTML)

	return &sdk.FeedItem{
		ID:          articleURL,
		Title:       strings.TrimSpace(title),
		URL:         articleURL,
		Summary:     summary,
		Cover:       coverImage,
		Image:       coverImage,
		Content:     description,
		PublishedAt: publishedAt,
	}, nil
}

func cleanArticleHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	// Remove unwanted elements
	doc.Find(".bff-google-ad, .bff-recommend-article, button.cursor-pointer, .bff-inline-image-expand-icon, img[alt=\"expend icon\"], astro-island, .further-reading, .read-on-app-cover").Remove()

	result, _ := doc.Html()
	return result
}

func buildDescription(cover, summary, content string) string {
	var sb strings.Builder

	if cover != "" {
		sb.WriteString(fmt.Sprintf("<img src=\"%s\" style=\"max-width: 100%%; margin-bottom: 1rem;\"/>\n", cover))
	}

	if summary != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>%s</strong></p>\n", summary))
	}

	if content != "" {
		sb.WriteString(fmt.Sprintf("<div>%s</div>", content))
	}

	return sb.String()
}
