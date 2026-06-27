package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	siteName  = "包子漫画"
	baseURL   = "https://baozimh.org"
	apiBase   = "https://v2.apikk.top"
	coverBase = "https://c-nc-1.6wm.top/manga/"
	pageLimit = 36
	userAgent = "Mozilla/5.0 (compatible; OrbitPlugins/1.0)"
)

var channelPaths = map[string]string{
	"baozi_hot":      "/manga-genre/hots",
	"baozi_cn":       "/manga-genre/cn",
	"baozi_jp":       "/manga-genre/jp",
	"baozi_kr":       "/manga-genre/kr",
	"baozi_en":       "/manga-genre/ou-mei",
	"baozi_lianai":   "/manga-tag/lianai",
	"baozi_gufeng":   "/manga-tag/gufeng",
	"baozi_xuanhuan": "/manga-tag/xuanhuan",
	"baozi_yineng":   "/manga-tag/yineng",
	"baozi_xuanyi":   "/manga-tag/xuanyi",
	"baozi_kehuan":   "/manga-tag/kehuan",
	"baozi_chuanyue": "/manga-tag/chuanyue",
	"baozi_mouxian":  "/manga-tag/mouxian",
	"baozi_rexie":    "/manga-tag/rexie",
	"baozi_gaoxiao":  "/manga-tag/gaoxiao",
	"baozi_dushi":    "/manga-tag/dushi",
	"baozi_hougong":  "/manga-tag/hougong",
	"baozi_qita":     "/manga-genre/qita",
}

var (
	reComicCard = regexp.MustCompile(`(?is)<a\s+href="/manga/([^"]+)"[^>]*>.*?<img[^>]+class="card"[^>]+src="([^"]+)"[^>]*>.*?<h3[^>]*class="cardtitle"[^>]*>([^<]*)</h3>`)
	reMangaMID  = regexp.MustCompile(`data-mid="(\d+)"`)

	reComicTitle  = regexp.MustCompile(`(?is)<meta[^>]*property="og:title"[^>]*content="([^"]*)"`)
	reComicAuthor = regexp.MustCompile(`(?is)<meta[^>]*(?:name|property)="og:novel:author"[^>]*content="([^"]*)"`)
	reComicCover  = regexp.MustCompile(`(?is)<meta[^>]*property="og:image"[^>]*content="([^"]*)"`)
	reComicDesc   = regexp.MustCompile(`(?is)<meta[^>]*property="og:description"[^>]*content="([^"]*)"`)
	reDetailTitle = regexp.MustCompile(`(?is)<h1[^>]*class="[^"]*title[^"]*"[^>]*>\s*([^<]+?)\s*</h1>`)
)

var channelLabels = map[string]string{
	"baozi_hot":      "热门漫画",
	"baozi_cn":       "国漫",
	"baozi_jp":       "日漫",
	"baozi_kr":       "韩漫",
	"baozi_en":       "欧美",
	"baozi_lianai":   "恋爱",
	"baozi_gufeng":   "古风",
	"baozi_xuanhuan": "玄幻",
	"baozi_yineng":   "异能",
	"baozi_xuanyi":   "悬疑",
	"baozi_kehuan":   "科幻",
	"baozi_chuanyue": "穿越",
	"baozi_mouxian":  "冒险",
	"baozi_rexie":    "热血",
	"baozi_gaoxiao":  "搞笑",
	"baozi_dushi":    "都市",
	"baozi_hougong":  "后宫",
	"baozi_qita":     "其他",
	"baozi_search":   "搜索",
}

type mangaAPIResponse struct {
	Status bool `json:"status"`
	Data   struct {
		ID       string         `json:"id"`
		Title    string         `json:"title"`
		Slug     string         `json:"slug"`
		Cover    string         `json:"cover"`
		Desc     string       `json:"desc"`
		Chapters []apiChapter `json:"chapters"`
	} `json:"data"`
}

