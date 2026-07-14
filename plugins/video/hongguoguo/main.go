package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const defaultBaseURL = "https://www.hongguoguo.tv"

var (
	rePlayPath   = regexp.MustCompile(`/vod/play/id/(\d+)/sid/(\d+)/nid/(\d+)`)
	rePlayerAAAA = regexp.MustCompile(`(?s)var\s+player_aaaa\s*=\s*(\{.*?})\s*</script>`)
	rePageLink   = regexp.MustCompile(`/page/(\d+)\.html`)
	reStripIcon  = regexp.MustCompile(`^\s*[\x{e600}-\x{e7ff}\x{f000}-\x{f8ff}]`)
)

var typeLabels = map[string]string{
	"jingxuanduanju": "精选短剧",
	"chuanyueduanju": "穿越短剧",
	"guzhuangduanju": "古装短剧",
	"nixiduanju":     "逆袭短剧",
	"dushiduanju":    "都市短剧",
	"fuliduanju":     "福利短剧",
}

func main() {
	sdk.Run(&HongGuoGuoPlugin{})
}

type HongGuoGuoPlugin struct{}

type playerAAAA struct {
	Flag     string `json:"flag"`
	Encrypt  int    `json:"encrypt"`
	URL      string `json:"url"`
	URLNext  string `json:"url_next"`
	From     string `json:"from"`
	Link     string `json:"link"`
	LinkNext string `json:"link_next"`
	LinkPre  string `json:"link_pre"`
	ID       string `json:"id"`
	SID      int    `json:"sid"`
	NID      int    `json:"nid"`
	VodData  struct {
		VodName     string `json:"vod_name"`
		VodActor    string `json:"vod_actor"`
		VodDirector string `json:"vod_director"`
		VodClass    string `json:"vod_class"`
	} `json:"vod_data"`
}

type listItem struct {
	ID      string
	Title   string
	URL     string
	Cover   string
	Score   string
	Remarks string
	Year    string
	Summary string
}

type episodeRef struct {
	NID   string
	SID   string
	Title string
	URL   string
}

