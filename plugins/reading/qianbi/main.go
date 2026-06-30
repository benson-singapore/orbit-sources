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
	siteName  = "铅笔小说"
	baseURL   = "https://www.23qb.net"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	introChapterID = "0"
	introID        = "intro" // legacy id, still accepted on fetch
)

var categoryLabels = map[string]string{
	"all":      "全部",
	"yanqing":  "言情",
	"dushi":    "都市",
	"danmei":   "耽美",
	"chuanyue": "穿越",
	"qingchun": "青春",
	"xuanhuan": "玄幻",
	"wuxia":    "武侠",
	"lishi":    "军事",
	"youxi":    "竞技",
	"kehuan":   "科幻",
	"xuanyi":   "悬疑",
	"tongren":  "同人",
	"zhichang": "职场",
}

var categoryIDs = map[string]int{
	"all":      0,
	"yanqing":  1,
	"dushi":    2,
	"danmei":   3,
	"chuanyue": 4,
	"qingchun": 5,
	"xuanhuan": 6,
	"wuxia":    7,
	"lishi":    8,
	"youxi":    9,
	"kehuan":   10,
	"xuanyi":   11,
	"tongren":  12,
	"zhichang": 13,
}

var (
	reBookCard      = regexp.MustCompile(`(?is)<div\s+class=['"]module-item['"]>([\s\S]*?<div\s+class=['"]module-item-text['"]>[^<]*</div>)\s*</div>`)
	reBookHref      = regexp.MustCompile(`(?is)href=['"]/book/(\d+)/['"]`)
	reBookTitle     = regexp.MustCompile(`(?is)class=['"]module-item-title['"][^>]*title=['"]([^'"]*)['"][^>]*>([^<]+)<`)
	reBookAuthor    = regexp.MustCompile(`(?is)<div\s+class=['"]module-item-text['"]>([^<]+)</div>`)
	reBookCover     = regexp.MustCompile(`(?is)data-src=['"]([^'"]+)['"]`)
	reChapterLink   = regexp.MustCompile(`(?is)<a\s+class=['"]module-row-text['"]\s+href=['"]/book/\d+/(\d+)\.html['"][^>]*title=['"]([^'"]*)['"]`)
	reChapterBody   = regexp.MustCompile(`(?is)<div\s+class=['"]article-content['"]>(.*?)</div>`)
	reListNext      = regexp.MustCompile(`(?is)<a\s+href=['"]([^'#][^'"]*)['"]\s+class=['"]page-number\s+page-next['"]`)
	rePageTitle     = regexp.MustCompile(`(?is)<title>([^<]+)</title>`)
	reReadParams    = regexp.MustCompile(`(?is)chaptername:'([^']*)'`)
	reChapterVIP    = regexp.MustCompile(`(?is)chapterisvip:'([^']*)'`)
	reStripTags     = regexp.MustCompile(`(?s)<[^>]+>`)
	reOGTitle       = regexp.MustCompile(`(?is)<meta\s+property=['"]og:title['"]\s+content=['"]([^'"]*)['"]`)
	reOGAuthor      = regexp.MustCompile(`(?is)<meta\s+property=['"]og:novel:author['"]\s+content=['"]([^'"]*)['"]`)
	reOGDesc        = regexp.MustCompile(`(?is)<meta\s+property=['"]og:description['"]\s+content=['"]([^'"]*)['"]`)
	reOGImage       = regexp.MustCompile(`(?is)<meta\s+property=['"]og:image['"]\s+content=['"]([^'"]*)['"]`)
	reOGTags        = regexp.MustCompile(`(?is)<meta\s+property=['"]og:novel:tags['"]\s+content=['"]([^'"]*)['"]`)
	reOGStatus      = regexp.MustCompile(`(?is)<meta\s+property=['"]og:novel:status['"]\s+content=['"]([^'"]*)['"]`)
	reOGCategory    = regexp.MustCompile(`(?is)<meta\s+property=['"]og:novel:category['"]\s+content=['"]([^'"]*)['"]`)
	reCatalogTitle  = regexp.MustCompile(`(?is)<h1\s+class=['"]page-title['"]>\s*<a\s+href=['"]/book/\d+/?['"]>([^<]+)</a>`)
	reSearchTitle   = regexp.MustCompile(`(?is)<h3>\s*<a\s+href=['"]/book/(\d+)/['"][^>]*title=['"]([^'"]*)['"][^>]*>([^<]+)</a>`)
	reSearchSummary = regexp.MustCompile(`(?is)<div\s+class=['"]novel-info-item['"]>([\s\S]*?)</div>`)
)

