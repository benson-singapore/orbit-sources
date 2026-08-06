package main

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

const (
	siteOrigin = "https://notfound9.github.io"
	baseURL    = "https://notfound9.github.io/interviewGuide"
	author     = "大厂面试指北"
)

func main() { sdk.Run(&Plugin{}) }

type Plugin struct{}

func (p *Plugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch req.Route {
	case "/interviewguide/list":
		section := strings.TrimSpace(req.Params["section"])
		if section == "" {
			section = "java"
		}
		return fetchList(section)
	case "/interviewguide/detail/:id":
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
		pageURL := docsifyURL(entry.Path)
		items = append(items, sdk.FeedItem{
			ID:          pageURL,
			Title:       entry.Title,
			URL:         pageURL,
			Summary:     sec.Label + " · " + entry.Title,
			Author:      author,
			PublishedAt: time.Now().Format(time.RFC3339),
			Tags:        []string{sec.Label},
		})
	}
	return &sdk.FeedResult{
		Title:       "大厂面试指北 · " + sec.Label,
		Description: "Java 后端面试题精选：Java / Redis / MySQL / JVM / 系统设计 / 算法等",
		Items:       items,
	}, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	mdPath := markdownPathFromID(id)
	if mdPath == "" {
		return nil, fmt.Errorf("invalid article id: %s", id)
	}
	mdURL := markdownURL(mdPath)
	pageURL := docsifyURL(mdPath)

	body, status, err := host.HTTPGet(mdURL, defaultHeaders())
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d for %s", status, mdURL)
	}

	rawMD := cleanMarkdown(string(body))
	contentHTML, err := renderMarkdownHTML(rawMD)
	if err != nil {
		return nil, fmt.Errorf("render markdown: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<div class=\"markdown-body\">" + contentHTML + "</div>"))
	if err != nil {
		return nil, fmt.Errorf("parse rendered html: %w", err)
	}
	root := doc.Find(".markdown-body").First()
	cleanArticleHTML(root)

	title := titleFromCatalog(mdPath)
	if title == "" {
		title = firstHeading(root)
	}
	if title == "" {
		title = path.Base(mdPath)
	}

	summary := firstParagraph(root)
	content, _ := root.Html()
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("article content not found")
	}

	cover := firstContentImage(root)

	item := sdk.FeedItem{
		ID:          pageURL,
		Title:       title,
		URL:         pageURL,
		Content:     content,
		Summary:     summary,
		Author:      author,
		Cover:       cover,
		Image:       cover,
		PublishedAt: time.Now().Format(time.RFC3339),
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func renderMarkdownHTML(md string) (string, error) {
	converter := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithUnsafe()),
	)
	var buf bytes.Buffer
	if err := converter.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func titleFromCatalog(mdPath string) string {
	mdPath = normalizeMDPath(mdPath)
	for _, sec := range catalogSections {
		for _, item := range sec.Items {
			if normalizeMDPath(item.Path) == mdPath {
				return item.Title
			}
		}
	}
	return ""
}

func firstHeading(root *goquery.Selection) string {
	for _, sel := range []string{"h1", "h2", "h3"} {
		text := cleanText(root.Find(sel).First().Text())
		if text != "" {
			return text
		}
	}
	return ""
}

func firstParagraph(root *goquery.Selection) string {
	var summary string
	root.Find("p").EachWithBreak(func(_ int, p *goquery.Selection) bool {
		text := cleanText(p.Text())
		if text == "" || isPromoLine(text) {
			return true
		}
		summary = text
		return false
	})
	if len([]rune(summary)) > 180 {
		summary = string([]rune(summary)[:180]) + "…"
	}
	return summary
}

func firstContentImage(sel *goquery.Selection) string {
	var cover string
	sel.Find("img[src]").EachWithBreak(func(_ int, img *goquery.Selection) bool {
		src := absoluteAssetURL(img.AttrOr("src", ""))
		if src == "" || isPromoImageURL(src) {
			return true
		}
		cover = src
		return false
	})
	return cover
}

func markdownPathFromID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if decoded, err := url.QueryUnescape(id); err == nil {
		id = decoded
	}

	// Docsify hash URL: https://.../interviewGuide/#/docs/JavaBasic
	if i := strings.Index(id, "#/"); i >= 0 {
		hashPath := strings.TrimPrefix(id[i+2:], "/")
		hashPath = strings.SplitN(hashPath, "?", 2)[0]
		hashPath = strings.SplitN(hashPath, "#", 2)[0]
		return ensureMarkdownExt(hashPath)
	}

	// Raw markdown / absolute site path
	if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") {
		u, err := url.Parse(id)
		if err != nil {
			return ""
		}
		p := strings.TrimPrefix(u.Path, "/interviewGuide/")
		p = strings.TrimPrefix(p, "/")
		return ensureMarkdownExt(p)
	}

	id = strings.TrimPrefix(id, "/")
	id = strings.TrimPrefix(id, "interviewGuide/")
	return ensureMarkdownExt(id)
}

func ensureMarkdownExt(p string) string {
	p = normalizeMDPath(p)
	if p == "" || p == "/" {
		return "README.md"
	}
	if strings.HasSuffix(strings.ToLower(p), ".md") {
		return p
	}
	return p + ".md"
}

func normalizeMDPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	return path.Clean(p)
}

func markdownURL(mdPath string) string {
	mdPath = normalizeMDPath(mdPath)
	return baseURL + "/" + mdPath
}

func docsifyURL(mdPath string) string {
	mdPath = normalizeMDPath(mdPath)
	route := strings.TrimSuffix(mdPath, ".md")
	route = strings.TrimSuffix(route, ".MD")
	if route == "README" || route == "" {
		return baseURL + "/#/"
	}
	return baseURL + "/#/" + route
}

func cleanText(raw string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
}

func defaultHeaders() map[string]string {
	return map[string]string{
		"Accept":     "text/markdown,text/plain,*/*",
		"Referer":    baseURL + "/",
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	}
}
