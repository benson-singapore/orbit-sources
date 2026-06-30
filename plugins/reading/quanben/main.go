package main

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	siteName  = "全本小说网"
	baseURL   = "https://www.quanben.io"
	userAgent = "Mozilla/5.0 (compatible; OrbitPlugins/1.0)"
	introID   = "intro"
)

var categoryLabels = map[string]string{
	"xuanhuan":  "玄幻",
	"dushi":     "都市",
	"yanqing":   "言情",
	"chuanyue":  "穿越",
	"qingchun":  "青春",
	"xianxia":   "仙侠",
	"lingyi":    "灵异",
	"xuanyi":    "悬疑",
	"lishi":     "历史",
	"junshi":    "军事",
	"youxi":     "游戏",
	"jingji":    "竞技",
	"kehuan":    "科幻",
	"zhichang":  "职场",
	"guanchang": "官场",
	"xianyan":   "现言",
	"danmei":    "耽美",
	"qita":      "其它",
}

var (
	reBookCard     = regexp.MustCompile(`(?is)<div\s+class=['"]list2['"][^>]*>(.*?)</div>`)
	reBookHref     = regexp.MustCompile(`(?is)<h3[^>]*>\s*<a\s+href=['"]/n/([^/'"]+)/?['"]`)
	reBookTitle    = regexp.MustCompile(`(?is)itemprop=['"]name['"][^>]*>([^<]+)<`)
	reBookAuthor   = regexp.MustCompile(`(?is)作者:\s*<span[^>]*>([^<]+)</span>`)
	reBookCover    = regexp.MustCompile(`(?is)<img[^>]+src=['"]([^'"]+)['"]`)
	reBookDesc       = regexp.MustCompile(`(?is)itemprop=['"]description['"][^>]*>\s*<p>(.*?)</p>`)
	reBookDescInline = regexp.MustCompile(`(?is)itemprop=['"]description['"][^>]*>([^<]+)<`)
	reBookCategory   = regexp.MustCompile(`(?is)类别:\s*<span[^>]*>([^<]+)</span>`)
	reBookStatus     = regexp.MustCompile(`(?is)状态:\s*<span[^>]*>([^<]+)</span>`)
	reDetailH1       = regexp.MustCompile(`(?is)<h1[^>]*itemprop=['"]name[^'"]*['"][^>]*>([^<]+)</h1>`)
	reChapterLink    = regexp.MustCompile(`(?is)<li[^>]*>\s*<a\s+href=['"]/?(?:amp/)?n/[^/]+/(\d+)\.html['"][^>]*>([^<]+)</a>`)
	reHeadline     = regexp.MustCompile(`(?is)<h1\s+class=['"]headline['"][^>]*>([^<]+)</h1>`)
	reChapterBody  = regexp.MustCompile(`(?is)<div\s+id=['"]content['"][^>]*>(.*?)</div>`)
	reNextPage     = regexp.MustCompile(`(?is)<a\s+href=['"]([^'"]+)['"][^>]*rel=['"]next['"]`)
	rePageTitle    = regexp.MustCompile(`(?is)<title>([^<]+)</title>`)
	reStripTags    = regexp.MustCompile(`(?s)<[^>]+>`)
	reRemoveAd     = regexp.MustCompile(`(?is)<span\s+id=['"]ad['"][^>]*>.*?</span>`)
	rePageMarkers  = regexp.MustCompile(`<!--PAGE\s+\d+-->`)
)

func main() {
	sdk.Run(&QuanbenPlugin{})
}

type QuanbenPlugin struct{}

