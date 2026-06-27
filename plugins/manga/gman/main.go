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

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	siteName  = "G社漫画"
	baseURL   = "https://m.g-mh.org"
	apiBase   = "https://v2.apikk.top"
	pageLimit = 36
	userAgent = "Mozilla/5.0 (compatible; OrbitPlugins/1.0)"
)

var channelPaths = map[string]string{
	"gman_all":        "/manga",
	"gman_kr":         "/manga-genre/kr",
	"gman_hots":       "/manga-genre/hots",
	"gman_cn":         "/manga-genre/cn",
	"gman_qita":       "/manga-genre/qita",
	"gman_jp":         "/manga-genre/jp",
	"gman_oumei":      "/manga-genre/ou-mei",
	"gman_fuchou":     "/manga-tag/fuchou",
	"gman_gufeng":     "/manga-tag/gufeng",
	"gman_qihuan":     "/manga-tag/qihuan",
	"gman_nixi":       "/manga-tag/nixi",
	"gman_yineng":     "/manga-tag/yineng",
	"gman_zhaixiang":  "/manga-tag/zhaixiang",
	"gman_chuanyue":   "/manga-tag/chuanyue",
	"gman_rexue":      "/manga-tag/rexue",
	"gman_chunai":     "/manga-tag/chunai",
	"gman_xitong":     "/manga-tag/xitong",
	"gman_zhongsheng": "/manga-tag/zhongsheng",
	"gman_maoxian":    "/manga-tag/maoxian",
	"gman_lingyi":     "/manga-tag/lingyi",
	"gman_danvzhu":    "/manga-tag/danvzhu",
	"gman_juqing":     "/manga-tag/juqing",
	"gman_lianai":     "/manga-tag/lianai",
	"gman_xuanhuan":   "/manga-tag/xuanhuan",
	"gman_nvshen":     "/manga-tag/nvshen",
	"gman_kehuan":     "/manga-tag/kehuan",
	"gman_mohuan":     "/manga-tag/mohuan",
	"gman_tuili":      "/manga-tag/tuili",
	"gman_lieqi":      "/manga-tag/lieqi",
	"gman_zhiyu":      "/manga-tag/zhiyu",
	"gman_doushi":     "/manga-tag/doushi",
	"gman_yixing":     "/manga-tag/yixing",
	"gman_qingchun":   "/manga-tag/qingchun",
	"gman_mori":       "/manga-tag/mori",
	"gman_xuanyi":     "/manga-tag/xuanyi",
	"gman_xiuxian":    "/manga-tag/xiuxian",
	"gman_zhandou":    "/manga-tag/zhandou",
}

var channelLabels = map[string]string{
	"gman_all":        "全部",
	"gman_kr":         "韩漫",
	"gman_hots":       "热门漫画",
	"gman_cn":         "国漫",
	"gman_qita":       "其他",
	"gman_jp":         "日漫",
	"gman_oumei":      "欧美",
	"gman_fuchou":     "复仇",
	"gman_gufeng":     "古风",
	"gman_qihuan":     "奇幻",
	"gman_nixi":       "逆袭",
	"gman_yineng":     "异能",
	"gman_zhaixiang":  "宅向",
	"gman_chuanyue":   "穿越",
	"gman_rexue":      "热血",
	"gman_chunai":     "纯爱",
	"gman_xitong":     "系统",
	"gman_zhongsheng": "重生",
	"gman_maoxian":    "冒险",
	"gman_lingyi":     "灵异",
	"gman_danvzhu":    "大女主",
	"gman_juqing":     "剧情",
	"gman_lianai":     "恋爱",
	"gman_xuanhuan":   "玄幻",
	"gman_nvshen":     "女神",
	"gman_kehuan":     "科幻",
	"gman_mohuan":     "魔幻",
	"gman_tuili":      "推理",
	"gman_lieqi":      "猎奇",
	"gman_zhiyu":      "治愈",
	"gman_doushi":     "都市",
	"gman_yixing":     "异形",
	"gman_qingchun":   "青春",
	"gman_mori":       "末日",
	"gman_xuanyi":     "悬疑",
	"gman_xiuxian":    "修仙",
	"gman_zhandou":    "战斗",
	"gman_search":     "搜索",
}

