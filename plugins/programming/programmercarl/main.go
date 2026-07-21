package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL = "https://programmercarl.com"
	author  = "代码随想录"
)

func main() { sdk.Run(&Plugin{}) }

type Plugin struct{}

func (p *Plugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch req.Route {
	case "/programmercarl/list":
		section := strings.TrimSpace(req.Params["section"])
		if section == "" {
			section = "perf"
		}
		return fetchList(section)
	case "/programmercarl/detail/:id":
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
	sec, ok := catalogByID[section]
	if !ok || sec == nil {
		return nil, fmt.Errorf("unknown section: %s", section)
	}
	items := make([]sdk.FeedItem, 0, len(sec.Items))
	for _, entry := range sec.Items {
		articleURL := absoluteURL(entry.Path)
		items = append(items, sdk.FeedItem{
			ID:          articleURL,
			Title:       entry.Title,
			URL:         articleURL,
			Summary:     sec.Label + " · " + entry.Title,
			Author:      author,
			PublishedAt: time.Now().Format(time.RFC3339),
			Tags:        []string{sec.Label},
		})
	}
	return &sdk.FeedResult{
		Title:       "代码随想录 · " + sec.Label,
		Description: "代码随想录算法与编程基础教程",
		Items:       items,
	}, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	articleURL := detailURLFromID(id)
	body, status, err := host.HTTPGet(articleURL, defaultHeaders())
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse detail html: %w", err)
	}
	item := parseDetailItem(doc, id, articleURL)
	if item.Title == "" {
		return nil, fmt.Errorf("article title not found")
	}
	if item.Content == "" {
		return nil, fmt.Errorf("article content not found")
	}
	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func parseDetailItem(doc *goquery.Document, id, articleURL string) sdk.FeedItem {
	contentSel := doc.Find(".theme-default-content.content__default").First()
	if contentSel.Length() == 0 {
		contentSel = doc.Find(".theme-default-content").First()
	}

	title := cleanText(contentSel.Find("h1").First().Text())
	title = strings.TrimLeft(title, "# ")
	if title == "" {
		title = cleanText(doc.Find(`meta[property="og:title"]`).AttrOr("content", ""))
	}
	if title == "" {
		title = cleanText(doc.Find("title").First().Text())
		if i := strings.Index(title, " | "); i > 0 {
			title = title[:i]
		}
	}

	summary := cleanText(doc.Find(`meta[name="description"]`).AttrOr("content", ""))
	if summary == "" {
		summary = cleanText(doc.Find(`meta[property="og:description"]`).AttrOr("content", ""))
	}

	publishedAt := strings.TrimSpace(doc.Find(`meta[property="article:modified_time"]`).AttrOr("content", ""))
	if publishedAt == "" {
		publishedAt = time.Now().Format(time.RFC3339)
	} else if t, err := time.Parse(time.RFC3339Nano, publishedAt); err == nil {
		publishedAt = t.Format(time.RFC3339)
	} else if t, err := time.Parse(time.RFC3339, publishedAt); err == nil {
		publishedAt = t.Format(time.RFC3339)
	}

	tags := parseMetaTags(doc.Find(`meta[name="tags"], meta[property="article:tag"]`))
	cleaned := cleanArticleContent(contentSel.Clone())
	contentHTML, _ := cleaned.Html()
	contentHTML = strings.TrimSpace(contentHTML)

	cover := firstContentImage(cleaned)

	return sdk.FeedItem{
		ID:          id,
		Title:       title,
		URL:         articleURL,
		Content:     contentHTML,
		Summary:     summary,
		Author:      author,
		Cover:       cover,
		Image:       cover,
		PublishedAt: publishedAt,
		Tags:        tags,
	}
}

func parseMetaTags(sel *goquery.Selection) []string {
	seen := map[string]struct{}{}
	var tags []string
	sel.Each(func(_ int, s *goquery.Selection) {
		raw := s.AttrOr("content", "")
		for _, part := range strings.Split(raw, ",") {
			tag := strings.TrimSpace(part)
			if tag == "" {
				continue
			}
			if _, ok := seen[tag]; ok {
				continue
			}
			seen[tag] = struct{}{}
			tags = append(tags, tag)
		}
	})
	return tags
}

func firstContentImage(sel *goquery.Selection) string {
	var cover string
	sel.Find("img[src]").EachWithBreak(func(_ int, img *goquery.Selection) bool {
		src := absoluteURL(img.AttrOr("src", ""))
		if src == "" || isPromoImageURL(src) {
			return true
		}
		cover = src
		return false
	})
	return cover
}

func detailURLFromID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") {
		return id
	}
	if decoded, err := url.QueryUnescape(id); err == nil {
		if strings.HasPrefix(decoded, "http://") || strings.HasPrefix(decoded, "https://") {
			return decoded
		}
		if strings.HasPrefix(decoded, "/") {
			return absoluteURL(decoded)
		}
	}
	if strings.HasPrefix(id, "/") {
		return absoluteURL(id)
	}
	return absoluteURL("/" + id)
}

func absoluteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "/") {
		return baseURL + raw
	}
	return baseURL + "/" + raw
}

func cleanText(raw string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
}

func defaultHeaders() map[string]string {
	return map[string]string{
		"Accept":     "text/html,application/xhtml+xml",
		"Referer":    baseURL + "/",
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	}
}
