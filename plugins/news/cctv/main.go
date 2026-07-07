package main

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&CCTVPlugin{})
}

type CCTVPlugin struct{}

const (
	baseURL     = "https://news.cctv.com"
	jsonpBase   = "https://news.cctv.com/2019/07/gaiban/cmsdatainterface/page"
	defaultUA   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var sectionLabels = map[string]string{
	"news":    "新闻",
	"china":   "国内",
	"world":   "国际",
	"society": "社会",
	"law":     "法治",
	"ent":     "文娱",
	"tech":    "科技",
	"life":    "生活",
}

var (
	reContentDate  = regexp.MustCompile(`var\s+contentdate\s*=\s*'([^']+)'`)
	rePublishDate  = regexp.MustCompile(`var\s+publishDate\s*=\s*"([^"]+)"`)
	reCommentBrief = regexp.MustCompile(`var\s+commentbreif\s*=\s*"([^"]*)"`)
	reVideoCode    = regexp.MustCompile(`\[!--begin:htmlVideoCode--\].*?\[!--end:htmlVideoCode--\]`)
	reDateInURL    = regexp.MustCompile(`/(\d{4})/(\d{2})/(\d{2})/`)
	reProtocolRel  = regexp.MustCompile(`(src|data-img)="//`)
	reImgSrc       = regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
)

type jsonpResponse struct {
	Data jsonpData `json:"data"`
}

type jsonpData struct {
	List  []jsonpItem `json:"list"`
	Total int         `json:"total"`
}

type jsonpItem struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	FocusDate  string `json:"focus_date"`
	URL        string `json:"url"`
	Image      string `json:"image"`
	Brief      string `json:"brief"`
	Keywords   string `json:"keywords"`
}

func (p *CCTVPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/cctv/list":
		return fetchList(req.Params)
	case req.Route == "/cctv/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(params map[string]string) (*sdk.FeedResult, error) {
	section := strings.TrimSpace(params["section"])
	if section == "" {
		section = "news"
	}
	if _, ok := sectionLabels[section]; !ok {
		return nil, fmt.Errorf("unknown section: %s", section)
	}

	page := parsePage(params["page"])
	if page < 1 {
		page = 1
	}

	data, err := fetchJSONP(section, page)
	if err != nil {
		return nil, err
	}

	label := sectionLabels[section]
	items := parseListItems(data.List)
	if len(items) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("央视网 - %s", label),
		Description: "央视网新闻频道图文资讯",
		Items:       items,
	}

	if hasNextPage(section, page) {
		result.HasMore = true
		result.Next = map[string]string{
			"page": strconv.Itoa(page + 1),
		}
	}

	return result, nil
}

func fetchJSONP(section string, page int) (*jsonpData, error) {
	url := fmt.Sprintf("%s/%s_%d.jsonp?cb=%s", jsonpBase, section, page, section)

	body, status, err := host.HTTPGet(url, map[string]string{
		"User-Agent": defaultUA,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch list failed: %w", err)
	}
	if status == 404 {
		return nil, fmt.Errorf("page %d not found", page)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("list http status %d", status)
	}

	payload := stripJSONPWrapper(string(body), section)
	var resp jsonpResponse
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		return nil, fmt.Errorf("parse list json: %w", err)
	}
	if len(resp.Data.List) == 0 {
		return nil, fmt.Errorf("empty list on page %d", page)
	}

	return &resp.Data, nil
}

func stripJSONPWrapper(raw, callback string) string {
	raw = strings.TrimSpace(raw)
	prefix := callback + "("
	if strings.HasPrefix(raw, prefix) && strings.HasSuffix(raw, ")") {
		return raw[len(prefix) : len(raw)-1]
	}
	if idx := strings.Index(raw, "({"); idx >= 0 && strings.HasSuffix(raw, ")") {
		return raw[idx+1 : len(raw)-1]
	}
	return raw
}

func parseListItems(list []jsonpItem) []sdk.FeedItem {
	seen := make(map[string]struct{})
	var items []sdk.FeedItem

	for _, entry := range list {
		title := strings.TrimSpace(entry.Title)
		articleURL := normalizeArticleURL(entry.URL)
		if title == "" || articleURL == "" || !isSupportedArticleURL(articleURL) {
			continue
		}
		if _, exists := seen[articleURL]; exists {
			continue
		}
		seen[articleURL] = struct{}{}

		cover := firstNonEmpty(entry.Image)
		summary := strings.TrimSpace(entry.Brief)

		items = append(items, sdk.FeedItem{
			ID:          articleURL,
			Title:       title,
			URL:         articleURL,
			Summary:     summary,
			Cover:       cover,
			Image:       cover,
			PublishedAt: parseFocusDate(entry.FocusDate, articleURL),
			Tags:        splitKeywords(entry.Keywords),
		})
	}

	return items
}