var (
	reComicCard = regexp.MustCompile(`(?is)<a\s+href="/manga/([^"]+)"[^>]*>.*?<img[^>]+class="card"[^>]+src="([^"]+)"[^>]*>.*?<h3[^>]*class="cardtitle"[^>]*>([^<]*)</h3>`)

	reComicTitle          = regexp.MustCompile(`(?is)<meta[^>]*property="og:title"[^>]*content="([^"]*)"`)
	reComicAuthor         = regexp.MustCompile(`(?is)<span[^>]*>\s*作者：\s*</span>.*?<span>\s*([^<]+?)\s*</span>`)
	reComicCover          = regexp.MustCompile(`(?is)<meta[^>]*property="og:image"[^>]*content="([^"]*)"`)
	reComicDesc           = regexp.MustCompile(`(?is)<meta[^>]*property="og:description"[^>]*content="([^"]*)"`)
	reDetailTitle         = regexp.MustCompile(`(?is)<h1[^>]*class="[^"]*text-xl[^"]*"[^>]*>\s*([^<]+?)\s*(?:<span|</h1>)`)
	reLatestChap          = regexp.MustCompile(`(?is)最新章節：\s*<span[^>]*id="lastchap"[^>]*>\s*([^<]+?)\s*</span>`)
	reChapterDrawerConfig = regexp.MustCompile(`(?is)<div[^>]*id="chapterDrawerConfig"([^>]*)>`)
	reDrawerMID           = regexp.MustCompile(`data-mid="(\d+)"`)
	reDrawerAPIHost       = regexp.MustCompile(`data-api-host="([^"]+)"`)
	reMangaMID            = regexp.MustCompile(`data-mid="(\d+)"`)
	reChaptersDelisted    = regexp.MustCompile(`章節下架`)
	reGenre               = regexp.MustCompile(`(?is)<a\s+href="/manga-genre/[^"]+"[^>]*>.*?<span>\s*([^<,]+?)\s*,?\s*</span>`)
	reTag                 = regexp.MustCompile(`(?is)<a\s+href="/manga-tag/[^"]+"[^>]*>.*?<span[^>]*>\s*#?\s*([^<]+?)\s*</span>`)
)

type drawerConfig struct {
	MID     string
	APIHost string
}

type mangaAPIResponse struct {
	Status bool `json:"status"`
	Code   int  `json:"code"`
	Data   struct {
		ID       string       `json:"id"`
		Title    string       `json:"title"`
		Slug     string       `json:"slug"`
		Cover    string       `json:"cover"`
		Desc     string       `json:"desc"`
		Chapters []apiChapter `json:"chapters"`
	} `json:"data"`
}

type apiChapter struct {
	ID         string `json:"id"`
	Attributes struct {
		Title     string  `json:"title"`
		Slug      string  `json:"slug"`
		Order     float64 `json:"order"`
		UpdatedAt string  `json:"updatedAt"`
	} `json:"attributes"`
}

