package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL   = "https://www.gequbao.com"
	defaultUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
)

type auth struct {
	cookie string
	ua     string
}

var (
	reArtistName   = regexp.MustCompile(`(?s)<h1 class="artist-name">\s*([^<]+)`)
	reArtistDesc   = regexp.MustCompile(`(?s)<p class="artist-desc scrollable">\s*(.*?)\s*</p>`)
	reAvatar       = regexp.MustCompile(`(?s)<img class="artist-avatar[^"]*"\s+src="([^"]+)"`)
	reSongRows     = regexp.MustCompile(`(?s)<div class="row no-gutters py-2d5 border-top align-items-center">(.*?)</div>\s*</div>`)
	reSongURL      = regexp.MustCompile(`href="(/music/\d+)"`)
	reSongTitle    = regexp.MustCompile(`title="([^"]+)"`)
	reSongDuration = regexp.MustCompile(`(?s)<div class="col-md-2 text-center text-muted font-smaller d-none d-md-block">\s*([^<]+)\s*</div>`)
	reAppData      = regexp.MustCompile(`(?s)window\.appData\s*=\s*JSON\.parse\('(.+?)'\);`)
	reTitleTag     = regexp.MustCompile(`(?s)<title>\s*(.*?)\s*</title>`)
	reLrc          = regexp.MustCompile(`(?s)<div class="content-lrc mt-1" id="content-lrc">(.*?)</div>`)
	reIDFromURL    = regexp.MustCompile(`/music/(\d+)`)
	reTags         = regexp.MustCompile(`\s*-\s*`)
)

func main() {
	sdk.Run(&GequbaoPlugin{})
}

type GequbaoPlugin struct{}

type appData struct {
	MP3ID      int    `json:"mp3_id"`
	PlayID     string `json:"play_id"`
	Title      string `json:"mp3_title"`
	Author     string `json:"mp3_author"`
	Cover      string `json:"mp3_cover"`
	Duration   string `json:"mp3_duration"`
	LRCIsEmpty bool   `json:"lrc_is_empty"`
	VIPDownURL bool   `json:"vip_down_url"`
}

type playURLResp struct {
	Code int `json:"code"`
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
	Msg string `json:"msg"`
}