func main() {
	sdk.Run(&QianbiPlugin{})
}

type QianbiPlugin struct{}

func (p *QianbiPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/qianbi/list" || strings.HasPrefix(req.Route, "/qianbi/list"):
		return fetchList(req.Params)
	case req.Route == "/qianbi/search" || strings.HasPrefix(req.Route, "/qianbi/search"):
		query := strings.TrimSpace(req.Params["query"])
		if query == "" {
			return nil, fmt.Errorf("missing query parameter")
		}
		return fetchSearch(query)
	case req.Route == "/qianbi/chapters/:id" || strings.HasPrefix(req.Route, "/qianbi/chapters"):
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchChapters(id)
	case req.Route == "/qianbi/chapter/:chapterId" || strings.HasPrefix(req.Route, "/qianbi/chapter"):
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
	if next := reListNext.FindStringSubmatch(htmlBody); len(next) > 1 {
		result.HasMore = true
		result.Next = copyParams(params)
		result.Next["page"] = strconv.Itoa(page + 1)
	}
	return result, nil
}

func fetchSearch(query string) (*sdk.FeedResult, error) {
	searchURL := baseURL + "/search.html?" + url.Values{
		"searchkey": {query},
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

func fetchChapters(bookID string) (*sdk.FeedResult, error) {
	pageURL := bookURL(bookID)
	listURL := fmt.Sprintf("%s/book/%s/catalog", baseURL, bookID)
	body, status, err := httpGet(listURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	htmlBody := string(body)
	chapters := parseChapterList(htmlBody, bookID)
	if len(chapters) == 0 {
		return nil, fmt.Errorf("no chapters found")
	}

	title := extractCatalogTitle(htmlBody)
	if title == "" {
		if detailBody, detailStatus, detailErr := httpGet(pageURL); detailErr == nil && detailStatus >= 200 && detailStatus < 300 {
			title = parseBookDetail(string(detailBody)).title
		}
	}
	if title == "" {
		title = bookID
	}

	detail := bookDetail{}
	if detailBody, detailStatus, detailErr := httpGet(pageURL); detailErr == nil && detailStatus >= 200 && detailStatus < 300 {
		detail = parseBookDetail(string(detailBody))
	}

	introItem := sdk.FeedItem{
		ID:    introChapterID,
		Title: "第0章 简介 / 详情",
		URL:   pageURL,
		Tags:  []string{"介绍"},
	}
	if detail.title != "" || detail.description != "" {
		introItem.Content = buildBookIntroHTML(detail, pageURL)
		introItem.Cover = detail.cover
		introItem.Image = detail.cover
		introItem.Author = detail.author
		introItem.Summary = detail.description
		if detail.category != "" {
			introItem.Tags = []string{detail.category}
		}
	}

	items := make([]sdk.FeedItem, 0, len(chapters)+1)
	items = append(items, introItem)
	items = append(items, chapters...)

	descParts := []string{fmt.Sprintf("共 %d 章", len(chapters))}
	if detail.author != "" {
		descParts = append(descParts, detail.author)
	}
	if detail.description != "" {
		descParts = append(descParts, detail.description)
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: strings.Join(descParts, " · "),
		Items:       items,
	}, nil
}

func fetchChapter(bookID, chapterID string) (*sdk.FeedResult, error) {
	if isIntroChapter(chapterID) {
		return fetchIntroChapter(bookID)
	}

	title, content, err := fetchChapterContent(bookID, chapterID)
	if err != nil {
		return nil, err
	}
	if content == "" {
		return nil, fmt.Errorf("chapter content not found")
	}

	chapterURL := chapterPageURL(bookID, chapterID, 1)
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

func fetchChapterContent(bookID, chapterID string) (string, string, error) {
	var parts []string
	title := ""
	page := 1

	for {
		pageURL := chapterPageURL(bookID, chapterID, page)
		body, status, err := httpGet(pageURL)
		if err != nil {
			return "", "", err
		}
		if status < 200 || status >= 300 {
			if page == 1 {
				return "", "", fmt.Errorf("http status %d", status)
			}
			break
		}

		htmlBody := string(body)
		if vip := firstMatch(reChapterVIP, htmlBody); vip == "1" {
			return "", "", fmt.Errorf("vip chapter not supported")
		}

		pageTitle, pageContent := extractChapter(htmlBody)
		if pageContent == "" {
			if page == 1 {
				return "", "", fmt.Errorf("chapter content not found")
			}
			break
		}

		if title == "" {
			title = pageTitle
		}
		parts = append(parts, pageContent)

		if !hasChapterSubPage(htmlBody, bookID, chapterID, page+1) {
			break
		}
		page++
	}

	if title == "" {
		title = "正文"
	}

	combined := strings.Join(parts, "\n")
	content := fmt.Sprintf(`<div class="chapter-content">%s</div>`, combined)
	return title, content, nil
}

func hasChapterSubPage(htmlBody, bookID, chapterID string, nextPage int) bool {
	pattern := fmt.Sprintf(`(?is)href=['"]/book/%s/%s_%d\.html['"]`, regexp.QuoteMeta(bookID), regexp.QuoteMeta(chapterID), nextPage)
	return regexp.MustCompile(pattern).MatchString(htmlBody)
}

func parseBookList(htmlBody string) []sdk.FeedItem {
	items := parseModuleBookList(htmlBody)
	if len(items) == 0 {
		items = parseSearchBookList(htmlBody)
	}
	return items
}

func parseModuleBookList(htmlBody string) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, 32)
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
		bookID := strings.TrimSpace(hrefMatch[1])
		if bookID == "" || seen[bookID] {
			continue
		}

		title := ""
		if m := reBookTitle.FindStringSubmatch(block); len(m) > 2 {
			title = cleanText(firstNonEmpty(m[1], m[2]))
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

		seen[bookID] = true
		items = append(items, sdk.FeedItem{
			ID:          bookID,
			Title:       title,
			URL:         bookURL(bookID),
			Author:      author,
			Cover:       cover,
			Image:       cover,
			PublishedAt: time.Now().Format(time.RFC3339),
		})
	}

	return items
}

func parseSearchBookList(htmlBody string) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, 16)
	seen := make(map[string]bool)

	for _, block := range strings.Split(htmlBody, `class="module-search-item"`) {
		if len(block) < 64 {
			continue
		}

		m := reSearchTitle.FindStringSubmatch(block)
		if len(m) < 4 {
			continue
		}
		bookID := strings.TrimSpace(m[1])
		if bookID == "" || seen[bookID] {
			continue
		}

		title := cleanText(firstNonEmpty(m[2], m[3]))
		if title == "" {
			continue
		}

		cover := ""
		if cm := reBookCover.FindStringSubmatch(block); len(cm) > 1 {
			cover = normalizeImageURL(cm[1])
		}

		summary := ""
		if sm := reSearchSummary.FindStringSubmatch(block); len(sm) > 1 {
			summary = truncate(cleanText(sm[1]), 120)
		}

		seen[bookID] = true
		items = append(items, sdk.FeedItem{
			ID:          bookID,
			Title:       title,
			URL:         bookURL(bookID),
			Summary:     summary,
			Cover:       cover,
			Image:       cover,
			PublishedAt: time.Now().Format(time.RFC3339),
		})
	}

	return items
}