type apiChapter struct {
	ID         string `json:"id"`
	Attributes struct {
		Title     string `json:"title"`
		Slug      string `json:"slug"`
		Order     int    `json:"order"`
		UpdatedAt string `json:"updatedAt"`
	} `json:"attributes"`
}

type chapterGetResponse struct {
	Status bool `json:"status"`
	Data   struct {
		Info struct {
			Title  string `json:"title"`
			Slug   string `json:"slug"`
			Images struct {
				Images string `json:"images"`
				Line   int    `json:"line"`
			} `json:"images"`
		} `json:"info"`
	} `json:"data"`
}

type decodedPage struct {
	Order int    `json:"order"`
	URL   string `json:"url"`
}

func main() {
	sdk.Run(&BaoziPlugin{})
}

type BaoziPlugin struct{}

func (p *BaoziPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/baozi/list" || strings.HasPrefix(req.Route, "/baozi/list"):
		return fetchList(req.ChannelID, req.Params)
	case req.Route == "/baozi/search" || strings.HasPrefix(req.Route, "/baozi/search"):
		query := strings.TrimSpace(req.Params["query"])
		if query == "" {
			return nil, fmt.Errorf("missing query parameter")
		}
		return fetchSearch(query)
	case req.Route == "/baozi/chapters/:id" || strings.HasPrefix(req.Route, "/baozi/chapters"):
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchChapters(id)
	case req.Route == "/baozi/chapter/:chapterId" || strings.HasPrefix(req.Route, "/baozi/chapter"):
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

func fetchList(channelID string, params map[string]string) (*sdk.FeedResult, error) {
	listPath, ok := channelPaths[channelID]
	if !ok {
		return nil, fmt.Errorf("unknown channel: %s", channelID)
	}

	page := pageNum(params)
	listURL := baseURL + listPagePath(listPath, page)

	body, err := httpGet(listURL)
	if err != nil {
		return nil, err
	}
	htmlText := string(body)

	items := filterUnseen(parseComicCardsHTML(htmlText), parseSeenIDs(params["seenIds"]))
	if len(items) == 0 {
		return nil, fmt.Errorf("empty comic list")
	}

	result := &sdk.FeedResult{
		Title:       siteName + " · " + channelTitle(channelID),
		Description: fmt.Sprintf("第 %d 页 · %d 部作品", page, len(items)),
		Items:       items,
	}
	if hasNextPage(htmlText, listPath, page) {
		result.HasMore = true
		result.Next = copyParams(params)
		result.Next["page"] = strconv.Itoa(page + 1)
		result.Next["seenIds"] = formatSeenIDs(appendSeenIDs(parseSeenIDs(params["seenIds"]), items))
	}
	return result, nil
}

func fetchSearch(query string) (*sdk.FeedResult, error) {
	searchURL := baseURL + "/s?" + url.Values{"q": {query}}.Encode()
	body, err := httpGet(searchURL)
	if err != nil {
		return nil, err
	}
	htmlText := string(body)

	items := parseComicCardsHTML(htmlText)
	if len(items) == 0 {
		return nil, fmt.Errorf("no results for: %s", query)
	}

	return &sdk.FeedResult{
		Title:       siteName + " · 搜索",
		Description: fmt.Sprintf("关键词「%s」· %d 部作品", query, len(items)),
		Items:       items,
	}, nil
}

func fetchChapters(comicSlug string) (*sdk.FeedResult, error) {
	pageURL := mangaURL(comicSlug)
	body, err := httpGet(pageURL)
	if err != nil {
		return nil, err
	}
	htmlText := string(body)

	mid := firstMatch(reMangaMID, htmlText)
	if mid == "" {
		return nil, fmt.Errorf("manga id not found: %s", comicSlug)
	}

	apiURL := apiBase + "/api/manga/get?mid=" + url.QueryEscape(mid) + "&mode=all"
	apiBody, err := httpGet(apiURL, refererHeaders(pageURL))
	if err != nil {
		return nil, err
	}

	var resp mangaAPIResponse
	if err := json.Unmarshal(apiBody, &resp); err != nil {
		return nil, fmt.Errorf("parse manga api: %w", err)
	}
	if !resp.Status || len(resp.Data.Chapters) == 0 {
		return nil, fmt.Errorf("no chapters for: %s", comicSlug)
	}

	title := firstNonEmpty(resp.Data.Title, cleanOGTitle(htmlText), comicSlug)
	desc := strings.TrimSpace(resp.Data.Desc)
	if desc == "" {
		desc = firstMatch(reComicDesc, htmlText)
	}

	items := make([]sdk.FeedItem, 0, len(resp.Data.Chapters)+1)
	items = append(items, sdk.FeedItem{
		ID:    "intro",
		Title: "简介 / 详情",
		URL:   pageURL,
		Tags:  []string{"介绍"},
	})

	for _, ch := range resp.Data.Chapters {
		chTitle := strings.TrimSpace(ch.Attributes.Title)
		if chTitle == "" {
			chTitle = ch.Attributes.Slug
		}
		items = append(items, sdk.FeedItem{
			ID:    ch.ID,
			Title: chTitle,
			URL:   baseURL + "/manga/" + comicSlug + "/" + ch.Attributes.Slug,
		})
	}

	descParts := []string{fmt.Sprintf("共 %d 话", len(resp.Data.Chapters))}
	if desc != "" {
		descParts = append(descParts, truncate(desc, 80))
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: strings.Join(descParts, " · "),
		Items:       items,
	}, nil
}

func fetchChapter(parentID, chapterID string) (*sdk.FeedResult, error) {
	if chapterID == "intro" {
		return fetchIntroChapter(parentID)
	}

	pageURL := mangaURL(parentID)
	body, err := httpGet(pageURL)
	if err != nil {
		return nil, err
	}
	htmlText := string(body)

	mid := firstMatch(reMangaMID, htmlText)
	if mid == "" {
		return nil, fmt.Errorf("manga id not found: %s", parentID)
	}

	apiURL := apiBase + "/api/v2/chapter/getinfo?m=" + url.QueryEscape(mid) + "&c=" + url.QueryEscape(chapterID)
	apiBody, err := httpGet(apiURL, refererHeaders(pageURL))
	if err != nil {
		return nil, err
	}

	var resp chapterGetResponse
	if err := json.Unmarshal(apiBody, &resp); err != nil {
		return nil, fmt.Errorf("parse chapter api: %w", err)
	}
	if !resp.Status {
		return nil, fmt.Errorf("chapter api failed: %s", chapterID)
	}

	pages, err := decodeChapterImages(resp.Data.Info.Images.Images)
	if err != nil {
		return nil, fmt.Errorf("decode chapter images: %w", err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no images in chapter: %s", chapterID)
	}

	cdn := chapterCDN(resp.Data.Info.Images.Line)
	images := make([]string, 0, len(pages))
	sort.Slice(pages, func(i, j int) bool { return pages[i].Order < pages[j].Order })
	for _, p := range pages {
		imgURL := strings.TrimSpace(p.URL)
		if imgURL == "" {
			continue
		}
		if strings.HasPrefix(imgURL, "http://") || strings.HasPrefix(imgURL, "https://") {
			images = append(images, imgURL)
		} else {
			images = append(images, cdn+imgURL)
		}
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("no images in chapter: %s", chapterID)
	}

	title := firstNonEmpty(cleanOGTitle(htmlText), parentID)
	chapterTitle := strings.TrimSpace(resp.Data.Info.Title)
	fullTitle := title + " · " + chapterTitle
	cover := firstMatch(reComicCover, htmlText)
	contentJSON, err := json.Marshal(images)
	if err != nil {
		return nil, fmt.Errorf("marshal chapter images: %w", err)
	}

	item := sdk.FeedItem{
		ID:      chapterID,
		Title:   fullTitle,
		URL:     baseURL + "/manga/" + parentID + "/" + strings.TrimSpace(resp.Data.Info.Slug),
		Summary: fmt.Sprintf("共 %d 页", len(images)),
		Content: string(contentJSON),
		Cover:   cover,
		Image:   cover,
	}

	return &sdk.FeedResult{
		Title:       fullTitle,
		Description: siteName,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func fetchIntroChapter(comicSlug string) (*sdk.FeedResult, error) {
	pageURL := mangaURL(comicSlug)
	body, err := httpGet(pageURL)
	if err != nil {
		return nil, err
	}
	htmlText := string(body)

	title := firstNonEmpty(cleanOGTitle(htmlText), firstMatch(reDetailTitle, htmlText), comicSlug)
	cover := firstMatch(reComicCover, htmlText)
	desc := strings.TrimSpace(firstMatch(reComicDesc, htmlText))

	content := buildComicIntroHTML(title, "", cover, nil, "", desc, pageURL)
	item := sdk.FeedItem{
		ID:      "intro",
		Title:   title + " · 简介",
		URL:     pageURL,
		Content: content,
		Cover:   cover,
		Image:   cover,
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: siteName,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func parseComicCardsHTML(htmlText string) []sdk.FeedItem {
	seen := make(map[string]struct{})
	items := make([]sdk.FeedItem, 0)
	for _, m := range reComicCard.FindAllStringSubmatch(htmlText, -1) {
		if len(m) < 4 {
			continue
		}
		id := strings.TrimSpace(m[1])
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		cover := strings.TrimSpace(html.UnescapeString(m[2]))
		title := strings.TrimSpace(html.UnescapeString(m[3]))
		if title == "" {
			title = id
		}

		items = append(items, sdk.FeedItem{
			ID:      id,
			Title:   title,
			URL:     mangaURL(id),
			Cover:   cover,
			Image:   cover,
		})
	}
	return items
}

func filterUnseen(items []sdk.FeedItem, seen map[string]struct{}) []sdk.FeedItem {
	if len(seen) == 0 {
		return items
	}
	out := make([]sdk.FeedItem, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		out = append(out, item)
	}
	return out
}

func parseSeenIDs(raw string) map[string]struct{} {
	seen := make(map[string]struct{})
	for _, id := range strings.Split(raw, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			seen[id] = struct{}{}
		}
	}
	return seen
}

func appendSeenIDs(seen map[string]struct{}, items []sdk.FeedItem) map[string]struct{} {
	next := make(map[string]struct{}, len(seen)+len(items))
	for id := range seen {
		next[id] = struct{}{}
	}
	for _, item := range items {
		next[item.ID] = struct{}{}
	}
	return next
}

func formatSeenIDs(seen map[string]struct{}) string {
	if len(seen) == 0 {
		return ""
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func listPagePath(listPath string, page int) string {
	if page <= 1 {
		return listPath
	}
	// Site page 1 is the bare list path; page 2 is /page/1, page 3 is /page/2, etc.
	return listPath + "/page/" + strconv.Itoa(page-1)
}

func hasNextPage(htmlText, listPath string, page int) bool {
	next := listPath + "/page/" + strconv.Itoa(page)
	return strings.Contains(htmlText, `href="`+next+`"`)
}

func mangaURL(slug string) string {
	return baseURL + "/manga/" + url.PathEscape(slug)
}

func chapterCDN(line int) string {
	if line == 2 {
		return "https://f40-1-4.g-mh.online"
	}
	return "https://t40-1-4.g-mh.online"
}

func cleanOGTitle(htmlText string) string {
	title := firstMatch(reComicTitle, htmlText)
	title = strings.TrimSuffix(title, "-包子漫畫 - 包子漫畫")
	title = strings.TrimSuffix(title, "-包子漫画 - 包子漫画")
	return strings.TrimSpace(title)
}

func refererHeaders(referer string) map[string]string {
	return map[string]string{
		"Accept":          "application/json,text/html,*/*",
		"User-Agent":      userAgent,
		"Accept-Language": "zh-TW,zh;q=0.9",
		"Referer":         referer,
	}
}

func buildComicIntroHTML(title, author, cover string, tags []string, latest, desc, sourceURL string) string {
	var sb strings.Builder
	sb.WriteString(`<article class="comic-detail" style="margin:0;padding:0;background:#0b0b0b;color:#e5e7eb;line-height:1.6;">`)

	sb.WriteString(`<header style="position:sticky;top:0;z-index:2;padding:10px 14px;background:rgba(11,11,11,.92);backdrop-filter:blur(8px);border-bottom:1px solid #232323;">`)
	sb.WriteString(fmt.Sprintf(`<h1 style="margin:0;font-size:16px;font-weight:700;line-height:1.35;">%s</h1>`, htmlEscape(title)))
	if author != "" {
		sb.WriteString(fmt.Sprintf(`<p style="margin:4px 0 0;font-size:12px;color:#9ca3af;">作者：%s</p>`, htmlEscape(author)))
	}
	sb.WriteString(`</header>`)

	sb.WriteString(`<section class="comic-detail-body" style="padding:14px;">`)

	if cover != "" {
		sb.WriteString(`<div style="display:flex;gap:14px;align-items:flex-start;">`)
		sb.WriteString(fmt.Sprintf(
			`<img class="comic-detail-cover" src="%s" alt="%s 封面" loading="lazy" decoding="async" style="width:120px;min-width:120px;height:168px;object-fit:cover;background:#000;border-radius:0!important;clip-path:none!important;"/>`,
			htmlEscape(cover),
			htmlEscape(title),
		))
		sb.WriteString(`<div style="flex:1;min-width:0;">`)
		if latest != "" {
			sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 10px;color:#9ca3af;font-size:12px;">最新：%s</p>`, htmlEscape(latest)))
		}
		sb.WriteString(fmt.Sprintf(`<p style="margin:0;color:#9ca3af;font-size:12px;">来源：<a href="%s" style="color:#93c5fd;text-decoration:underline;">打开原网页</a></p>`, htmlEscape(sourceURL)))
		sb.WriteString(`</div></div>`)
	}

	if desc != "" {
		sb.WriteString(`<h2 style="margin:16px 0 8px;font-size:14px;color:#e5e7eb;">简介</h2>`)
		sb.WriteString(fmt.Sprintf(`<p class="comic-detail-desc" style="margin:0;color:#d1d5db;white-space:pre-wrap;">%s</p>`, htmlEscape(desc)))
	} else {
		sb.WriteString(`<p style="margin:16px 0 0;color:#9ca3af;font-size:12px;">暂无简介</p>`)
	}

	sb.WriteString(`</section></article>`)
	return sb.String()
}

func httpGet(rawURL string, extra ...map[string]string) ([]byte, error) {
	headers := map[string]string{
		"Accept":          "text/html,application/json,*/*",
		"User-Agent":      userAgent,
		"Accept-Language": "zh-TW,zh;q=0.9",
	}
	if len(extra) > 0 {
		for k, v := range extra[0] {
			headers[k] = v
		}
	}
	body, status, err := host.HTTPGet(rawURL, headers)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d for %s", status, rawURL)
	}
	return body, nil
}

func pageNum(params map[string]string) int {
	page, err := strconv.Atoi(strings.TrimSpace(params["page"]))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func channelTitle(channelID string) string {
	if label := strings.TrimSpace(channelLabels[channelID]); label != "" {
		return label
	}
	return "列表"
}

func copyParams(params map[string]string) map[string]string {
	next := make(map[string]string, len(params))
	for k, v := range params {
		next[k] = v
	}
	return next
}

func firstMatch(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(m[1]))
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
	if len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}
