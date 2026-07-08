package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL     = "https://www.cnblogs.com"
	defaultSize = 20
)

func main() { sdk.Run(&CnblogsPlugin{}) }

type CnblogsPlugin struct{}

var sectionMap = map[string]struct{ Label, First, Paged string }{
	"home":     {"首页", "/", "/sitehome/p/%d"},
	"all":      {"所有随笔", "/cate/all", "/cate/all/%d"},
	"articles": {"所有文章", "/cate/articles", "/cate/articles/%d"},
	"backend":  {"后端开发", "/cate/2/", "/cate/2/%d"},
	"java":     {"Java", "/cate/java/", "/cate/java/%d"},
	"dotnet":   {".NET", "/cate/dotnet/", "/cate/dotnet/%d"},
	"frontend": {"前端开发", "/cate/108703/", "/cate/108703/%d"},
	"ai":       {"人工智能", "/cate/ai/", "/cate/ai/%d"},
}

func (p *CnblogsPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch req.Route {
	case "/cnblogs/list":
		section := strings.TrimSpace(req.Params["section"])
		if section == "" {
			section = "home"
		}
		return fetchList(section, req.Params)
	case "/cnblogs/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(section string, params map[string]string) (*sdk.FeedResult, error) {
	cfg, ok := sectionMap[section]
	if !ok {
		return nil, fmt.Errorf("unknown section: %s", section)
	}
	page := parsePositiveInt(params["page"], 1)
	body, status, err := host.HTTPGet(listURL(cfg, page), defaultHeaders("text/html,application/xhtml+xml"))
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	items := parseListItems(doc)
	if len(items) == 0 {
		return nil, fmt.Errorf("no items found")
	}
	result := &sdk.FeedResult{Title: "博客园 · " + cfg.Label, Description: "博客园技术文章聚合", Items: items}
	if hasNextPage(doc, page) || len(items) >= defaultSize {
		result.HasMore = true
		result.Next = map[string]string{"page": strconv.Itoa(page + 1)}
	}
	return result, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	articleURL := detailURLFromID(id)
	body, status, err := host.HTTPGet(articleURL, defaultHeaders("text/html,application/xhtml+xml"))
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
	return &sdk.FeedResult{Title: item.Title, Description: item.Summary, Items: []sdk.FeedItem{item}}, nil
}

func parseListItems(doc *goquery.Document) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, defaultSize)
	doc.Find("article.post-item").Each(func(_ int, sel *goquery.Selection) {
		id, _ := sel.Attr("data-post-id")
		id = strings.TrimSpace(id)
		titleNode := sel.Find("a.post-item-title").First()
		title := cleanText(titleNode.Text())
		link, _ := titleNode.Attr("href")
		link = normalizeURL(link)
		if id == "" {
			id = postIDFromURL(link)
		}
		if id == "" || title == "" || link == "" {
			return
		}
		author := cleanText(sel.Find("a.post-item-author").First().Text())
		avatar, _ := sel.Find("img.avatar").First().Attr("src")
		item := sdk.FeedItem{ID: link, Title: title, URL: link, Summary: cleanText(sel.Find("p.post-item-summary").First().Text()), Author: author, AuthorAvatar: normalizeURL(avatar), PublishedAt: parseListTime(sel.Find("footer .post-meta-item").First().Text())}
		item.Tags = append(item.Tags, "ID "+id)
		item.Tags = append(item.Tags, parseMetricTags(sel)...)
		items = append(items, item)
	})
	return items
}

func parseDetailItem(doc *goquery.Document, id, articleURL string) sdk.FeedItem {
	metadata := parseJSONLD(doc)
	title := cleanText(doc.Find("#cb_post_title_url").First().Text())
	if title == "" {
		title = cleanText(doc.Find(".postTitle2").First().Text())
	}
	if title == "" {
		title = metadata.Title
	}
	contentHTML, _ := doc.Find("#cnblogs_post_body").First().Html()
	author := cleanText(doc.Find("#Header1_HeaderTitle").First().Text())
	if author == "" {
		author = metadata.Author
	}
	return sdk.FeedItem{ID: id, Title: title, URL: articleURL, Content: strings.TrimSpace(contentHTML), Summary: metadata.Summary, Author: author, PublishedAt: parseDetailTime(doc.Find("#post-date").First(), metadata.PublishedAt), Tags: metadata.Keywords}
}

type articleMetadata struct {
	Title, Author, PublishedAt, Summary string
	Keywords                            []string
}

