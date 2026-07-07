package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&HuanqiuPlugin{})
}

type HuanqiuPlugin struct{}

const (
	baseURL             = "https://www.huanqiu.com"
	defaultUA           = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	defaultAPIPageSize  = 24
	defaultHomePageSize = 20
)

var sectionURLs = map[string]string{
	"home":    "https://www.huanqiu.com/",
	"world":   "https://world.huanqiu.com/",
	"china":   "https://china.huanqiu.com/",
	"mil":     "https://mil.huanqiu.com/",
	"taiwan":  "https://taiwan.huanqiu.com/",
	"opinion": "https://opinion.huanqiu.com/",
	"finance": "https://finance.huanqiu.com/",
	"tech":    "https://tech.huanqiu.com/",
	"society": "https://society.huanqiu.com/",
	"health":  "https://health.huanqiu.com/",
	"sports":  "https://sports.huanqiu.com/",
	"auto":    "https://auto.huanqiu.com/",
	"ent":     "https://ent.huanqiu.com/",
}

var sectionLabels = map[string]string{
	"home":    "首页",
	"world":   "国际",
	"china":   "国内",
	"mil":     "军事",
	"taiwan":  "台海",
	"opinion": "评论",
	"finance": "财经",
	"tech":    "科技",
	"society": "社会",
	"health":  "健康",
	"sports":  "体育",
	"auto":    "汽车",
	"ent":     "大文娱",
}

func (p *HuanqiuPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch req.Route {
	case "/huanqiu/list":
		return fetchList(req.Params)
	case "/huanqiu/detail/:id":
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
		section = "home"
	}

	listURL, ok := sectionURLs[section]
	if !ok {
		return nil, fmt.Errorf("unknown section: %s", section)
	}

	page := parsePage(params["page"])
	var items []sdk.FeedItem
	if page == 1 {
		fetchURL := listURL
		if section == "home" {
			fetchURL = "https://m.huanqiu.com/"
		}

		body, status, err := host.HTTPGet(fetchURL, requestHeaders())
		if err != nil {
			return nil, fmt.Errorf("fetch list page failed: %w", err)
		}
		if status < 200 || status >= 300 {
			return nil, fmt.Errorf("list page http status %d", status)
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			return nil, fmt.Errorf("parse list page: %w", err)
		}

		items = parseListItems(doc, fetchURL)
	}

	apiItems, hasMore := fetchPagedListItems(section, listURL, page)
	items = appendUniqueItems(items, apiItems...)
	if len(items) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	label := sectionLabels[section]
	if label == "" {
		label = section
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("环球网 - %s", label),
		Description: "环球网文字新闻，覆盖国内、国际、军事、财经、科技、评论等栏目",
		Items:       items,
	}
	if hasMore {
		result.HasMore = true
		result.Next = map[string]string{
			"page": strconv.Itoa(page + 1),
		}
	}

	return result, nil
}

func parseListItems(doc *goquery.Document, listURL string) []sdk.FeedItem {
	var items []sdk.FeedItem

	appendItem := func(item sdk.FeedItem) {
		item.URL = normalizeURL(item.URL, listURL)
		item.ID = item.URL
		item.Title = strings.TrimSpace(item.Title)
		item.Cover = normalizeURL(item.Cover, listURL)
		item.Image = item.Cover
		if item.URL == "" || item.Title == "" || !validArticleURL(item.URL) {
			return
		}
		if item.PublishedAt == "" {
			item.PublishedAt = time.Now().UTC().Format(time.RFC3339)
		}
		items = appendUniqueItems(items, item)
	}

	doc.Find(".data-container .item").Each(func(_ int, node *goquery.Selection) {
		addlType := firstTextareaText(node, "item-addltype", "addltype")
		if addlType != "" && !strings.EqualFold(addlType, "article") && !strings.EqualFold(addlType, "normal") && !strings.EqualFold(addlType, "video") {
			return
		}

		aid := firstTextareaText(node, "item-aid", "aid")
		title := firstTextareaText(node, "item-title", "title")
		hostName := firstTextareaText(node, "item-cnf-host")
		href := firstTextareaText(node, "item-href", "href")
		cover := firstTextareaText(node, "item-cover", "cover")
		publishedAt := parseTimestamp(firstTextareaText(node, "item-time", "time", "ctime", "xtime"))

		if title == "" {
			return
		}
		if href == "" && aid != "" && hostName != "" {
			href = "https://" + hostName + "/article/" + aid
		}

		appendItem(sdk.FeedItem{
			Title:       title,
			URL:         href,
			Cover:       cover,
			PublishedAt: publishedAt,
		})
	})

	doc.Find(`a[href*="/article/"]`).Each(func(_ int, link *goquery.Selection) {
		href, _ := link.Attr("href")
		title := strings.TrimSpace(link.Text())
		if title == "" {
			title = strings.TrimSpace(link.AttrOr("title", ""))
		}
		if title == "" {
			title = strings.TrimSpace(link.Find("img").First().AttrOr("alt", ""))
		}

		cover := link.Find("img").First().AttrOr("src", "")
		if cover == "" {
			cover = link.Closest("li, div").Find("img").First().AttrOr("src", "")
		}

		appendItem(sdk.FeedItem{
			Title: title,
			URL:   href,
			Cover: cover,
		})
	})

	return items
}