func parseChapterList(htmlBody, bookID string) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, 512)
	seen := make(map[string]bool)

	for _, match := range reChapterLink.FindAllStringSubmatch(htmlBody, -1) {
		if len(match) < 3 {
			continue
		}
		chapterID := strings.TrimSpace(match[1])
		chTitle := cleanText(firstNonEmpty(match[2]))
		if chapterID == "" || chTitle == "" || seen[chapterID] {
			continue
		}
		seen[chapterID] = true

		chapterURL := chapterPageURL(bookID, chapterID, 1)
		items = append(items, sdk.FeedItem{
			ID:          chapterID,
			Title:       chTitle,
			URL:         chapterURL,
			PublishedAt: time.Now().Format(time.RFC3339),
		})
	}

	return items
}

func isIntroChapter(chapterID string) bool {
	switch strings.TrimSpace(chapterID) {
	case introChapterID, introID:
		return true
	default:
		return false
	}
}

func extractCatalogTitle(htmlBody string) string {
	return firstMatch(reCatalogTitle, htmlBody)
}

func extractChapter(htmlBody string) (string, string) {
	title := firstMatch(reReadParams, htmlBody)
	if title == "" {
		raw := firstMatch(rePageTitle, htmlBody)
		title = strings.TrimSuffix(raw, "_铅笔小说")
		if idx := strings.Index(title, "正文卷 "); idx >= 0 {
			title = strings.TrimPrefix(title[idx:], "正文卷 ")
		}
	}

	content := ""
	if m := reChapterBody.FindStringSubmatch(htmlBody); len(m) > 1 {
		content = strings.TrimSpace(m[1])
	}

	return title, content
}