func (p *HongGuoGuoPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(req.Var("baseURL")), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	switch {
	case req.Route == "/hongguoguo/list/:type" || strings.HasPrefix(req.Route, "/hongguoguo/list"):
		typeID := strings.TrimSpace(req.Params["type"])
		if typeID == "" {
			typeID = "jingxuanduanju"
		}
		return fetchList(baseURL, typeID, req.Params)
	case req.Route == "/hongguoguo/search/:query" || strings.HasPrefix(req.Route, "/hongguoguo/search"):
		query := strings.TrimSpace(req.Params["query"])
		if query == "" {
			return nil, fmt.Errorf("missing query parameter")
		}
		return fetchSearch(baseURL, query, req.Params)
	case req.Route == "/hongguoguo/chapters/:id" || strings.HasPrefix(req.Route, "/hongguoguo/chapters"):
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchChapters(baseURL, id)
	case req.Route == "/hongguoguo/episode/:chapterId" || strings.HasPrefix(req.Route, "/hongguoguo/episode"):
		parentID := strings.TrimSpace(req.Params["id"])
		chapterID := strings.TrimSpace(req.Params["chapterId"])
		if parentID == "" || chapterID == "" {
			return nil, fmt.Errorf("missing id or chapterId parameter")
		}
		sid := strings.TrimSpace(req.Params["sid"])
		if sid == "" {
			sid = "1"
		}
		return fetchEpisode(baseURL, parentID, sid, chapterID)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(baseURL, typeID string, params map[string]string) (*sdk.FeedResult, error) {
	if _, ok := typeLabels[typeID]; !ok {
		return nil, fmt.Errorf("unsupported type: %s", typeID)
	}
	page := pageNum(params)
	listURL := fmt.Sprintf("%s/vod/show/id/%s/page/%d.html", baseURL, typeID, page)

	doc, err := fetchDoc(listURL)
	if err != nil {
		return nil, err
	}

	items := parseVodCards(doc, baseURL)
	if len(items) == 0 {
		return nil, fmt.Errorf("empty list for type=%s page=%d", typeID, page)
	}

	label := typeLabels[typeID]
	result := &sdk.FeedResult{
		Title:       "红果果 · " + label,
		Description: fmt.Sprintf("%s · 第 %d 页", label, page),
		Items:       listItemsToFeed(items),
	}
	if hasNextPage(doc, page) || len(items) >= 24 {
		result.HasMore = true
		result.Next = map[string]string{
			"type": typeID,
			"page": strconv.Itoa(page + 1),
		}
	}
	return result, nil
}

func fetchSearch(baseURL, keyword string, params map[string]string) (*sdk.FeedResult, error) {
	page := pageNum(params)
	encoded := url.PathEscape(keyword)
	var listURL string
	if page <= 1 {
		listURL = fmt.Sprintf("%s/vod/search/wd/%s.html", baseURL, encoded)
	} else {
		listURL = fmt.Sprintf("%s/vod/search/wd/%s/page/%d.html", baseURL, encoded, page)
	}

	doc, err := fetchDoc(listURL)
	if err != nil {
		return nil, err
	}

	items := parseVodCards(doc, baseURL)
	if len(items) == 0 {
		return nil, fmt.Errorf("no results for: %s", keyword)
	}

	result := &sdk.FeedResult{
		Title:       "红果果 · 搜索",
		Description: fmt.Sprintf("关键词「%s」· 第 %d 页", keyword, page),
		Items:       listItemsToFeed(items),
	}
	if hasNextPage(doc, page) || len(items) >= 24 {
		result.HasMore = true
		result.Next = map[string]string{
			"query": keyword,
			"page":  strconv.Itoa(page + 1),
		}
	}
	return result, nil
}

func fetchChapters(baseURL, vodID string) (*sdk.FeedResult, error) {
	playURL := fmt.Sprintf("%s/vod/play/id/%s/sid/1/nid/1.html", baseURL, vodID)
	doc, body, err := fetchDocBody(playURL)
	if err != nil {
		return nil, err
	}

	meta := parsePlayMeta(doc, body, vodID)
	episodes := parseEpisodeList(doc, vodID, baseURL)
	if len(episodes) == 0 {
		episodes = []episodeRef{{
			NID:   "1",
			SID:   "1",
			Title: "全集",
			URL:   playURL,
		}}
	}

	publishedAt := time.Now().UTC().Format(time.RFC3339)
	items := make([]sdk.FeedItem, 0, len(episodes))
	for _, ep := range episodes {
		items = append(items, sdk.FeedItem{
			ID:          ep.NID,
			Title:       ep.Title,
			URL:         ep.URL,
			Cover:       meta.Cover,
			Image:       meta.Cover,
			PublishedAt: publishedAt,
			Tags:        nonEmptyTags(meta.Remarks, meta.Year, meta.Score),
		})
	}

	title := meta.Title
	if title == "" {
		title = "短剧 " + vodID
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: joinMeta(meta.Remarks, meta.Year, fmt.Sprintf("共 %d 集", len(episodes))),
		Items:       items,
	}, nil
}

func fetchEpisode(baseURL, vodID, sid, nid string) (*sdk.FeedResult, error) {
	playURL := fmt.Sprintf("%s/vod/play/id/%s/sid/%s/nid/%s.html", baseURL, vodID, sid, nid)
	doc, body, err := fetchDocBody(playURL)
	if err != nil {
		return nil, err
	}

	player, err := parsePlayerAAAA(body)
	if err != nil {
		return nil, err
	}

	streamURL, err := decodePlayURL(player.URL, player.Encrypt)
	if err != nil {
		return nil, fmt.Errorf("decode play url: %w", err)
	}
	if streamURL == "" {
		return nil, fmt.Errorf("empty play url for id=%s nid=%s", vodID, nid)
	}

	meta := parsePlayMeta(doc, body, vodID)
	epTitle := currentEpisodeTitle(doc, nid)
	showTitle := firstNonEmpty(player.VodData.VodName, meta.Title, "短剧 "+vodID)
	title := showTitle
	if epTitle != "" && epTitle != showTitle {
		title = showTitle + " · " + epTitle
	}

	item := sdk.FeedItem{
		ID:          nid,
		Title:       title,
		URL:         streamURL,
		Summary:     joinMeta(meta.Remarks, meta.Year, player.From),
		Content:     buildEpisodeHTML(showTitle, epTitle, streamURL, meta, player),
		Cover:       meta.Cover,
		Image:       meta.Cover,
		PublishedAt: time.Now().UTC().Format(time.RFC3339),
		Tags:        nonEmptyTags(meta.Remarks, meta.Year, player.From, meta.Score),
		Author:      firstNonEmpty(player.VodData.VodActor, meta.Actor),
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "红果果 · 播放",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func fetchDoc(rawURL string) (*goquery.Document, error) {
	doc, _, err := fetchDocBody(rawURL)
	return doc, err
}

func fetchDocBody(rawURL string) (*goquery.Document, string, error) {
	body, status, err := host.HTTPGet(rawURL, defaultHeaders(rawURL))
	if err != nil {
		return nil, "", fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, "", fmt.Errorf("http status %d for %s", status, rawURL)
	}
	htmlBody := string(body)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil, "", fmt.Errorf("parse html: %w", err)
	}
	return doc, htmlBody, nil
}

func defaultHeaders(pageURL string) map[string]string {
	referer := defaultBaseURL + "/"
	if u, err := url.Parse(pageURL); err == nil && u.Scheme != "" && u.Host != "" {
		referer = u.Scheme + "://" + u.Host + "/"
	}
	return map[string]string{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Referer":    referer,
	}
}

func parseVodCards(doc *goquery.Document, baseURL string) []listItem {
	seen := make(map[string]struct{})
	var items []listItem

	doc.Find("li.l-list-box").Each(func(_ int, li *goquery.Selection) {
		a := li.Find("a.tim-link").First()
		href, _ := a.Attr("href")
		m := rePlayPath.FindStringSubmatch(href)
		if len(m) < 4 {
			return
		}
		id := m[1]
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}

		title := strings.TrimSpace(a.Find("h2.tim-title").First().Text())
		if title == "" {
			title = cleanEpisodeTitle(a.Find(".Title, .title-h").First().Text())
		}
		if title == "" {
			return
		}

		cover, _ := a.Find(".lazy").Attr("data-original")
		if cover == "" {
			cover, _ = a.Find("img").Attr("src")
		}
		score := strings.TrimSpace(a.Find(".tim-tag .f").First().Text())
		remarks := strings.TrimSpace(a.Find(".tim-tag .b").First().Text())

		suspension := li.Find(".suspension")
		year := strings.TrimSpace(suspension.Find(".Info").First().Text())
		summary := strings.TrimSpace(suspension.Find(".Blurb").First().Text())

		items = append(items, listItem{
			ID:      id,
			Title:   title,
			URL:     absURL(baseURL, href),
			Cover:   strings.TrimSpace(cover),
			Score:   score,
			Remarks: remarks,
			Year:    year,
			Summary: summary,
		})
	})

	return items
}

func listItemsToFeed(items []listItem) []sdk.FeedItem {
	out := make([]sdk.FeedItem, 0, len(items))
	for _, item := range items {
		summary := joinMeta(item.Remarks, item.Year, item.Score)
		if item.Summary != "" && item.Summary != "暂无简介" {
			if summary != "" {
				summary += " · "
			}
			summary += truncate(item.Summary, 80)
		}
		out = append(out, sdk.FeedItem{
			ID:          item.ID,
			Title:       item.Title,
			URL:         item.URL,
			Cover:       item.Cover,
			Image:       item.Cover,
			Summary:     summary,
			PublishedAt: time.Now().UTC().Format(time.RFC3339),
			Tags:        nonEmptyTags(item.Remarks, item.Year, item.Score),
		})
	}
	return out
}

type playMeta struct {
	Title   string
	Cover   string
	Remarks string
	Year    string
	Score   string
	Actor   string
	Blurb   string
}

func parsePlayMeta(doc *goquery.Document, body, vodID string) playMeta {
	meta := playMeta{}

	if title := strings.TrimSpace(doc.Find("h1.tim-title").First().Text()); title != "" {
		meta.Title = strings.Split(title, "-")[0]
		meta.Title = strings.TrimSpace(meta.Title)
	}
	if img, ok := doc.Find(`meta[property="og:image"]`).Attr("content"); ok {
		meta.Cover = strings.TrimSpace(img)
	}
	meta.Remarks = strings.TrimSpace(doc.Find(".play-detail .time").First().Text())

	doc.Find(".play-detail .top10 p").Each(func(_ int, p *goquery.Selection) {
		text := strings.TrimSpace(p.Text())
		switch {
		case strings.HasPrefix(text, "评分："):
			meta.Score = strings.TrimPrefix(text, "评分：")
		case strings.HasPrefix(text, "主演："):
			meta.Actor = strings.TrimPrefix(text, "主演：")
		case strings.Contains(text, "年份："):
			if i := strings.Index(text, "年份："); i >= 0 {
				meta.Year = strings.TrimSpace(stripTags(text[i+len("年份："):]))
			}
		case strings.HasPrefix(text, "简介："):
			meta.Blurb = strings.TrimSpace(strings.TrimPrefix(text, "简介："))
		}
	})

	if meta.Title == "" {
		if m := rePlayerAAAA.FindStringSubmatch(body); len(m) == 2 {
			if player, err := parsePlayerAAAA(body); err == nil {
				meta.Title = player.VodData.VodName
			}
		}
	}
	if meta.Title == "" {
		meta.Title = "短剧 " + vodID
	}
	return meta
}

func parseEpisodeList(doc *goquery.Document, vodID, baseURL string) []episodeRef {
	seen := make(map[string]struct{})
	var eps []episodeRef

	doc.Find(".anthology-list a.tim-link").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := rePlayPath.FindStringSubmatch(href)
		if len(m) < 4 || m[1] != vodID {
			return
		}
		nid := m[3]
		if _, ok := seen[nid]; ok {
			return
		}
		seen[nid] = struct{}{}
		title := cleanEpisodeTitle(a.Text())
		if title == "" {
			title = "第" + nid + "集"
		}
		eps = append(eps, episodeRef{
			NID:   nid,
			SID:   m[2],
			Title: title,
			URL:   absURL(baseURL, href),
		})
	})
	return eps
}