func fetchPagedListItems(section, listURL string, page int) ([]sdk.FeedItem, bool) {
	if section == "home" {
		return fetchHomeRecommendItems(page)
	}
	return fetchSectionAPIItems(listURL, page)
}

func fetchHomeRecommendItems(page int) ([]sdk.FeedItem, bool) {
	offset := (page - 1) * defaultHomePageSize
	apiURL := fmt.Sprintf("https://m.huanqiu.com/api/index/recommend?offset=%d&limit=%d", offset, defaultHomePageSize)
	body, status, err := host.HTTPGet(apiURL, apiRequestHeaders("https://m.huanqiu.com/"))
	if err != nil || status < 200 || status >= 300 {
		return nil, false
	}
	return parseAPIItems(body, baseURL, defaultHomePageSize)
}

func fetchSectionAPIItems(listURL string, page int) ([]sdk.FeedItem, bool) {
	parsed, err := url.Parse(listURL)
	if err != nil || parsed.Host == "" {
		return nil, false
	}

	channelURL := parsed.Scheme + "://" + parsed.Host + "/api/channel_pc"
	body, status, err := host.HTTPGet(channelURL, apiRequestHeaders(listURL))
	if err != nil || status < 200 || status >= 300 {
		return nil, false
	}

	var channel channelResponse
	if err := json.Unmarshal(body, &channel); err != nil {
		return nil, false
	}

	nodes := collectChannelNodes(channel.Children)
	if len(nodes) == 0 {
		return nil, false
	}

	offset := (page - 1) * defaultAPIPageSize
	listAPIURL := parsed.Scheme + "://" + parsed.Host + "/api/list?" + url.Values{
		"node":   []string{quoteNodes(nodes)},
		"offset": []string{strconv.Itoa(offset)},
		"limit":  []string{strconv.Itoa(defaultAPIPageSize)},
	}.Encode()

	body, status, err = host.HTTPGet(listAPIURL, apiRequestHeaders(listURL))
	if err != nil || status < 200 || status >= 300 {
		return nil, false
	}
	return parseAPIItems(body, listURL, defaultAPIPageSize)
}

type channelResponse struct {
	Children map[string]channelNode `json:"children"`
}

type channelNode struct {
	Node     string                 `json:"node"`
	Children map[string]channelNode `json:"children"`
}

type apiListResponse struct {
	List []apiListItem `json:"list"`
}

type apiListItem struct {
	AID            string      `json:"aid"`
	Title          string      `json:"title"`
	Summary        string      `json:"summary"`
	AddlType       string      `json:"addltype"`
	Cover          string      `json:"cover"`
	Host           string      `json:"host"`
	CTime          string      `json:"ctime"`
	XTime          string      `json:"xtime"`
	ExtDisplayTime string      `json:"ext_displaytime"`
	Source         apiSource   `json:"source"`
	TypeData       apiTypedata `json:"typedata"`
	ExtSerious     string      `json:"ext-serious"`
}

type apiSource struct {
	Name string `json:"name"`
}

type apiTypedata struct {
	Gallery apiGallery `json:"gallery"`
	Video   apiVideo   `json:"video"`
}

type apiGallery struct {
	Members []apiMedia `json:"members"`
}

type apiVideo struct {
	Members []apiMedia `json:"members"`
}

type apiMedia struct {
	URL   string `json:"url"`
	Cover string `json:"cover"`
}

func collectChannelNodes(children map[string]channelNode) []string {
	var nodes []string
	var walk func(map[string]channelNode)
	walk = func(items map[string]channelNode) {
		for _, item := range items {
			if strings.TrimSpace(item.Node) != "" {
				nodes = append(nodes, item.Node)
			}
			if len(item.Children) > 0 {
				walk(item.Children)
			}
		}
	}
	walk(children)
	return nodes
}

func quoteNodes(nodes []string) string {
	quoted := make([]string, 0, len(nodes))
	for _, node := range nodes {
		node = strings.TrimSpace(node)
		if node != "" {
			quoted = append(quoted, strconv.Quote(node))
		}
	}
	return strings.Join(quoted, ",")
}