func (p *GequbaoPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	a := authFromReq(req)

	switch {
	case req.Route == "/gequbao/channel" || strings.HasPrefix(req.Route, "/gequbao/channel"):
		return fetchChannel(req, a)
	case req.Route == "/gequbao/detail" || strings.HasPrefix(req.Route, "/gequbao/detail"):
		return fetchDetail(req, a)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func authFromReq(req *sdk.FetchRequest) auth {
	cookie := strings.TrimSpace(req.Var("cookie"))
	ua := strings.TrimSpace(req.Var("userAgent"))
	if ua == "" {
		ua = defaultUA
	}
	return auth{cookie: cookie, ua: ua}
}

func fetchChannel(req *sdk.FetchRequest, a auth) (*sdk.FeedResult, error) {
	pageURL := strings.TrimSpace(req.Params["url"])
	if pageURL == "" {
		return nil, fmt.Errorf("missing url parameter")
	}
	pageURL = absURL(pageURL)

	body, status, err := httpGet(pageURL, a)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPResponse(status, body, "channel page"); err != nil {
		return nil, err
	}
	page := string(body)

	profileName := cleanText(reArtistName.FindStringSubmatch(page), 1)
	profileDesc := cleanHTMLBlock(reArtistDesc.FindStringSubmatch(page), 1)
	avatar := cleanText(reAvatar.FindStringSubmatch(page), 1)

	rows := reSongRows.FindAllStringSubmatch(page, -1)
	if len(rows) == 0 {
		return nil, fmt.Errorf("no songs found on page")
	}

	items := make([]sdk.FeedItem, 0, len(rows))
	seen := make(map[string]bool)
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		item, ok := parseRow(row[1], profileName, avatar)
		if !ok || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no valid songs parsed")
	}

	channelTitle := profileName
	if channelTitle == "" {
		channelTitle = strings.TrimSpace(req.Params["label"])
	}
	if channelTitle == "" {
		channelTitle = "歌曲宝"
	}

	result := &sdk.FeedResult{
		Title:       channelTitle,
		Description: profileDesc,
		Items:       items,
	}

	currentPage := pageNum(req.Params["page"])
	nextPage := currentPage + 1
	if hasNextPage(page, nextPage) {
		result.HasMore = true
		result.Next = cloneParams(req.Params)
		result.Next["page"] = strconv.Itoa(nextPage)
		result.Next["url"] = withPage(pageURL, nextPage)
	}

	return result, nil
}

func fetchDetail(req *sdk.FetchRequest, a auth) (*sdk.FeedResult, error) {
	pageURL := strings.TrimSpace(req.Params["url"])
	if pageURL == "" {
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing url or id parameter")
		}
		pageURL = "/music/" + id
	}
	pageURL = absURL(pageURL)

	body, status, err := httpGet(pageURL, a)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPResponse(status, body, "detail page"); err != nil {
		return nil, err
	}
	page := string(body)

	data, err := parseAppData(page)
	if err != nil {
		return nil, err
	}
	playURL, err := fetchPlayURL(data.PlayID, pageURL, a)
	if err != nil {
		return nil, err
	}

	lyrics := extractLyrics(page)
	title := strings.TrimSpace(data.Title)
	if title == "" {
		title = fallbackTitle(page)
	}
	author := strings.TrimSpace(data.Author)

	item := sdk.FeedItem{
		ID:          songID(pageURL, data.MP3ID),
		Title:       title,
		URL:         playURL,
		Summary:     joinNonEmpty(" · ", author, data.Duration),
		Author:      author,
		Cover:       absURL(data.Cover),
		Image:       absURL(data.Cover),
		Tags:        splitTags(author),
		Content:     buildAudioHTML(title, author, playURL, absURL(data.Cover), lyrics),
		PublishedAt: "",
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "歌曲详情",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func parseRow(rowHTML, fallbackAuthor, fallbackCover string) (sdk.FeedItem, bool) {
	rawURL := cleanText(reSongURL.FindStringSubmatch(rowHTML), 1)
	if rawURL == "" {
		return sdk.FeedItem{}, false
	}
	titleAttr := cleanText(reSongTitle.FindStringSubmatch(rowHTML), 1)
	duration := cleanText(reSongDuration.FindStringSubmatch(rowHTML), 1)

	title, author := splitSongTitle(titleAttr)
	if title == "" {
		return sdk.FeedItem{}, false
	}
	if author == "" {
		author = fallbackAuthor
	}

	fullURL := absURL(rawURL)
	return sdk.FeedItem{
		ID:          songID(fullURL, 0),
		Title:       title,
		URL:         fullURL,
		Summary:     joinNonEmpty(" · ", author, duration),
		Author:      author,
		Cover:       absURL(fallbackCover),
		Image:       absURL(fallbackCover),
		Tags:        splitTags(author),
		PublishedAt: "",
	}, true
}

func parseAppData(page string) (*appData, error) {
	m := reAppData.FindStringSubmatch(page)
	if len(m) < 2 {
		return nil, fmt.Errorf("window.appData not found")
	}

	var decoded string
	raw := `"` + m[1] + `"`
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("decode appData string failed: %w", err)
	}

	var data appData
	if err := json.Unmarshal([]byte(decoded), &data); err != nil {
		return nil, fmt.Errorf("parse appData json failed: %w", err)
	}
	if strings.TrimSpace(data.PlayID) == "" {
		return nil, fmt.Errorf("missing play_id in appData")
	}
	return &data, nil
}

func httpGet(rawURL string, a auth) ([]byte, int, error) {
	headers := map[string]string{
		"Accept":     "text/html,application/json,*/*",
		"Referer":    baseURL + "/",
		"User-Agent": a.ua,
	}
	if a.cookie != "" {
		headers["Cookie"] = a.cookie
	}
	return host.HTTPGet(rawURL, headers)
}

func fetchPlayURL(playID, referer string, a auth) (string, error) {
	postBody := "id=" + url.QueryEscape(playID)
	headers := map[string]string{
		"Accept":           "application/json, text/javascript, */*; q=0.01",
		"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
		"Origin":           baseURL,
		"Referer":          referer,
		"User-Agent":       a.ua,
		"X-Requested-With": "XMLHttpRequest",
	}
	if a.cookie != "" {
		headers["Cookie"] = a.cookie
	}
	body, status, err := host.HTTPPost(baseURL+"/member/common-play-url", headers, postBody)
	if err != nil {
		return "", err
	}
	if err := checkHTTPResponse(status, body, "play-url api"); err != nil {
		return "", err
	}

	var resp playURLResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse play-url response failed: %w", err)
	}
	if resp.Code != 1 || strings.TrimSpace(resp.Data.URL) == "" {
		return "", fmt.Errorf("play-url response invalid: %s", resp.Msg)
	}
	return strings.ReplaceAll(resp.Data.URL, `\/`, `/`), nil
}

func checkHTTPResponse(status int, body []byte, label string) error {
	if status >= 200 && status < 300 && !isCloudflareChallenge(body) {
		return nil
	}
	if isCloudflareChallenge(body) || status == 403 || status == 503 {
		return fmt.Errorf("captcha: %s blocked by Cloudflare", label)
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("%s status %d", label, status)
	}
	return nil
}