func (p *QuanbenPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/quanben/list" || strings.HasPrefix(req.Route, "/quanben/list"):
		return fetchList(req.Params)
	case req.Route == "/quanben/search" || strings.HasPrefix(req.Route, "/quanben/search"):
		query := strings.TrimSpace(req.Params["query"])
		if query == "" {
			return nil, fmt.Errorf("missing query parameter")
		}
		return fetchSearch(query)
	case req.Route == "/quanben/chapters/:id" || strings.HasPrefix(req.Route, "/quanben/chapters"):
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchChapters(id)
	case req.Route == "/quanben/chapter/:chapterId" || strings.HasPrefix(req.Route, "/quanben/chapter"):
		parentID := strings.TrimSpace(req.Params["id"])
		chapterID := strings.TrimSpace(req.Params["chapterId"])
		if parentID == "" || chapterID == "" {
			return nil, fmt.Errorf("missing id or chapterId parameter")
		}
		return fetchChapter(parentID, chapterID)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(params map[string]string) (*sdk.FeedResult, error) {
	slug := strings.TrimSpace(params["slug"])
	if slug == "" {
		return nil, fmt.Errorf("missing slug parameter")
	}
	label, ok := categoryLabels[slug]
	if !ok {
		return nil, fmt.Errorf("unknown category: %s", slug)
	}

	page := pageNum(params["page"])
	listURL := categoryURL(slug, page)

	body, status, err := httpGet(listURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	htmlBody := string(body)
	items := parseBookList(htmlBody)
	if len(items) == 0 {
		return nil, fmt.Errorf("no books found")
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("%s · %s", siteName, label),
		Description: fmt.Sprintf("%s小说列表 · 第 %d 页", label, page),
		Items:       items,
	}
	if next := reNextPage.FindStringSubmatch(htmlBody); len(next) > 1 {
		result.HasMore = true
		result.Next = copyParams(params)
		result.Next["page"] = strconv.Itoa(page + 1)
	}
	return result, nil
}

func fetchSearch(query string) (*sdk.FeedResult, error) {
	searchURL := baseURL + "/index.php?" + url.Values{
		"c":        {"book"},
		"a":        {"search"},
		"keywords": {query},
	}.Encode()

	body, status, err := httpGet(searchURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	items := parseBookList(string(body))
	if len(items) == 0 {
		return nil, fmt.Errorf("no results for: %s", query)
	}

	return &sdk.FeedResult{
		Title:       fmt.Sprintf("%s · 搜索：%s", siteName, query),
		Description: fmt.Sprintf("搜索「%s」共 %d 部小说", query, len(items)),
		Items:       items,
	}, nil
}

func fetchChapters(bookSlug string) (*sdk.FeedResult, error) {
	pageURL := bookURL(bookSlug)
	listURL := fmt.Sprintf("%s/amp/n/%s/list.html", baseURL, bookSlug)
	body, status, err := httpGet(listURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	htmlBody := string(body)
	chapters := parseChapterList(htmlBody, bookSlug)
	if len(chapters) == 0 {
		return nil, fmt.Errorf("no chapters found")
	}

	title := extractBookTitle(htmlBody)
	if title == "" {
		title = bookSlug
	}

	items := make([]sdk.FeedItem, 0, len(chapters)+1)
	items = append(items, sdk.FeedItem{
		ID:    introID,
		Title: "简介 / 详情",
		URL:   pageURL,
		Tags:  []string{"介绍"},
	})
	items = append(items, chapters...)

	desc := ""
	if detailBody, detailStatus, detailErr := httpGet(pageURL); detailErr == nil && detailStatus >= 200 && detailStatus < 300 {
		desc = strings.TrimSpace(parseBookDetail(string(detailBody)).description)
	}

	descParts := []string{fmt.Sprintf("共 %d 章", len(chapters))}
	if desc != "" {
		descParts = append(descParts, truncate(desc, 80))
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: strings.Join(descParts, " · "),
		Items:       items,
	}, nil
}

func fetchChapter(bookSlug, chapterID string) (*sdk.FeedResult, error) {
	if chapterID == introID {
		return fetchIntroChapter(bookSlug)
	}

	chapterURL := fmt.Sprintf("%s/n/%s/%s.html", baseURL, bookSlug, chapterID)
	body, status, err := httpGet(chapterURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	htmlBody := string(body)
	title, content := extractChapter(htmlBody)
	if title == "" {
		title = "正文"
	}
	if content == "" {
		return nil, fmt.Errorf("chapter content not found")
	}

	item := sdk.FeedItem{
		ID:      chapterID,
		Title:   title,
		URL:     chapterURL,
		Content: content,
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "章节正文",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func parseBookList(htmlBody string) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, 16)
	seen := make(map[string]bool)

	for _, card := range reBookCard.FindAllStringSubmatch(htmlBody, -1) {
		if len(card) < 2 {
			continue
		}
		block := card[1]

		hrefMatch := reBookHref.FindStringSubmatch(block)
		if len(hrefMatch) < 2 {
			continue
		}
		slug := strings.TrimSpace(hrefMatch[1])
		if slug == "" || seen[slug] {
			continue
		}

		title := ""
		if m := reBookTitle.FindStringSubmatch(block); len(m) > 1 {
			title = cleanText(m[1])
		}
		if title == "" {
			continue
		}

		author := ""
		if m := reBookAuthor.FindStringSubmatch(block); len(m) > 1 {
			author = cleanText(m[1])
		}

		cover := ""
		if m := reBookCover.FindStringSubmatch(block); len(m) > 1 {
			cover = normalizeImageURL(m[1])
		}

		summary := ""
		if m := reBookDescInline.FindStringSubmatch(block); len(m) > 1 {
			summary = cleanText(m[1])
		}

		seen[slug] = true
		bookURL := fmt.Sprintf("%s/n/%s/", baseURL, slug)
		items = append(items, sdk.FeedItem{
			ID:          slug,
			Title:       title,
			URL:         bookURL,
			Summary:     summary,
			Author:      author,
			Cover:       cover,
			Image:       cover,
			PublishedAt: time.Now().Format(time.RFC3339),
		})
	}

	return items
}

func parseChapterList(htmlBody, bookSlug string) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, 256)
	seen := make(map[string]bool)

	for _, match := range reChapterLink.FindAllStringSubmatch(htmlBody, -1) {
		if len(match) < 3 {
			continue
		}
		chapterID := strings.TrimSpace(match[1])
		title := cleanText(match[2])
		if chapterID == "" || title == "" || seen[chapterID] {
			continue
		}
		seen[chapterID] = true

		chapterURL := fmt.Sprintf("%s/n/%s/%s.html", baseURL, bookSlug, chapterID)
		items = append(items, sdk.FeedItem{
			ID:          chapterID,
			Title:       title,
			URL:         chapterURL,
			PublishedAt: time.Now().Format(time.RFC3339),
		})
	}

	return items
}

func extractBookTitle(htmlBody string) string {
	return firstNonEmpty(
		firstMatch(reBookTitle, htmlBody),
		cleanPageTitle(htmlBody),
	)
}

func cleanPageTitle(htmlBody string) string {
	title := firstMatch(rePageTitle, htmlBody)
	title = strings.TrimSuffix(title, " - 全本小说网")
	if idx := strings.Index(title, "》"); idx > 0 && strings.Contains(title, "《") {
		title = strings.TrimPrefix(title[strings.Index(title, "《"):], "《")
		title = strings.TrimSuffix(title, "》")
	}
	return title
}

func firstMatch(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return cleanText(m[1])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func htmlEscape(s string) string {
	return html.EscapeString(s)
}

func extractChapter(htmlBody string) (string, string) {
	title := ""
	if m := reHeadline.FindStringSubmatch(htmlBody); len(m) > 1 {
		title = cleanText(m[1])
	}

	content := ""
	if m := reChapterBody.FindStringSubmatch(htmlBody); len(m) > 1 {
		bodyHTML := strings.TrimSpace(m[1])
		bodyHTML = reRemoveAd.ReplaceAllString(bodyHTML, "")
		bodyHTML = rePageMarkers.ReplaceAllString(bodyHTML, "")
		content = fmt.Sprintf(`<div class="chapter-content">%s</div>`, bodyHTML)
	}

	return title, content
}

func fetchIntroChapter(bookSlug string) (*sdk.FeedResult, error) {
	pageURL := bookURL(bookSlug)
	body, status, err := httpGet(pageURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	detail := parseBookDetail(string(body))
	title := firstNonEmpty(detail.title, bookSlug)
	content := buildBookIntroHTML(detail, pageURL)

	item := sdk.FeedItem{
		ID:      introID,
		Title:   title + " · 简介",
		URL:     pageURL,
		Content: content,
		Cover:   detail.cover,
		Image:   detail.cover,
		Author:  detail.author,
		Summary: detail.description,
	}
	if detail.category != "" {
		item.Tags = []string{detail.category}
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: siteName,
		Items:       []sdk.FeedItem{item},
	}, nil
}

type bookDetail struct {
	title       string
	author      string
	category    string
	status      string
	cover       string
	description string
}

func parseBookDetail(htmlBody string) bookDetail {
	detail := bookDetail{
		title: firstNonEmpty(
			firstMatch(reDetailH1, htmlBody),
			firstMatch(reBookTitle, htmlBody),
			cleanPageTitle(htmlBody),
		),
		author:   firstMatch(reBookAuthor, htmlBody),
		category: firstMatch(reBookCategory, htmlBody),
		status:   firstMatch(reBookStatus, htmlBody),
		cover:    normalizeImageURL(firstMatch(reBookCover, htmlBody)),
	}

	if m := reBookDesc.FindStringSubmatch(htmlBody); len(m) > 1 {
		detail.description = cleanText(m[1])
	} else if m := reBookDescInline.FindStringSubmatch(htmlBody); len(m) > 1 {
		detail.description = cleanText(m[1])
	}

	return detail
}

func buildBookIntroHTML(detail bookDetail, sourceURL string) string {
	var sb strings.Builder
	sb.WriteString(`<article class="book-detail" style="margin:0;padding:0;line-height:1.7;color:#1f2937;">`)

	sb.WriteString(`<header style="padding:12px 16px;border-bottom:1px solid #e5e7eb;background:#fafafa;">`)
	sb.WriteString(fmt.Sprintf(`<h1 style="margin:0;font-size:18px;font-weight:700;">%s</h1>`, htmlEscape(detail.title)))
	if detail.author != "" {
		sb.WriteString(fmt.Sprintf(`<p style="margin:6px 0 0;font-size:13px;color:#6b7280;">作者：%s</p>`, htmlEscape(detail.author)))
	}
	sb.WriteString(`</header>`)

	sb.WriteString(`<section style="padding:16px;">`)
	if detail.cover != "" {
		sb.WriteString(`<div style="display:flex;gap:16px;align-items:flex-start;">`)
		sb.WriteString(fmt.Sprintf(
			`<img src="%s" alt="%s 封面" loading="lazy" style="width:96px;min-width:96px;height:128px;object-fit:cover;border:1px solid #e5e7eb;border-radius:4px;"/>`,
			htmlEscape(detail.cover),
			htmlEscape(detail.title),
		))
		sb.WriteString(`<div style="flex:1;min-width:0;font-size:13px;color:#6b7280;">`)
		if detail.category != "" {
			sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 8px;">类别：%s</p>`, htmlEscape(detail.category)))
		}
		if detail.status != "" {
			sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 8px;">状态：%s</p>`, htmlEscape(detail.status)))
		}
		sb.WriteString(fmt.Sprintf(`<p style="margin:0;">来源：<a href="%s" style="color:#2563eb;">打开原网页</a></p>`, htmlEscape(sourceURL)))
		sb.WriteString(`</div></div>`)
	} else {
		meta := make([]string, 0, 2)
		if detail.category != "" {
			meta = append(meta, "类别："+detail.category)
		}
		if detail.status != "" {
			meta = append(meta, "状态："+detail.status)
		}
		if len(meta) > 0 {
			sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 12px;font-size:13px;color:#6b7280;">%s</p>`, htmlEscape(strings.Join(meta, " · "))))
		}
		sb.WriteString(fmt.Sprintf(`<p style="margin:0;font-size:13px;color:#6b7280;">来源：<a href="%s" style="color:#2563eb;">打开原网页</a></p>`, htmlEscape(sourceURL)))
	}

	if detail.description != "" {
		sb.WriteString(`<h2 style="margin:20px 0 8px;font-size:15px;font-weight:600;">简介</h2>`)
		sb.WriteString(fmt.Sprintf(`<p style="margin:0;white-space:pre-wrap;color:#374151;">%s</p>`, htmlEscape(detail.description)))
	} else {
		sb.WriteString(`<p style="margin:20px 0 0;font-size:13px;color:#9ca3af;">暂无简介</p>`)
	}

	sb.WriteString(`</section></article>`)
	return sb.String()
}

func bookURL(slug string) string {
	return fmt.Sprintf("%s/n/%s/", baseURL, slug)
}

func categoryURL(slug string, page int) string {
	if page <= 1 {
		return fmt.Sprintf("%s/c/%s.html", baseURL, slug)
	}
	return fmt.Sprintf("%s/c/%s_%d.html", baseURL, slug, page)
}

func pageNum(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func copyParams(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func normalizeImageURL(raw string) string {
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

func cleanText(s string) string {
	s = reStripTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.TrimSpace(s)
	return strings.Join(strings.Fields(s), " ")
}

func httpGet(rawURL string) ([]byte, int, error) {
	body, status, err := host.HTTPGet(rawURL, map[string]string{
		"User-Agent": userAgent,
		"Accept":     "text/html,application/xhtml+xml",
	})
	if err != nil {
		return nil, 0, fmt.Errorf("http get failed: %w", err)
	}
	return body, status, nil
}