func parseAPIItems(body []byte, base string, pageSize int) ([]sdk.FeedItem, bool) {
	var response apiListResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, false
	}

	var items []sdk.FeedItem
	for _, article := range response.List {
		item := apiArticleToFeedItem(article, base)
		if item == nil {
			continue
		}
		items = appendUniqueItems(items, *item)
	}
	return items, len(response.List) >= pageSize
}

func apiArticleToFeedItem(article apiListItem, base string) *sdk.FeedItem {
	addlType := strings.TrimSpace(article.AddlType)
	if strings.EqualFold(addlType, "gallery") {
		return nil
	}

	aid := strings.TrimSpace(article.AID)
	title := strings.TrimSpace(article.Title)
	if aid == "" || title == "" {
		return nil
	}

	hostName := strings.TrimSpace(article.Host)
	if hostName == "" {
		if parsed, err := url.Parse(base); err == nil {
			hostName = parsed.Host
		}
	}
	if hostName == "" {
		hostName = "www.huanqiu.com"
	}

	articleURL := normalizeURL("https://"+hostName+"/article/"+aid, base)
	if !validArticleURL(articleURL) {
		return nil
	}

	cover := article.Cover
	if cover == "" && len(article.TypeData.Video.Members) > 0 {
		cover = article.TypeData.Video.Members[0].Cover
	}
	if cover == "" && len(article.TypeData.Gallery.Members) > 0 {
		cover = article.TypeData.Gallery.Members[0].URL
	}
	cover = normalizeURL(cover, articleURL)

	publishedAt := parseTimestamp(firstNonEmpty(article.XTime, article.CTime, article.ExtDisplayTime))
	if publishedAt == "" {
		publishedAt = time.Now().UTC().Format(time.RFC3339)
	}

	return &sdk.FeedItem{
		ID:          articleURL,
		Title:       title,
		URL:         articleURL,
		Summary:     strings.TrimSpace(article.Summary),
		Author:      strings.TrimSpace(article.Source.Name),
		Cover:       cover,
		Image:       cover,
		PublishedAt: publishedAt,
	}
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	articleURL := normalizeArticleURL(id)
	if articleURL == "" {
		return nil, fmt.Errorf("invalid article id")
	}

	body, status, err := host.HTTPGet(articleURL, requestHeaders())
	if err != nil {
		return nil, fmt.Errorf("fetch article failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("article http status %d", status)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse article page: %w", err)
	}

	item, err := pageToFeedItem(articleURL, doc)
	if err != nil {
		return nil, err
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{*item},
	}, nil
}

func pageToFeedItem(articleURL string, doc *goquery.Document) (*sdk.FeedItem, error) {
	title := textareaText(doc.Selection, "article-title")
	if title == "" {
		title = strings.TrimSpace(doc.Find("meta[property=\"og:title\"]").AttrOr("content", ""))
	}
	if title == "" {
		title = strings.TrimSpace(doc.Find("title").First().Text())
	}
	if title == "" {
		return nil, fmt.Errorf("no title found")
	}

	summary := textareaText(doc.Selection, "article-summary")
	if summary == "" {
		summary = strings.TrimSpace(doc.Find("meta[property=\"og:description\"]").AttrOr("content", ""))
	}
	if summary == "" {
		summary = strings.TrimSpace(doc.Find("meta[name=\"description\"]").AttrOr("content", ""))
	}

	contentHTML := textareaText(doc.Selection, "article-content")
	contentHTML = cleanArticleHTML(contentHTML)

	cover := normalizeURL(textareaText(doc.Selection, "article-cover"), articleURL)
	if cover == "" {
		cover = normalizeURL(doc.Find("meta[property=\"og:image\"]").AttrOr("content", ""), articleURL)
	}
	if cover == "" {
		cover = firstImageURL(contentHTML, articleURL)
	}

	publishedAt := parseTimestamp(textareaText(doc.Selection, "article-time"))
	if publishedAt == "" {
		publishedAt = parseTimestamp(textareaText(doc.Selection, "article-ext-xtime"))
	}
	if publishedAt == "" {
		publishedAt = time.Now().UTC().Format(time.RFC3339)
	}

	author := strings.TrimSpace(textareaText(doc.Selection, "article-author"))
	source := cleanInlineHTML(textareaText(doc.Selection, "article-source-name"))
	editor := cleanInlineHTML(textareaText(doc.Selection, "article-editor-name"))
	if author == "" {
		author = source
	}

	return &sdk.FeedItem{
		ID:          articleURL,
		Title:       title,
		URL:         articleURL,
		Summary:     summary,
		Author:      author,
		Cover:       cover,
		Image:       cover,
		Content:     buildDescription(cover, summary, source, editor, contentHTML),
		PublishedAt: publishedAt,
	}, nil
}