func isCloudflareChallenge(body []byte) bool {
	s := strings.ToLower(string(body))
	return strings.Contains(s, "cf-browser-verification") ||
		strings.Contains(s, "challenge-platform") ||
		strings.Contains(s, "just a moment") ||
		strings.Contains(s, "cf-challenge")
}

func extractLyrics(page string) string {
	m := reLrc.FindStringSubmatch(page)
	if len(m) < 2 {
		return ""
	}
	raw := m[1]
	raw = strings.ReplaceAll(raw, "<br />", "\n")
	raw = strings.ReplaceAll(raw, "<br/>", "\n")
	raw = strings.ReplaceAll(raw, "<br>", "\n")
	raw = stripTags(raw)
	return strings.TrimSpace(html.UnescapeString(raw))
}

func fallbackTitle(page string) string {
	m := reTitleTag.FindStringSubmatch(page)
	if len(m) < 2 {
		return "歌曲宝"
	}
	title := html.UnescapeString(strings.TrimSpace(m[1]))
	title = strings.TrimSuffix(title, " - 歌曲宝")
	return strings.TrimSpace(title)
}

func buildAudioHTML(title, author, audioURL, cover, lyrics string) string {
	var sb strings.Builder
	sb.WriteString(`<article style="color:#1f2937;line-height:1.6;">`)
	if cover != "" {
		sb.WriteString(`<div style="display:flex;gap:14px;align-items:flex-start;margin-bottom:14px;">`)
		sb.WriteString(`<img src="` + html.EscapeString(cover) + `" alt="" style="width:96px;height:96px;border-radius:8px;object-fit:cover;flex-shrink:0;">`)
		sb.WriteString(`<div><h2 style="margin:0 0 8px;font-size:18px;">` + html.EscapeString(title) + `</h2>`)
		if author != "" {
			sb.WriteString(`<p style="margin:0;color:#6b7280;font-size:14px;">` + html.EscapeString(author) + `</p>`)
		}
		sb.WriteString(`</div></div>`)
	}
	sb.WriteString(`<audio controls preload="metadata" style="width:100%;display:block;">`)
	sb.WriteString(`<source src="` + html.EscapeString(audioURL) + `" type="audio/mpeg">`)
	sb.WriteString(`</audio>`)
	if strings.TrimSpace(lyrics) != "" {
		sb.WriteString(`<p style="margin-top:14px;white-space:pre-wrap;color:#4b5563;font-size:13px;">` + html.EscapeString(lyrics) + `</p>`)
	}
	sb.WriteString(`</article>`)
	return sb.String()
}

func splitSongTitle(input string) (string, string) {
	input = strings.TrimSpace(html.UnescapeString(input))
	if input == "" {
		return "", ""
	}
	parts := reTags.Split(input, 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return input, ""
}

func songID(rawURL string, fallbackID int) string {
	if fallbackID > 0 {
		return "gequbao:" + strconv.Itoa(fallbackID)
	}
	if m := reIDFromURL.FindStringSubmatch(rawURL); len(m) > 1 {
		return "gequbao:" + m[1]
	}
	rawURL = strings.TrimPrefix(rawURL, baseURL)
	rawURL = strings.Trim(rawURL, "/")
	if rawURL == "" {
		return "gequbao:track"
	}
	return "gequbao:" + strings.ReplaceAll(rawURL, "/", ":")
}

func splitTags(author string) []string {
	author = strings.TrimSpace(author)
	if author == "" {
		return nil
	}
	parts := strings.FieldsFunc(author, func(r rune) bool {
		return r == '/' || r == '、' || r == ',' || r == '，'
	})
	seen := make(map[string]bool)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{author}
	}
	return out
}

func cleanText(groups []string, idx int) string {
	if len(groups) <= idx {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(groups[idx]))
}

func cleanHTMLBlock(groups []string, idx int) string {
	if len(groups) <= idx {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(stripTags(groups[idx])))
}

func stripTags(input string) string {
	inTag := false
	var sb strings.Builder
	for _, r := range input {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}

func absURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return baseURL + raw
}

func hasNextPage(pageHTML string, next int) bool {
	return strings.Contains(pageHTML, "page="+strconv.Itoa(next))
}

func withPage(rawURL string, page int) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	q.Set("page", strconv.Itoa(page))
	u.RawQuery = q.Encode()
	return u.String()
}

func pageNum(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func cloneParams(params map[string]string) map[string]string {
	out := make(map[string]string, len(params)+1)
	for k, v := range params {
		out[k] = v
	}
	return out
}

func joinNonEmpty(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}