func currentEpisodeTitle(doc *goquery.Document, nid string) string {
	var title string
	doc.Find(".anthology-list a.tim-link").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		href, _ := a.Attr("href")
		m := rePlayPath.FindStringSubmatch(href)
		if len(m) >= 4 && m[3] == nid {
			title = cleanEpisodeTitle(a.Text())
			return false
		}
		return true
	})
	if title == "" {
		h1 := strings.TrimSpace(doc.Find("h1.tim-title").First().Text())
		if parts := strings.SplitN(h1, "-", 2); len(parts) == 2 {
			title = strings.TrimSpace(parts[1])
		}
	}
	if title == "" {
		title = "第" + nid + "集"
	}
	return title
}

func parsePlayerAAAA(body string) (*playerAAAA, error) {
	m := rePlayerAAAA.FindStringSubmatch(body)
	if len(m) < 2 {
		return nil, fmt.Errorf("player_aaaa not found")
	}
	var player playerAAAA
	if err := json.Unmarshal([]byte(m[1]), &player); err != nil {
		return nil, fmt.Errorf("parse player_aaaa: %w", err)
	}
	return &player, nil
}

func decodePlayURL(raw string, encrypt int) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	switch encrypt {
	case 0:
		return raw, nil
	case 1:
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	case 2:
		unescaped, err := url.QueryUnescape(raw)
		if err != nil {
			unescaped = raw
		}
		decoded, err := base64.StdEncoding.DecodeString(unescaped)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	default:
		return raw, nil
	}
}

