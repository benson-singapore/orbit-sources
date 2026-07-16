package main

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL            = "https://www.nodeseek.com"
	defaultAvatarURL   = baseURL + "/favicon.ico"
	defaultUserAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
	defaultPreviewSize = 5
	defaultPageSize    = 20
)

var (
	postURLPattern      = regexp.MustCompile(`/post-(\d+)-\d+`)
	blockTagPattern     = regexp.MustCompile(`(?i)</?(div|section|article|header|footer|main|aside|nav|ul|ol|li|p|pre|blockquote|h[1-6]|table|tr|td|th)[^>]*>`)
	brPattern           = regexp.MustCompile(`(?i)<br\s*/?>`)
	scriptStylePattern  = regexp.MustCompile(`(?is)<(script|style|noscript)[^>]*>.*?</(script|style|noscript)>`)
	tagPattern          = regexp.MustCompile(`(?s)<[^>]+>`)
	multiBlankPattern   = regexp.MustCompile(`\n{3,}`)
	spacePattern        = regexp.MustCompile(`[ \t\r\f\v]+`)
	floorLinePattern    = regexp.MustCompile(`^#\d+$`)
	relativeTimePattern = regexp.MustCompile(`(\d+)\s*(minute|minutes|hour|hours|day|days)\s+ago`)
	absoluteDatePattern = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}(?: \d{2}:\d{2})?\b`)
	boilerplateLineSet  = map[string]struct{}{"所有版块": {}, "快捷功能区": {}, "登录注册": {}, "推荐阅读": {}, "管理记录": {}, "幸运抽奖": {}, "邀请好友": {}, "合作商家": {}, "友站链接": {}, "loading...": {}, "新评论新帖子": {}}
	categoryLabelMap    = map[string]string{"daily": "日常", "tech": "技术", "news": "情报", "review": "测评", "trade": "交易", "carpool": "拼车", "promo": "推广", "life": "生活", "dev": "Dev", "pic": "贴图", "expose": "曝光", "pointless": "无意义", "sandbox": "沙盒"}
)

func main() {
	sdk.Run(&NodeSeekPlugin{})
}

type NodeSeekPlugin struct{}

type postRef struct {
	ID    string
	Title string
	URL   string
}

type postMeta struct {
	ID          string
	Title       string
	URL         string
	Author      string
	Avatar      string
	Handle      string
	Summary     string
	Content     string
	PublishedAt string
	Category    string
	Cover       string
	Image       string
	Media       []sdk.SocialMedia
	Stats       *sdk.SocialStats
	Kind        string
}

type commentBlock struct {
	Floor string
	Body  string
}

func (p *NodeSeekPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	cookie := strings.TrimSpace(req.Var("cookie"))

	switch {
	case req.Route == "/nodeseek/home":
		return fetchHome(cookie)
	case strings.HasPrefix(req.Route, "/nodeseek/category"):
		slug := strings.TrimSpace(req.Params["slug"])
		if slug == "" {
			return nil, fmt.Errorf("missing slug parameter")
		}
		page := parsePage(req.Params["page"])
		return fetchCategory(slug, page, cookie)
	case strings.HasPrefix(req.Route, "/nodeseek/post"):
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id, cookie)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchHome(cookie string) (*sdk.FeedResult, error) {
	doc, _, err := fetchDocument(baseURL+"/", cookie)
	if err != nil {
		return nil, err
	}

	refs := extractPostRefs(doc)
	if len(refs) == 0 {
		return nil, fmt.Errorf("no posts found on homepage")
	}

	items := buildListItems(refs)
	if err := enrichListItems(items, cookie, defaultPreviewSize); err != nil {
		// Best effort: fall back to title-only cards when previews fail.
	}

	return &sdk.FeedResult{
		Title:       "NodeSeek",
		Description: "NodeSeek 首页帖子流",
		Items:       items,
	}, nil
}

func fetchCategory(slug string, page int, cookie string) (*sdk.FeedResult, error) {
	categoryURL := baseURL + "/categories/" + url.PathEscape(slug)
	if page > 1 {
		categoryURL += "/page-" + strconv.Itoa(page)
	}

	doc, _, err := fetchDocument(categoryURL, cookie)
	if err != nil {
		return nil, err
	}

	refs := extractPostRefs(doc)
	if len(refs) == 0 {
		return nil, fmt.Errorf("no posts found in category %s", slug)
	}

	items := buildListItems(refs)
	if err := enrichListItems(items, cookie, defaultPreviewSize); err != nil {
		// Keep the list usable even if preview hydration fails.
	}

	label := categoryLabelMap[slug]
	if label == "" {
		label = slug
	}

	result := &sdk.FeedResult{
		Title:       "NodeSeek · " + label,
		Description: "NodeSeek " + label + "版块",
		Items:       items,
	}
	if len(items) >= defaultPageSize {
		result.HasMore = true
		result.Next = map[string]string{"page": strconv.Itoa(page + 1)}
	}
	return result, nil
}

func fetchDetail(id, cookie string) (*sdk.FeedResult, error) {
	meta, err := fetchPostMeta(id, cookie)
	if err != nil {
		return nil, err
	}

	item := sdk.FeedItem{
		ID:           meta.ID,
		Title:        meta.Title,
		URL:          meta.URL,
		Summary:      meta.Summary,
		Content:      meta.Content,
		Author:       meta.Author,
		AuthorAvatar: meta.Avatar,
		AuthorHandle: meta.Handle,
		PublishedAt:  meta.PublishedAt,
		Kind:         meta.Kind,
		Stats:        meta.Stats,
		Media:        meta.Media,
		Cover:        meta.Cover,
		Image:        meta.Image,
	}
	if meta.Category != "" {
		item.Tags = append(item.Tags, meta.Category)
	}

	return &sdk.FeedResult{
		Title:       meta.Title,
		Description: meta.Summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func fetchPostMeta(id, cookie string) (*postMeta, error) {
	postID := normalizePostID(id)
	if postID == "" {
		return nil, fmt.Errorf("invalid post id: %s", id)
	}

	postURL := canonicalPostURL(postID)
	doc, raw, err := fetchDocument(postURL, cookie)
	if err != nil {
		return nil, err
	}

	text := htmlToText(raw)
	lines := splitMeaningfulLines(text)
	sections := parseFloorSections(lines)
	mainBody := strings.TrimSpace(sections["#0"])
	if mainBody == "" {
		mainBody = fallbackMainBody(lines)
	}
	if strings.TrimSpace(mainBody) == "" {
		return nil, fmt.Errorf("post body not found: %s", postID)
	}

	comments := collectComments(sections)
	title := firstNonEmpty(
		cleanText(metaContent(doc, `meta[property="og:title"]`)),
		cleanText(metaContent(doc, `meta[name="twitter:title"]`)),
		cleanText(extractHeadingTitle(lines)),
		cleanText(doc.Find("title").First().Text()),
		"NodeSeek Post",
	)
	title = cleanupTitle(title)

	author, avatar := extractAuthorInfo(doc)
	if author == "" {
		author = "NodeSeek"
	}
	if avatar == "" {
		avatar = defaultAvatarURL
	}

	publishedAt, category := extractPublishedMeta(lines)
	media, cover := extractMedia(doc)

	content := buildDetailContent(title, mainBody, comments, media)
	summary := truncateRunes(firstNonEmpty(firstContentLine(mainBody), title), 160)
	stats := &sdk.SocialStats{
		Replies: len(comments),
	}

	return &postMeta{
		ID:          postID,
		Title:       title,
		URL:         postURL,
		Author:      author,
		Avatar:      avatar,
		Summary:     summary,
		Content:     content,
		PublishedAt: publishedAt,
		Category:    category,
		Cover:       cover,
		Image:       cover,
		Media:       media,
		Stats:       stats,
		Kind:        classifyKind(mainBody, len(media), len(comments)),
	}, nil
}

func fetchDocument(pageURL, cookie string) (*goquery.Document, []byte, error) {
	body, status, err := host.HTTPGet(pageURL, requestHeaders(cookie))
	if err != nil {
		return nil, nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, nil, fmt.Errorf("nodeseek returned http status %d", status)
	}
	if isCloudflareChallenge(body) {
		return nil, nil, fmt.Errorf("nodeseek is protected by Cloudflare; configure plugin variable cookie with a valid browser session")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("parse html: %w", err)
	}
	return doc, body, nil
}

func buildListItems(refs []postRef) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, len(refs))
	for _, ref := range refs {
		if ref.ID == "" || ref.Title == "" {
			continue
		}
		items = append(items, sdk.FeedItem{
			ID:           ref.ID,
			Title:        ref.Title,
			URL:          ref.URL,
			Summary:      ref.Title,
			Content:      html.EscapeString(ref.Title),
			Author:       "NodeSeek",
			AuthorAvatar: defaultAvatarURL,
			PublishedAt:  time.Now().UTC().Format(time.RFC3339),
			Kind:         "short",
			Stats:        &sdk.SocialStats{},
		})
	}
	return items
}

func enrichListItems(items []sdk.FeedItem, cookie string, limit int) error {
	if limit > len(items) {
		limit = len(items)
	}
	var firstErr error
	for i := 0; i < limit; i++ {
		meta, err := fetchPostMeta(items[i].ID, cookie)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		items[i].Summary = firstNonEmpty(meta.Summary, items[i].Summary)
		items[i].Content = firstNonEmpty(meta.Content, items[i].Content)
		items[i].Author = firstNonEmpty(meta.Author, items[i].Author)
		items[i].AuthorAvatar = firstNonEmpty(meta.Avatar, items[i].AuthorAvatar)
		items[i].AuthorHandle = firstNonEmpty(meta.Handle, items[i].AuthorHandle)
		items[i].PublishedAt = firstNonEmpty(meta.PublishedAt, items[i].PublishedAt)
		items[i].Kind = firstNonEmpty(meta.Kind, items[i].Kind)
		items[i].Stats = meta.Stats
		items[i].Media = meta.Media
		items[i].Cover = meta.Cover
		items[i].Image = meta.Image
		if meta.Category != "" {
			items[i].Tags = appendUnique(items[i].Tags, meta.Category)
		}
	}
	return firstErr
}

func extractPostRefs(doc *goquery.Document) []postRef {
	seen := map[string]struct{}{}
	refs := make([]postRef, 0, 32)

	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		href, ok := sel.Attr("href")
		if !ok {
			return
		}
		resolved := absolutizeURL(href)
		postID := normalizePostID(resolved)
		if postID == "" {
			return
		}
		if _, exists := seen[postID]; exists {
			return
		}

		title := cleanText(sel.Text())
		if !looksLikePostTitle(title) {
			return
		}

		seen[postID] = struct{}{}
		refs = append(refs, postRef{
			ID:    postID,
			Title: title,
			URL:   canonicalPostURL(postID),
		})
	})

	return refs
}

func extractAuthorInfo(doc *goquery.Document) (string, string) {
	author := cleanText(metaContent(doc, `meta[name="author"]`))
	avatar := ""

	doc.Find(`a[href*="/space/"]`).EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		name := cleanText(sel.Text())
		if author == "" && name != "" && !isBoilerplateLine(name) {
			author = name
		}
		if author != "" && cleanText(sel.Text()) == author {
			avatar = firstNonEmpty(
				resolveImageSelection(sel.Parent().Find("img").First()),
				resolveImageSelection(sel.PrevAllFiltered("img").First()),
				resolveImageSelection(sel.NextAllFiltered("img").First()),
				avatar,
			)
			return false
		}
		return true
	})

	if avatar == "" {
		doc.Find(`img[src*="avatar"], img[class*="avatar"], img[alt*="avatar"]`).EachWithBreak(func(_ int, sel *goquery.Selection) bool {
			avatar = resolveImageSelection(sel)
			return avatar == ""
		})
	}

	return author, avatar
}

func extractPublishedMeta(lines []string) (string, string) {
	now := time.Now().UTC()
	for _, line := range lines {
		line = cleanText(line)
		if !strings.Contains(line, "ago") && !strings.Contains(line, " in") && !absoluteDatePattern.MatchString(line) {
			continue
		}

		category := ""
		if idx := strings.LastIndex(line, " in"); idx >= 0 {
			category = cleanText(line[idx+3:])
		}

		if match := relativeTimePattern.FindStringSubmatch(line); len(match) == 3 {
			value, _ := strconv.Atoi(match[1])
			unit := match[2]
			t := now
			switch unit {
			case "minute", "minutes":
				t = now.Add(-time.Duration(value) * time.Minute)
			case "hour", "hours":
				t = now.Add(-time.Duration(value) * time.Hour)
			default:
				t = now.AddDate(0, 0, -value)
			}
			return t.Format(time.RFC3339), category
		}

		if raw := absoluteDatePattern.FindString(line); raw != "" {
			layout := "2006-01-02"
			if strings.Contains(raw, ":") {
				layout = "2006-01-02 15:04"
			}
			if parsed, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
				return parsed.UTC().Format(time.RFC3339), category
			}
		}
	}

	return now.Format(time.RFC3339), ""
}

func extractMedia(doc *goquery.Document) ([]sdk.SocialMedia, string) {
	media := make([]sdk.SocialMedia, 0, 4)
	seen := map[string]struct{}{}

	addImage := func(imageURL string) {
		imageURL = absolutizeURL(imageURL)
		if imageURL == "" || !strings.HasPrefix(imageURL, "http") {
			return
		}
		if _, exists := seen[imageURL]; exists {
			return
		}
		seen[imageURL] = struct{}{}
		media = append(media, sdk.SocialMedia{
			Type:      "image",
			URL:       imageURL,
			Thumbnail: imageURL,
		})
	}

	addImage(metaContent(doc, `meta[property="og:image"]`))
	addImage(metaContent(doc, `meta[name="twitter:image"]`))

	doc.Find("img[src]").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		src := resolveImageSelection(sel)
		if src != "" && !strings.Contains(src, "avatar") && !strings.Contains(src, "favicon") {
			addImage(src)
		}
		return len(media) < 4
	})

	cover := ""
	if len(media) > 0 {
		cover = media[0].URL
	}
	return media, cover
}

func parseFloorSections(lines []string) map[string]string {
	sections := make(map[string]string)
	currentFloor := ""
	var currentLines []string

	flush := func() {
		if currentFloor == "" {
			return
		}
		body := cleanupSectionBody(currentLines)
		if body != "" {
			sections[currentFloor] = body
		}
	}

	for _, line := range lines {
		line = cleanText(line)
		if line == "" {
			if currentFloor != "" {
				currentLines = append(currentLines, "")
			}
			continue
		}
		if line == "登录或者注册后评论." {
			break
		}
		if floorLinePattern.MatchString(line) {
			flush()
			currentFloor = line
			currentLines = currentLines[:0]
			continue
		}
		if currentFloor != "" {
			currentLines = append(currentLines, line)
		}
	}
	flush()

	return sections
}

func cleanupSectionBody(lines []string) string {
	lines = trimEmpty(lines)
	for len(lines) > 0 && isStatLine(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && isBoilerplateLine(lines[0]) {
		lines = lines[1:]
	}
	lines = trimEmpty(lines)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func collectComments(sections map[string]string) []commentBlock {
	floors := make([]string, 0, len(sections))
	for floor := range sections {
		if floor == "#0" {
			continue
		}
		floors = append(floors, floor)
	}
	sort.Slice(floors, func(i, j int) bool {
		return floorNumber(floors[i]) < floorNumber(floors[j])
	})

	comments := make([]commentBlock, 0, len(floors))
	for _, floor := range floors {
		body := strings.TrimSpace(sections[floor])
		if body == "" {
			continue
		}
		comments = append(comments, commentBlock{Floor: floor, Body: body})
	}
	return comments
}

func buildDetailContent(title, body string, comments []commentBlock, media []sdk.SocialMedia) string {
	var builder strings.Builder

	if title != "" {
		builder.WriteString(fmt.Sprintf("<h2>%s</h2>", html.EscapeString(title)))
	}
	for _, attachment := range media {
		if attachment.Type == "image" && attachment.URL != "" {
			builder.WriteString(fmt.Sprintf("<p><img src=\"%s\" style=\"max-width: 100%%; border-radius: 8px;\"/></p>", html.EscapeString(attachment.URL)))
		}
	}
	builder.WriteString(renderPlainText(body))

	if len(comments) > 0 {
		builder.WriteString("<h3>第一页评论</h3>")
		for _, comment := range comments {
			builder.WriteString(fmt.Sprintf(
				"<blockquote><p><strong>%s</strong></p>%s</blockquote>",
				html.EscapeString(comment.Floor),
				renderPlainText(comment.Body),
			))
		}
	}

	return builder.String()
}

func renderPlainText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "\n")
	var builder strings.Builder
	for _, part := range parts {
		line := strings.TrimSpace(part)
		if line == "" {
			continue
		}
		builder.WriteString("<p>")
		builder.WriteString(html.EscapeString(line))
		builder.WriteString("</p>")
	}
	return builder.String()
}

func htmlToText(body []byte) string {
	value := string(body)
	value = scriptStylePattern.ReplaceAllString(value, "\n")
	value = brPattern.ReplaceAllString(value, "\n")
	value = blockTagPattern.ReplaceAllString(value, "\n")
	value = tagPattern.ReplaceAllString(value, "")
	value = html.UnescapeString(value)
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = spacePattern.ReplaceAllString(value, " ")
	value = multiBlankPattern.ReplaceAllString(value, "\n\n")
	return strings.TrimSpace(value)
}

func splitMeaningfulLines(value string) []string {
	rawLines := strings.Split(value, "\n")
	lines := make([]string, 0, len(rawLines))
	lastBlank := false
	for _, line := range rawLines {
		cleaned := cleanText(line)
		if cleaned == "" {
			if !lastBlank {
				lines = append(lines, "")
				lastBlank = true
			}
			continue
		}
		lines = append(lines, cleaned)
		lastBlank = false
	}
	return trimEmpty(lines)
}

func fallbackMainBody(lines []string) string {
	start := -1
	end := len(lines)
	for i, line := range lines {
		if line == "#0" {
			start = i + 1
			continue
		}
		if start >= 0 && line == "登录或者注册后评论." {
			end = i
			break
		}
	}
	if start < 0 || start >= end {
		return ""
	}
	return cleanupSectionBody(lines[start:end])
}

func extractHeadingTitle(lines []string) string {
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func classifyKind(body string, mediaCount, commentCount int) string {
	if commentCount > 0 || mediaCount > 1 {
		return "long"
	}
	if len([]rune(body)) > 280 {
		return "long"
	}
	return "short"
}

func metaContent(doc *goquery.Document, selector string) string {
	content, _ := doc.Find(selector).First().Attr("content")
	return strings.TrimSpace(content)
}

func cleanupTitle(value string) string {
	value = cleanText(value)
	value = strings.TrimSuffix(value, " - NodeSeek")
	value = strings.TrimSuffix(value, " | NodeSeek")
	return strings.TrimSpace(value)
}

func looksLikePostTitle(value string) bool {
	value = cleanText(value)
	if value == "" || len([]rune(value)) < 3 {
		return false
	}
	if isBoilerplateLine(value) {
		return false
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return false
	}
	return true
}

func isBoilerplateLine(value string) bool {
	value = cleanText(value)
	_, exists := boilerplateLineSet[value]
	return exists
}

func isStatLine(value string) bool {
	value = cleanText(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func floorNumber(floor string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(floor, "#"))
	return n
}

func parsePage(value string) int {
	page, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func normalizePostID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if match := postURLPattern.FindStringSubmatch(value); len(match) == 2 {
		return match[1]
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return value
}

func canonicalPostURL(postID string) string {
	return baseURL + "/post-" + postID + "-1"
}

func absolutizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(rawURL, "http://"), strings.HasPrefix(rawURL, "https://"):
		return rawURL
	case strings.HasPrefix(rawURL, "//"):
		return "https:" + rawURL
	case strings.HasPrefix(rawURL, "/"):
		return baseURL + rawURL
	default:
		return baseURL + "/" + strings.TrimPrefix(rawURL, "./")
	}
}

func resolveImageSelection(sel *goquery.Selection) string {
	if sel == nil || sel.Length() == 0 {
		return ""
	}
	for _, attr := range []string{"src", "data-src", "data-original", "srcset"} {
		if value, ok := sel.Attr(attr); ok && strings.TrimSpace(value) != "" {
			if attr == "srcset" {
				value = strings.Fields(value)[0]
			}
			return absolutizeURL(value)
		}
	}
	return ""
}

func requestHeaders(cookie string) map[string]string {
	headers := map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
		"Cache-Control":   "no-cache",
		"Pragma":          "no-cache",
		"Referer":         baseURL + "/",
		"User-Agent":      defaultUserAgent,
	}
	if cookie != "" {
		headers["Cookie"] = cookie
	}
	return headers
}

func isCloudflareChallenge(body []byte) bool {
	value := strings.ToLower(string(body))
	return strings.Contains(value, "cf-mitigated") ||
		strings.Contains(value, "challenges.cloudflare.com") ||
		strings.Contains(value, "attention required") ||
		strings.Contains(value, "just a moment")
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func cleanText(value string) string {
	value = html.UnescapeString(value)
	value = strings.TrimSpace(value)
	value = spacePattern.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func trimEmpty(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func firstContentLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "…"
}