type chapterGetResponse struct {
	Code int `json:"code"`
	Data struct {
		Info struct {
			MangaTitle string `json:"mangatitle"`
			Slug       string `json:"slug"`
			Title      string `json:"title"`
			Images     struct {
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
	sdk.Run(&GManPlugin{})
}

type GManPlugin struct{}

func (p *GManPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/gman/list" || strings.HasPrefix(req.Route, "/gman/list"):
		return fetchList(req.ChannelID, req.Params)
	case req.Route == "/gman/search" || strings.HasPrefix(req.Route, "/gman/search"):
		query := strings.TrimSpace(req.Params["query"])
		if query == "" {
			return nil, fmt.Errorf("missing query parameter")
		}
		return fetchSearch(query, req.Params)
	case req.Route == "/gman/chapters/:id" || strings.HasPrefix(req.Route, "/gman/chapters"):
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchChapters(id)
	case req.Route == "/gman/chapter/:chapterId" || strings.HasPrefix(req.Route, "/gman/chapter"):
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
	listURL := baseURL + listPath
	if page > 1 {
		listURL += "/page/" + strconv.Itoa(page)
	}

	body, err := httpGet(listURL)
	if err != nil {
		return nil, err
	}
	htmlText := string(body)

	items := parseComicCardsHTML(htmlText)
	if len(items) == 0 {
		return nil, fmt.Errorf("empty comic list")
	}

	result := &sdk.FeedResult{
		Title:       siteName + " · " + channelTitle(channelID),
		Description: fmt.Sprintf("第 %d 页 · %d 部作品", page, len(items)),
		Items:       items,
	}
	if strings.Contains(htmlText, pagePath(listPath, page+1)) {
		result.HasMore = true
		result.Next = copyParams(params)
		result.Next["page"] = strconv.Itoa(page + 1)
	}
	return result, nil
}

func fetchSearch(query string, params map[string]string) (*sdk.FeedResult, error) {
	page := pageNum(params)
	searchURL := baseURL + "/s/" + url.PathEscape(query)
	if page > 1 {
		searchURL += "?page=" + strconv.Itoa(page)
	}

	body, err := httpGet(searchURL)
	if err != nil {
		return nil, err
	}
	htmlText := string(body)

	items := parseComicCardsHTML(htmlText)
	if len(items) == 0 {
		return nil, fmt.Errorf("no results for: %s", query)
	}

	result := &sdk.FeedResult{
		Title:       siteName + " · 搜索",
		Description: fmt.Sprintf("关键词「%s」· 第 %d 页 · %d 部作品", query, page, len(items)),
		Items:       items,
	}
	if strings.Contains(htmlText, "?page="+strconv.Itoa(page+1)) {
		result.HasMore = true
		result.Next = copyParams(params)
		result.Next["query"] = query
		result.Next["page"] = strconv.Itoa(page + 1)
	}
	return result, nil
}

func fetchChapters(comicSlug string) (*sdk.FeedResult, error) {
	pageURL := mangaURL(comicSlug)
	body, err := httpGet(pageURL)
	if err != nil {
		return nil, err
	}
	htmlText := string(body)

	title := firstNonEmpty(cleanOGTitle(htmlText), firstMatch(reDetailTitle, htmlText), comicSlug)
	desc := strings.TrimSpace(firstMatch(reComicDesc, htmlText))
	drawer := parseDrawerConfig(htmlText)
	if drawer.MID == "" {
		return nil, fmt.Errorf("manga id not found: %s", comicSlug)
	}

	resp, err := fetchMangaChapters(drawer, pageURL)
	if err != nil {
		return nil, err
	}
	if len(resp.Data.Chapters) == 0 {
		if reChaptersDelisted.MatchString(htmlText) {
			return nil, fmt.Errorf("chapters delisted for: %s", comicSlug)
		}
		return nil, fmt.Errorf("no chapters for: %s", comicSlug)
	}

	items := make([]sdk.FeedItem, 0, len(resp.Data.Chapters)+1)
	items = append(items, sdk.FeedItem{
		ID:    "intro",
		Title: "简介 / 详情",
		URL:   pageURL,
		Tags:  []string{"介绍"},
	})

	for _, ch := range resp.Data.Chapters {
		chapterID := strings.TrimSpace(ch.Attributes.Slug)
		chapterTitle := strings.TrimSpace(ch.Attributes.Title)
		updatedAt := strings.TrimSpace(ch.Attributes.UpdatedAt)
		if chapterTitle == "" {
			chapterTitle = ch.ID
		}

		item := sdk.FeedItem{
			ID:    chapterID,
			Title: chapterTitle,
			URL:   chapterURL(comicSlug, chapterID),
		}
		if updatedAt != "" {
			item.Summary = updatedAt
		}
		items = append(items, item)
	}
	if len(items) == 1 {
		return nil, fmt.Errorf("no chapters for: %s", comicSlug)
	}

	descParts := []string{fmt.Sprintf("共 %d 话", len(items)-1)}
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

	parentURL := mangaURL(parentID)
	body, err := httpGet(parentURL)
	if err != nil {
		return nil, err
	}
	parentHTML := string(body)
	drawer := parseDrawerConfig(parentHTML)
	if drawer.MID == "" {
		return nil, fmt.Errorf("manga id not found: %s", parentID)
	}

	mangaResp, err := fetchMangaChapters(drawer, parentURL)
	if err != nil {
		return nil, err
	}
	if len(mangaResp.Data.Chapters) == 0 {
		if reChaptersDelisted.MatchString(parentHTML) {
			return nil, fmt.Errorf("chapters delisted for: %s", parentID)
		}
		return nil, fmt.Errorf("manga api failed for: %s", parentID)
	}

	chapterInternalID := ""
	chapterTitle := ""
	for _, ch := range mangaResp.Data.Chapters {
		if strings.TrimSpace(ch.Attributes.Slug) != chapterID {
			continue
		}
		chapterInternalID = strings.TrimSpace(ch.ID)
		chapterTitle = strings.TrimSpace(ch.Attributes.Title)
		break
	}
	if chapterInternalID == "" {
		return nil, fmt.Errorf("chapter not found: %s", chapterID)
	}

	chapterAPIURL := chapterInfoAPIURL(drawer.APIHost, drawer.MID, chapterInternalID)
	chapterAPIBody, err := httpGetWithHeaders(chapterAPIURL, refererHeaders(parentURL))
	if err != nil {
		return nil, err
	}
	var resp chapterGetResponse
	if err := json.Unmarshal(chapterAPIBody, &resp); err != nil {
		return nil, fmt.Errorf("parse chapter api: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("chapter api failed: %s", chapterID)
	}

	pages, err := decodeChapterImages(resp.Data.Info.Images.Images)
	if err != nil {
		return nil, fmt.Errorf("decode chapter images: %w", err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no images in chapter: %s", chapterID)
	}

	sort.Slice(pages, func(i, j int) bool { return pages[i].Order < pages[j].Order })
	images := make([]string, 0, len(pages))
	for _, p := range pages {
		imgURL := strings.TrimSpace(p.URL)
		if imgURL == "" {
			continue
		}
		if !strings.HasPrefix(imgURL, "http://") && !strings.HasPrefix(imgURL, "https://") {
			imgURL = chapterCDN(resp.Data.Info.Images.Line) + imgURL
		}
		images = append(images, imgURL)
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("no images in chapter: %s", chapterID)
	}

	mangaTitle := firstNonEmpty(mangaResp.Data.Title, cleanOGTitle(parentHTML), parentID)
	if chapterTitle == "" {
		chapterTitle = strings.TrimSpace(resp.Data.Info.Title)
	}
	if chapterTitle == "" {
		chapterTitle = chapterID
	}
	fullTitle := mangaTitle + " · " + chapterTitle

	cover := firstNonEmpty(mangaResp.Data.Cover, firstMatch(reComicCover, parentHTML))
	contentJSON, err := json.Marshal(images)
	if err != nil {
		return nil, fmt.Errorf("marshal chapter images: %w", err)
	}

	item := sdk.FeedItem{
		ID:      chapterID,
		Title:   fullTitle,
		URL:     chapterURL(parentID, chapterID),
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
	author := firstMatch(reComicAuthor, htmlText)
	cover := firstMatch(reComicCover, htmlText)
	desc := strings.TrimSpace(firstMatch(reComicDesc, htmlText))
	latest := firstMatch(reLatestChap, htmlText)
	tags := parseTagValues(htmlText)

	content := buildComicIntroHTML(title, author, cover, tags, latest, desc, pageURL)
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
			ID:    id,
			Title: title,
			URL:   mangaURL(id),
			Cover: cover,
			Image: cover,
		})
	}
	return items
}

func parseTagValues(htmlText string) []string {
	seen := make(map[string]struct{})
	tags := make([]string, 0)
	appendValues := func(re *regexp.Regexp) {
		for _, m := range re.FindAllStringSubmatch(htmlText, -1) {
			if len(m) < 2 {
				continue
			}
			value := strings.TrimSpace(html.UnescapeString(m[1]))
			value = strings.Trim(value, ",# ")
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			tags = append(tags, value)
		}
	}
	appendValues(reGenre)
	appendValues(reTag)
	return tags
}

func pagePath(listPath string, page int) string {
	if page <= 1 {
		return listPath
	}
	return listPath + "/page/" + strconv.Itoa(page)
}

func mangaURL(slug string) string {
	return baseURL + "/manga/" + url.PathEscape(slug)
}

func chapterURL(slug, chapterID string) string {
	return mangaURL(slug) + "/" + url.PathEscape(chapterID)
}

func parseDrawerConfig(htmlText string) drawerConfig {
	cfg := drawerConfig{APIHost: apiBase}
	if m := reChapterDrawerConfig.FindStringSubmatch(htmlText); len(m) >= 2 {
		attrs := m[1]
		cfg.MID = firstMatch(reDrawerMID, attrs)
		if host := strings.TrimSpace(firstMatch(reDrawerAPIHost, attrs)); host != "" {
			cfg.APIHost = host
		}
	}
	if cfg.MID == "" {
		cfg.MID = firstMatch(reMangaMID, htmlText)
	}
	return cfg
}

func mangaListAPIURL(apiHost, mid string) string {
	return strings.TrimSuffix(apiHost, "/") + "/api/v2/manga/get?mid=" + url.QueryEscape(mid) + "&mode=all"
}

func mangaListAPIURLV1(apiHost, mid string) string {
	return strings.TrimSuffix(apiHost, "/") + "/api/manga/get?mid=" + url.QueryEscape(mid) + "&mode=all"
}

func fetchMangaChapters(drawer drawerConfig, referer string) (mangaAPIResponse, error) {
	var lastErr error
	for _, apiURL := range []string{
		mangaListAPIURL(drawer.APIHost, drawer.MID),
		mangaListAPIURLV1(drawer.APIHost, drawer.MID),
	} {
		apiBody, err := httpGetWithHeaders(apiURL, refererHeaders(referer))
		if err != nil {
			lastErr = err
			continue
		}
		var resp mangaAPIResponse
		if err := json.Unmarshal(apiBody, &resp); err != nil {
			lastErr = fmt.Errorf("parse manga api: %w", err)
			continue
		}
		if !mangaAPIOK(resp) {
			lastErr = fmt.Errorf("manga api failed")
			continue
		}
		if len(resp.Data.Chapters) > 0 {
			return resp, nil
		}
	}
	if lastErr != nil {
		return mangaAPIResponse{}, lastErr
	}
	return mangaAPIResponse{}, nil
}

func chapterInfoAPIURL(apiHost, mid, chapterInternalID string) string {
	return strings.TrimSuffix(apiHost, "/") + "/api/v2/chapter/getinfo?m=" + url.QueryEscape(mid) + "&c=" + url.QueryEscape(chapterInternalID)
}

func mangaAPIOK(resp mangaAPIResponse) bool {
	return resp.Status || resp.Code == 200
}

func chapterCDN(line int) string {
	if line == 2 {
		return "https://f40-1-4.g-mh.online"
	}
	return "https://t40-1-4.g-mh.online"
}

func cleanOGTitle(htmlText string) string {
	title := firstMatch(reComicTitle, htmlText)
	title = strings.TrimSuffix(title, "-G社漫畫")
	title = strings.TrimSpace(strings.TrimSuffix(title, " - G社漫畫"))
	return strings.TrimSpace(title)
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
		if len(tags) > 0 {
			sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 10px;color:#9ca3af;font-size:12px;">标签：%s</p>`, htmlEscape(strings.Join(tags, " / "))))
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

func httpGet(rawURL string) ([]byte, error) {
	return httpGetWithHeaders(rawURL, nil)
}

func httpGetWithHeaders(rawURL string, extra map[string]string) ([]byte, error) {
	headers := map[string]string{
		"Accept":          "text/html,application/json,*/*",
		"User-Agent":      userAgent,
		"Accept-Language": "zh-TW,zh;q=0.9",
		"Referer":         baseURL + "/",
	}
	for k, v := range extra {
		headers[k] = v
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

func refererHeaders(referer string) map[string]string {
	return map[string]string{
		"Accept":          "application/json,text/html,*/*",
		"User-Agent":      userAgent,
		"Accept-Language": "zh-TW,zh;q=0.9",
		"Referer":         referer,
	}
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