func fetchIntroChapter(bookID string) (*sdk.FeedResult, error) {
	pageURL := bookURL(bookID)
	body, status, err := httpGet(pageURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	detail := parseBookDetail(string(body))
	title := firstNonEmpty(detail.title, bookID)
	content := buildBookIntroHTML(detail, pageURL)

	item := sdk.FeedItem{
		ID:      introChapterID,
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
	} else if len(detail.tags) > 0 {
		item.Tags = detail.tags
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
	tags        []string
}

func parseBookDetail(htmlBody string) bookDetail {
	detail := bookDetail{
		title:       firstMatch(reOGTitle, htmlBody),
		author:      firstMatch(reOGAuthor, htmlBody),
		category:    firstMatch(reOGCategory, htmlBody),
		status:      firstMatch(reOGStatus, htmlBody),
		cover:       normalizeImageURL(firstMatch(reOGImage, htmlBody)),
		description: cleanText(firstMatch(reOGDesc, htmlBody)),
	}

	if tags := strings.TrimSpace(firstMatch(reOGTags, htmlBody)); tags != "" {
		for _, t := range strings.Fields(tags) {
			t = strings.TrimSpace(t)
			if t != "" {
				detail.tags = append(detail.tags, t)
			}
		}
		if detail.category == "" && len(detail.tags) > 0 {
			detail.category = detail.tags[0]
		}
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

func bookURL(bookID string) string {
	return fmt.Sprintf("%s/book/%s/", baseURL, bookID)
}

func chapterPageURL(bookID, chapterID string, page int) string {
	if page <= 1 {
		return fmt.Sprintf("%s/book/%s/%s.html", baseURL, bookID, chapterID)
	}
	return fmt.Sprintf("%s/book/%s/%s_%d.html", baseURL, bookID, chapterID, page)
}

func categoryURL(slug string, page int) string {
	catID := categoryIDs[slug]
	return fmt.Sprintf("%s/book/lastupdate_0_%d_0_0_0_0_0_%d_0.html", baseURL, catID, page)
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

func httpGet(rawURL string) ([]byte, int, error) {
	body, status, err := host.HTTPGet(rawURL, map[string]string{
		"User-Agent": userAgent,
		"Accept":     "text/html,application/xhtml+xml",
		"Referer":    baseURL + "/",
	})
	if err != nil {
		return nil, 0, fmt.Errorf("http get failed: %w", err)
	}
	return body, status, nil
}