func buildEpisodeHTML(showTitle, epTitle, streamURL string, meta playMeta, player *playerAAAA) string {
	var sb strings.Builder
	sb.WriteString(`<article class="hongguoguo-player" style="color:#1f2937;line-height:1.6;">`)
	sb.WriteString(`<video id="hongguoguo-player" controls playsinline preload="metadata" style="width:100%;max-width:100%;border-radius:12px;background:#000;display:block;">`)
	sb.WriteString(`<source src="`)
	sb.WriteString(html.EscapeString(streamURL))
	sb.WriteString(`" type="`)
	sb.WriteString(videoMIME(streamURL))
	sb.WriteString(`"></video>`)

	sb.WriteString(`<section style="margin-top:20px;padding-top:18px;border-top:1px solid #eef2f7;">`)
	sb.WriteString(`<div style="display:flex;gap:16px;align-items:flex-start;">`)
	if meta.Cover != "" {
		sb.WriteString(fmt.Sprintf(
			`<img src="%s" alt="%s" style="width:108px;min-width:108px;height:152px;object-fit:cover;border-radius:10px;background:#f3f4f6;"/>`,
			html.EscapeString(meta.Cover), html.EscapeString(showTitle),
		))
	}
	sb.WriteString(`<div style="flex:1;min-width:0;">`)
	sb.WriteString(fmt.Sprintf(`<h1 style="margin:0 0 6px;font-size:1.25rem;line-height:1.35;">%s</h1>`, html.EscapeString(showTitle)))
	if epTitle != "" {
		sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 10px;color:#6b7280;font-size:14px;">%s</p>`, html.EscapeString(epTitle)))
	}
	for _, tag := range nonEmptyTags(meta.Remarks, meta.Year, meta.Score, player.From) {
		sb.WriteString(fmt.Sprintf(
			`<span style="display:inline-block;margin:0 6px 6px 0;padding:4px 10px;border-radius:999px;background:#f3f4f6;color:#4b5563;font-size:12px;">%s</span>`,
			html.EscapeString(tag),
		))
	}
	sb.WriteString(`</div></div>`)
	if meta.Blurb != "" && meta.Blurb != "暂无简介" {
		sb.WriteString(`<div style="margin-top:16px;">`)
		sb.WriteString(`<h2 style="margin:0 0 8px;font-size:15px;">简介</h2>`)
		sb.WriteString(fmt.Sprintf(`<p style="margin:0;color:#374151;white-space:pre-wrap;">%s</p>`, html.EscapeString(meta.Blurb)))
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</section></article>`)
	return sb.String()
}

func hasNextPage(doc *goquery.Document, current int) bool {
	maxPage := current
	doc.Find("a[href*='/page/']").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := rePageLink.FindStringSubmatch(href)
		if len(m) == 2 {
			if n, err := strconv.Atoi(m[1]); err == nil && n > maxPage {
				maxPage = n
			}
		}
	})
	return maxPage > current
}

func pageNum(params map[string]string) int {
	page := 1
	if s := strings.TrimSpace(params["page"]); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			page = n
		}
	}
	return page
}

func absURL(baseURL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		return baseURL + href
	}
	return baseURL + "/" + href
}

func cleanEpisodeTitle(raw string) string {
	s := strings.TrimSpace(raw)
	s = reStripIcon.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func stripTags(s string) string {
	return strings.TrimSpace(regexp.MustCompile(`(?s)<[^>]+>`).ReplaceAllString(s, ""))
}

func videoMIME(u string) string {
	lower := strings.ToLower(u)
	switch {
	case strings.Contains(lower, ".m3u8"):
		return "application/vnd.apple.mpegurl"
	case strings.Contains(lower, ".mp4"):
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

func joinMeta(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "未知" && p != "0.0分" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

func nonEmptyTags(parts ...string) []string {
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "未知" || p == "0.0分" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" && v != "未知" {
			return v
		}
	}
	return ""
}

func truncate(s string, maxLen int) string {
	rs := []rune(s)
	if len(rs) <= maxLen {
		return s
	}
	return string(rs[:maxLen]) + "..."
}