func hasNextPage(section string, page int) bool {
	url := fmt.Sprintf("%s/%s_%d.jsonp?cb=%s", jsonpBase, section, page+1, section)
	_, status, err := host.HTTPGet(url, map[string]string{
		"User-Agent": defaultUA,
	})
	if err != nil {
		return false
	}
	return status >= 200 && status < 300
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	articleURL, err := resolveArticleURL(id)
	if err != nil {
		return nil, err
	}
	if !isSupportedArticleURL(articleURL) {
		return nil, fmt.Errorf("unsupported article type")
	}

	body, status, err := host.HTTPGet(articleURL, map[string]string{
		"User-Agent": defaultUA,
		"Accept":     "text/html,application/xhtml+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch article failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("article http status %d", status)
	}

	htmlText := string(body)
	item, err := parseArticlePage(articleURL, htmlText)
	if err != nil {
		return nil, err
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{*item},
	}, nil
}

func parseArticlePage(articleURL, htmlText string) (*sdk.FeedItem, error) {
	contentMatch := reContentDate.FindStringSubmatch(htmlText)
	if len(contentMatch) < 2 {
		return nil, fmt.Errorf("article content not found")
	}

	contentHTML := unescapeJSString(contentMatch[1])
	contentHTML = cleanArticleHTML(contentHTML)

	title := extractTitle(htmlText)
	if title == "" {
		return nil, fmt.Errorf("no title found")
	}

	summary := ""
	if m := reCommentBrief.FindStringSubmatch(htmlText); len(m) > 1 {
		summary = strings.TrimSpace(m[1])
	}

	publishedAt := ""
	if m := rePublishDate.FindStringSubmatch(htmlText); len(m) > 1 {
		publishedAt = parsePublishDate(strings.TrimSpace(m[1]))
	}
	if publishedAt == "" {
		publishedAt = publishedAtFromURL(articleURL)
	}

	cover := extractOGImage(htmlText)
	if isGenericCover(cover) {
		cover = ""
	}
	if cover == "" {
		cover = extractFirstImage(contentHTML)
	}
	content := buildContent(cover, summary, contentHTML)

	return &sdk.FeedItem{
		ID:          articleURL,
		Title:       title,
		URL:         articleURL,
		Summary:     summary,
		Cover:       cover,
		Image:       cover,
		Content:     content,
		PublishedAt: publishedAt,
	}, nil
}

func extractTitle(htmlText string) string {
	if m := regexp.MustCompile(`<title>([^<]+)</title>`).FindStringSubmatch(htmlText); len(m) > 1 {
		title := strings.TrimSpace(html.UnescapeString(m[1]))
		title = strings.Split(title, "_")[0]
		return strings.TrimSpace(title)
	}
	return ""
}

func extractOGImage(htmlText string) string {
	re := regexp.MustCompile(`<meta[^>]+property="og:image"[^>]+content="([^"]+)"`)
	if m := re.FindStringSubmatch(htmlText); len(m) > 1 {
		return normalizeAssetURL(m[1])
	}
	re2 := regexp.MustCompile(`<meta[^>]+content="([^"]+)"[^>]+property="og:image"`)
	if m := re2.FindStringSubmatch(htmlText); len(m) > 1 {
		return normalizeAssetURL(m[1])
	}
	return ""
}

func unescapeJSString(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\'`, "'")
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return html.UnescapeString(s)
}

func cleanArticleHTML(contentHTML string) string {
	contentHTML = reVideoCode.ReplaceAllString(contentHTML, "")
	contentHTML = reProtocolRel.ReplaceAllString(contentHTML, `$1="https://`)
	contentHTML = strings.TrimSpace(contentHTML)
	return contentHTML
}

func isGenericCover(url string) bool {
	url = strings.ToLower(strings.TrimSpace(url))
	return url == "" ||
		strings.Contains(url, "newslog200.jpg") ||
		strings.Contains(url, "templet/common")
}

func extractFirstImage(contentHTML string) string {
	if m := reImgSrc.FindStringSubmatch(contentHTML); len(m) > 1 {
		return normalizeAssetURL(m[1])
	}
	return ""
}

func buildContent(cover, summary, contentHTML string) string {
	var sb strings.Builder
	if cover != "" && !strings.Contains(contentHTML, cover) {
		sb.WriteString(fmt.Sprintf(`<img src="%s" style="max-width:100%%;margin-bottom:1rem;"/>`+"\n", cover))
	}
	if summary != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>%s</strong></p>\n", html.EscapeString(summary)))
	}
	if contentHTML != "" {
		sb.WriteString(contentHTML)
	}
	return sb.String()
}

func normalizeArticleURL(raw string) string {
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
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return baseURL + raw
}

func normalizeAssetURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	return raw
}

func resolveArticleURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("missing id parameter")
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw, nil
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw, nil
	}
	if strings.HasPrefix(raw, "/") {
		return baseURL + raw, nil
	}
	if strings.Contains(raw, "ARTI") && strings.HasSuffix(raw, ".shtml") {
		return baseURL + "/" + strings.TrimPrefix(raw, "/"), nil
	}
	return "", fmt.Errorf("invalid article id: %s", raw)
}

func isSupportedArticleURL(url string) bool {
	lower := strings.ToLower(url)
	if !strings.Contains(lower, "news.cctv.com") {
		return false
	}
	if strings.Contains(lower, "tv.cctv.com") || strings.Contains(lower, "/vide") {
		return false
	}
	return strings.Contains(lower, "/arti")
}

func parseFocusDate(raw, articleURL string) string {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.ParseInLocation(format, raw, time.Local); err == nil {
				return t.Format(time.RFC3339)
			}
		}
	}
	return publishedAtFromURL(articleURL)
}

func parsePublishDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) >= 14 {
		if t, err := time.ParseInLocation("20060102150405", raw[:14], time.Local); err == nil {
			return t.Format(time.RFC3339)
		}
	}
	return ""
}

func publishedAtFromURL(articleURL string) string {
	if m := reDateInURL.FindStringSubmatch(articleURL); len(m) == 4 {
		dateStr := fmt.Sprintf("%s-%s-%s", m[1], m[2], m[3])
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			return t.Format(time.RFC3339)
		}
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func splitKeywords(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ' ' || r == ',' || r == '，'
	})
	var tags []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = normalizeAssetURL(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func parsePage(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}