func requestHeaders() map[string]string {
	return map[string]string{
		"User-Agent":      defaultUA,
		"Accept":          "text/html,application/xhtml+xml",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
	}
}

func apiRequestHeaders(referer string) map[string]string {
	headers := requestHeaders()
	headers["Accept"] = "application/json,text/plain,*/*"
	headers["Referer"] = referer
	headers["X-Requested-With"] = "XMLHttpRequest"
	return headers
}

func textareaText(scope *goquery.Selection, className string) string {
	raw := strings.TrimSpace(scope.Find("textarea." + className).First().Text())
	return strings.TrimSpace(html.UnescapeString(raw))
}

func firstTextareaText(scope *goquery.Selection, classNames ...string) string {
	for _, className := range classNames {
		if text := textareaText(scope, className); text != "" {
			return text
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parsePage(raw string) int {
	page, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func appendUniqueItems(items []sdk.FeedItem, next ...sdk.FeedItem) []sdk.FeedItem {
	seen := make(map[string]struct{}, len(items)+len(next))
	for _, item := range items {
		key := feedItemKey(item)
		if key != "" {
			seen[key] = struct{}{}
		}
	}

	for _, item := range next {
		key := feedItemKey(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}
	return items
}

func feedItemKey(item sdk.FeedItem) string {
	if id := articleIDFromURL(item.URL); id != "" {
		return id
	}
	if id := articleIDFromURL(item.ID); id != "" {
		return id
	}
	if item.URL != "" {
		return item.URL
	}
	return item.ID
}

func validArticleURL(articleURL string) bool {
	return articleIDFromURL(articleURL) != ""
}

func articleIDFromURL(articleURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(articleURL))
	if err != nil {
		return ""
	}
	path := strings.Trim(parsed.Path, "/")
	parts := strings.Split(path, "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] != "article" {
			continue
		}
		id := strings.TrimSpace(parts[i+1])
		if len(id) < 6 {
			return ""
		}
		for _, r := range id {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				continue
			}
			return ""
		}
		return id
	}
	return ""
}

func parseTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return ""
	}

	var t time.Time
	switch {
	case n > 1_000_000_000_000:
		t = time.UnixMilli(n)
	case n > 1_000_000_000:
		t = time.Unix(n, 0)
	default:
		return ""
	}
	return t.Format(time.RFC3339)
}

func normalizeArticleURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/") {
		return normalizeURL(raw, baseURL)
	}
	return baseURL + "/article/" + raw
}

func normalizeURL(raw, base string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = html.UnescapeString(raw)
	if strings.HasPrefix(raw, "///") {
		raw = "/" + strings.TrimLeft(raw, "/")
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if !parsed.IsAbs() {
		baseParsed, baseErr := url.Parse(base)
		if baseErr == nil {
			parsed = baseParsed.ResolveReference(parsed)
		}
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func cleanArticleHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
	if err != nil {
		return raw
	}

	doc.Find("script, style, iframe, adv-loader, xpeng, .ad, [id*=\"ad\"], [class*=\"ad\"]").Remove()
	doc.Find("img").Each(func(_ int, img *goquery.Selection) {
		src, _ := img.Attr("src")
		if normalized := normalizeURL(src, baseURL); normalized != "" {
			img.SetAttr("src", normalized)
		}
		img.RemoveAttr("data-src")
		img.RemoveAttr("loading")
	})

	result, err := doc.Find("body").Children().Html()
	if err != nil || strings.TrimSpace(result) == "" {
		result, _ = doc.Html()
	}
	return strings.TrimSpace(result)
}

func cleanInlineHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
	if err != nil {
		return raw
	}
	return strings.TrimSpace(doc.Text())
}

func firstImageURL(contentHTML, base string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(contentHTML))
	if err != nil {
		return ""
	}
	return normalizeURL(doc.Find("img").First().AttrOr("src", ""), base)
}

func buildDescription(cover, summary, source, editor, content string) string {
	var sb strings.Builder

	if cover != "" {
		sb.WriteString(fmt.Sprintf("<img src=\"%s\" style=\"max-width: 100%%; margin-bottom: 1rem;\"/>\n", cover))
	}
	if summary != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>%s</strong></p>\n", html.EscapeString(summary)))
	}
	if source != "" || editor != "" {
		sb.WriteString("<p>")
		if source != "" {
			sb.WriteString("来源：" + html.EscapeString(source))
		}
		if editor != "" {
			if source != "" {
				sb.WriteString("　")
			}
			sb.WriteString(html.EscapeString(editor))
		}
		sb.WriteString("</p>\n")
	}
	if content != "" {
		sb.WriteString("<div>")
		sb.WriteString(content)
		sb.WriteString("</div>")
	}

	return sb.String()
}