func parseJSONLD(doc *goquery.Document) articleMetadata {
	var meta articleMetadata
	doc.Find(`script[type="application/ld+json"], script[type="application/ld&#x2B;json"]`).EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		var raw struct {
			Headline, Name, Description, DatePublished string
			Keywords                                   interface{}
			Author                                     struct{ Name string }
		}
		if err := json.Unmarshal([]byte(sel.Text()), &raw); err != nil {
			return true
		}
		meta.Title = strings.TrimSpace(firstNonEmpty(raw.Headline, raw.Name))
		meta.Author = strings.TrimSpace(raw.Author.Name)
		meta.PublishedAt = strings.TrimSpace(raw.DatePublished)
		meta.Summary = strings.TrimSpace(raw.Description)
		meta.Keywords = normalizeKeywords(raw.Keywords)
		return meta.Title == ""
	})
	return meta
}

func parseMetricTags(sel *goquery.Selection) []string {
	var tags []string
	sel.Find("a.post-meta-item").Each(func(_ int, item *goquery.Selection) {
		if title := cleanText(attr(item, "title")); title != "" {
			tags = append(tags, title)
		}
	})
	return tags
}

func listURL(cfg struct{ Label, First, Paged string }, page int) string {
	if page <= 1 {
		return baseURL + cfg.First
	}
	return baseURL + fmt.Sprintf(cfg.Paged, page)
}
func detailURLFromID(id string) string {
	if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") {
		return id
	}
	if decoded, err := url.QueryUnescape(id); err == nil && strings.HasPrefix(decoded, "https://") {
		return decoded
	}
	return baseURL + "/p/" + strings.TrimSpace(id)
}
func postIDFromURL(raw string) string {
	parts := strings.Split(strings.TrimRight(raw, "/"), "/p/")
	if len(parts) != 2 {
		return ""
	}
	return strings.Trim(strings.ReplaceAll(parts[1], "/", "-"), "-")
}
func hasNextPage(doc *goquery.Document, page int) bool {
	return doc.Find(fmt.Sprintf(`#paging_block a[href][onclick*="loadCategoryPostList(%d,"]`, page+1)).Length() > 0
}
func parseListTime(raw string) string {
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02"} {
		if t, ok := parseTimeInLocation(raw, layout); ok {
			return t
		}
	}
	return time.Now().Format(time.RFC3339)
}
func parseDetailTime(sel *goquery.Selection, fallback string) string {
	if v, ok := sel.Attr("data-date-updated"); ok {
		if t, ok := parseTimeInLocation(v, "2006-01-02 15:04"); ok {
			return t
		}
	}
	if t, ok := parseTimeInLocation(sel.Text(), "2006-01-02 15:04"); ok {
		return t
	}
	if fallback != "" {
		if t, err := time.Parse(time.RFC3339Nano, strings.ReplaceAll(fallback, "&#x2B;", "+")); err == nil {
			return t.Format(time.RFC3339)
		}
	}
	return time.Now().Format(time.RFC3339)
}
func parseTimeInLocation(raw, layout string) (string, bool) {
	loc := time.FixedZone("CST", 8*60*60)
	value := regexp.MustCompile(`\d{4}-\d{2}-\d{2}(?:\s+\d{2}:\d{2})?`).FindString(cleanText(raw))
	if value == "" || (strings.Contains(layout, "15:04") && !strings.Contains(value, " ")) {
		return "", false
	}
	if !strings.Contains(layout, "15:04") && strings.Contains(value, " ") {
		value = strings.Fields(value)[0]
	}
	t, err := time.ParseInLocation(layout, value, loc)
	if err != nil {
		return "", false
	}
	return t.Format(time.RFC3339), true
}
func normalizeKeywords(raw interface{}) []string {
	var tags []string
	switch value := raw.(type) {
	case string:
		for _, part := range strings.Split(value, ",") {
			if tag := strings.Trim(strings.TrimSpace(part), "#"); tag != "" {
				tags = append(tags, tag)
			}
		}
	case []interface{}:
		for _, item := range value {
			if tag := strings.TrimSpace(fmt.Sprint(item)); tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return tags
}
func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if strings.HasPrefix(raw, "/") {
		return baseURL + raw
	}
	return raw
}
func cleanText(raw string) string { return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ") }
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
func parsePositiveInt(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
func defaultHeaders(accept string) map[string]string {
	return map[string]string{"Accept": accept, "Referer": baseURL + "/", "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"}
}
func attr(sel *goquery.Selection, name string) string { value, _ := sel.Attr(name); return value }
