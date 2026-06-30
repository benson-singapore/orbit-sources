package main

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL      = "https://www.nationalgeographic.com"
	defaultSize  = 10
	maxSize      = 20
	defaultAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"
	stateMarker  = "window['__natgeo__']="
)

var channelLabels = map[string]string{
	"home":            "NatGeo · 首页",
	"animals":         "NatGeo · Animals",
	"travel":          "NatGeo · Travel",
	"science":         "NatGeo · Science",
	"environment":     "NatGeo · Environment",
	"history":         "NatGeo · History & Culture",
	"health":          "NatGeo · Health",
	"search":          "NatGeo · 搜索",
	"history-culture": "NatGeo · History & Culture",
}

var imageCropPrefs = []string{"16x9", "3x2", "raw", "square"}

var walkKeyOrder = []string{
	"frms", "mods", "mnu", "latest", "edgs", "tiles", "items",
	"content", "home", "hub", "header", "page",
}

func main() {
	sdk.Run(&NatGeoPlugin{})
}

type NatGeoPlugin struct{}

type listItem struct {
	Title   string
	URL     string
	Summary string
	Image   string
}

func (p *NatGeoPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	category := normalizeCategory(req.Params["category"])
	size := parseInt(req.Params["size"], defaultSize)
	if size > maxSize {
		size = maxSize
	}

	sectionURL := sectionURLFromCategory(category)
	body, status, err := host.HTTPGet(sectionURL, headers(sectionURL))
	if err != nil {
		return nil, fmt.Errorf("fetch section failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("section http status %d", status)
	}

	html := string(body)
	listItems := parseListItems(html, category)
	if len(listItems) == 0 {
		listItems = fallbackListItems(html, category)
	}
	if len(listItems) == 0 {
		return nil, fmt.Errorf("no list items found in %s", sectionURL)
	}

	selected, hasMore, next := paginateItems(listItems, req.Params, size)
	if len(selected) == 0 {
		return &sdk.FeedResult{
			Title:       feedTitle(category, req.ChannelID),
			Description: "已无更多图片",
			Items:       []sdk.FeedItem{},
			HasMore:     false,
		}, nil
	}

	items := make([]sdk.FeedItem, 0, len(selected))
	for _, item := range selected {
		if feedItem, ok := listItemToFeedItem(item); ok {
			items = append(items, feedItem)
		}
	}

	return &sdk.FeedResult{
		Title:       feedTitle(category, req.ChannelID),
		Description: fmt.Sprintf("实时拉取 NatGeo「%s」图片（list 页优先，不落库）", category),
		Items:       items,
		HasMore:     hasMore,
		Next:        next,
	}, nil
}

func paginateItems(all []listItem, params map[string]string, size int) ([]listItem, bool, map[string]string) {
	start := 0
	if lastID := strings.TrimSpace(params["lastId"]); lastID != "" {
		start = indexAfterLastID(all, lastID)
	} else {
		page := parseInt(params["page"], 1)
		if page < 1 {
			page = 1
		}
		start = (page - 1) * size
	}

	if start >= len(all) {
		return nil, false, nil
	}

	end := start + size
	if end > len(all) {
		end = len(all)
	}
	selected := all[start:end]
	hasMore := end < len(all)
	if !hasMore || len(selected) == 0 {
		return selected, false, nil
	}

	next := copyParams(params)
	next["lastId"] = itemID(selected[len(selected)-1])
	delete(next, "page")
	return selected, true, next
}

func indexAfterLastID(all []listItem, lastID string) int {
	for i, item := range all {
		if itemID(item) == lastID || extractArticleID(item.URL) == lastID {
			return i + 1
		}
	}
	return 0
}

func itemID(item listItem) string {
	slug := extractArticleID(item.URL)
	if slug == "" {
		return item.URL
	}
	return "natgeo-" + slug
}

func copyParams(params map[string]string) map[string]string {
	out := make(map[string]string, len(params)+1)
	for key, value := range params {
		out[key] = value
	}
	return out
}

func parseListItems(html, category string) []listItem {
	stateJSON, ok := extractNatGeoStateJSON(html)
	if !ok {
		return nil
	}

	var state any
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil
	}

	root := contentRoot(state, category)
	if root == nil {
		return nil
	}

	seen := map[string]bool{}
	items := make([]listItem, 0, 32)
	walkListJSON(root, &items, seen, category)
	return items
}

func contentRoot(state any, category string) any {
	root, ok := state.(map[string]any)
	if !ok {
		return nil
	}
	page, _ := root["page"].(map[string]any)
	if page == nil {
		return nil
	}
	content, _ := page["content"].(map[string]any)
	if content == nil {
		return nil
	}

	if isHomeCategory(category) {
		if home, ok := content["home"]; ok {
			return home
		}
	}
	if hub, ok := content["hub"]; ok {
		return hub
	}
	return content
}

func walkListJSON(value any, items *[]listItem, seen map[string]bool, category string) {
	switch node := value.(type) {
	case map[string]any:
		if item, ok := listItemFromObject(node); ok && matchesCategory(item.URL, category) {
			if !seen[item.URL] {
				seen[item.URL] = true
				*items = append(*items, item)
			}
		}
		walkMapChildren(node, items, seen, category)
	case []any:
		for _, child := range node {
			walkListJSON(child, items, seen, category)
		}
	}
}

func walkMapChildren(node map[string]any, items *[]listItem, seen map[string]bool, category string) {
	visited := map[string]bool{}
	for _, key := range walkKeyOrder {
		child, ok := node[key]
		if !ok {
			continue
		}
		visited[key] = true
		walkListJSON(child, items, seen, category)
	}

	remaining := make([]string, 0, len(node))
	for key := range node {
		if !visited[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		walkListJSON(node[key], items, seen, category)
	}
}

func listItemFromObject(obj map[string]any) (listItem, bool) {
	title, _ := obj["title"].(string)
	title = strings.TrimSpace(title)
	if title == "" {
		return listItem{}, false
	}

	imgObj, ok := obj["img"].(map[string]any)
	if !ok {
		return listItem{}, false
	}
	image := pickImageFromCrps(imgObj)
	if image == "" {
		return listItem{}, false
	}

	articleURL := articleURLFromCTAs(obj["ctas"])
	if articleURL == "" {
		return listItem{}, false
	}

	summary := firstNonEmpty(
		stringField(obj, "description"),
		stringField(obj, "abstract"),
		imageCaption(imgObj),
	)

	return listItem{
		Title:   cleanupText(title),
		URL:     articleURL,
		Summary: cleanupText(summary),
		Image:   image,
	}, true
}

func articleURLFromCTAs(raw any) string {
	ctas, ok := raw.([]any)
	if !ok {
		return ""
	}
	for _, entry := range ctas {
		cta, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		url, _ := cta["url"].(string)
		if strings.Contains(url, "/article/") {
			return url
		}
	}
	return ""
}

func pickImageFromCrps(imgObj map[string]any) string {
	crps, ok := imgObj["crps"].([]any)
	if !ok {
		return ""
	}

	for _, pref := range imageCropPrefs {
		for _, entry := range crps {
			crop, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			name, _ := crop["nm"].(string)
			if name != pref {
				continue
			}
			if url, _ := crop["url"].(string); url != "" {
				return enhanceImageURL(url)
			}
		}
	}

	for _, entry := range crps {
		crop, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if url, _ := crop["url"].(string); url != "" {
			return enhanceImageURL(url)
		}
	}
	return ""
}

func imageCaption(imgObj map[string]any) string {
	caption := stringField(imgObj, "dsc")
	if isWeakCaption(caption) {
		return ""
	}
	return caption
}

func isWeakCaption(caption string) bool {
	caption = strings.TrimSpace(caption)
	if caption == "" {
		return true
	}
	lower := strings.ToLower(caption)
	if lower == "tk" || lower == "caption to come." || len(caption) < 12 {
		return true
	}
	return false
}

func matchesCategory(articleURL, category string) bool {
	if isHomeCategory(category) {
		return true
	}

	section := categoryToSection(category)
	if strings.Contains(articleURL, "/"+section+"/article/") {
		return true
	}
	if strings.Contains(articleURL, "/premium/article/") {
		return true
	}
	return false
}

func categoryToSection(category string) string {
	switch normalizeCategory(category) {
	case "history-culture":
		return "history"
	default:
		return normalizeCategory(category)
	}
}

func normalizeCategory(category string) string {
	category = strings.TrimSpace(strings.ToLower(category))
	category = strings.Trim(category, "/")
	if category == "" || category == "latest" {
		return "home"
	}
	return category
}

func isHomeCategory(category string) bool {
	return normalizeCategory(category) == "home"
}

func enhanceImageURL(raw string) string {
	url := strings.TrimSpace(raw)
	if url == "" {
		return ""
	}
	if strings.Contains(url, "?") {
		return url
	}
	return url + "?w=1200"
}

func extractNatGeoStateJSON(html string) ([]byte, bool) {
	start := strings.Index(html, stateMarker)
	if start < 0 {
		return nil, false
	}
	start += len(stateMarker)

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(html); i++ {
		ch := html[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return []byte(html[start : i+1]), true
			}
		}
	}
	return nil, false
}

func fallbackListItems(html, category string) []listItem {
	links := extractArticleLinks(html)
	items := make([]listItem, 0, len(links))
	for _, link := range links {
		if !matchesCategory(link, category) {
			continue
		}
		if item, ok := fetchArticleAsItem(link); ok {
			items = append(items, listItem{
				Title:   item.Title,
				URL:     item.URL,
				Summary: item.Summary,
				Image:   item.Image,
			})
		}
	}
	return items
}

func listItemToFeedItem(item listItem) (sdk.FeedItem, bool) {
	if item.URL == "" || item.Image == "" {
		return sdk.FeedItem{}, false
	}

	title := item.Title
	if title == "" {
		title = "National Geographic"
	}
	summary := item.Summary
	if summary == "" {
		summary = title
	}

	return sdk.FeedItem{
		ID:      itemID(item),
		Title:   title,
		URL:     item.URL,
		Summary: summary,
		Content: buildListContent(item.Image, summary),
		Cover:   item.Image,
		Image:   item.Image,
	}, true
}

func buildListContent(image, summary string) string {
	var sb strings.Builder
	if image != "" {
		sb.WriteString(fmt.Sprintf(
			`<figure style="margin:0 0 1rem;"><img src="%s" alt="" style="width:100%%;max-width:100%%;border-radius:8px;object-fit:cover;"/></figure>`,
			image,
		))
		sb.WriteString("\n")
	}
	if summary != "" {
		sb.WriteString(fmt.Sprintf(`<p style="margin:0;line-height:1.6;">%s</p>`, html.EscapeString(summary)))
	}
	return sb.String()
}

func sectionURLFromCategory(category string) string {
	c := normalizeCategory(category)
	switch c {
	case "home":
		return baseURL + "/"
	case "history-culture":
		return baseURL + "/history/"
	default:
		return baseURL + "/" + c + "/"
	}
}

func feedTitle(category, channelID string) string {
	if label := channelLabels[category]; label != "" {
		return label
	}
	if label := channelLabels[channelID]; label != "" {
		return label
	}
	return "NatGeo"
}

func fetchArticleAsItem(link string) (sdk.FeedItem, bool) {
	body, status, err := host.HTTPGet(link, headers(link))
	if err != nil || status < 200 || status >= 300 {
		return sdk.FeedItem{}, false
	}
	html := string(body)

	title := extractMeta(html, "property", "og:title")
	if title == "" {
		title = extractTitleTag(html)
	}
	if title == "" {
		title = "National Geographic"
	}

	image := extractMeta(html, "property", "og:image")
	if image == "" {
		image = extractMeta(html, "name", "twitter:image")
	}
	if image == "" {
		return sdk.FeedItem{}, false
	}

	summary := extractMeta(html, "name", "description")
	if summary == "" {
		summary = title
	}

	id := extractArticleID(link)
	if id == "" {
		id = link
	}

	cleanTitle := cleanupText(title)
	cleanSummary := cleanupText(summary)

	return sdk.FeedItem{
		ID:      "natgeo-" + id,
		Title:   cleanTitle,
		URL:     link,
		Summary: cleanSummary,
		Content: buildListContent(image, cleanSummary),
		Cover:   image,
		Image:   image,
	}, true
}

func extractArticleLinks(html string) []string {
	re := regexp.MustCompile(`https://www\.nationalgeographic\.com/(?:[a-z-]+/article|premium/article)/[a-z0-9\-\/]+`)
	matches := re.FindAllString(html, -1)
	seen := map[string]bool{}
	links := make([]string, 0, len(matches))
	for _, m := range matches {
		if seen[m] {
			continue
		}
		seen[m] = true
		links = append(links, m)
	}
	return links
}

func extractMeta(html, attrName, attrValue string) string {
	quoted := regexp.QuoteMeta(attrValue)
	pattern := fmt.Sprintf(`<meta[^>]*%s=["']%s["'][^>]*content=["']([^"']+)["'][^>]*>`, attrName, quoted)
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(html)
	if len(m) > 1 {
		return cleanupText(m[1])
	}
	pattern2 := fmt.Sprintf(`<meta[^>]*content=["']([^"']+)["'][^>]*%s=["']%s["'][^>]*>`, attrName, quoted)
	re2 := regexp.MustCompile(pattern2)
	m2 := re2.FindStringSubmatch(html)
	if len(m2) > 1 {
		return cleanupText(m2[1])
	}
	return ""
}

func extractTitleTag(html string) string {
	re := regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	m := re.FindStringSubmatch(html)
	if len(m) > 1 {
		return cleanupText(m[1])
	}
	return ""
}

func extractArticleID(link string) string {
	parts := strings.Split(strings.Trim(link, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func stringField(obj map[string]any, key string) string {
	value, _ := obj[key].(string)
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func cleanupText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return s
}

func parseInt(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func headers(referer string) map[string]string {
	return map[string]string{
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
		"Cache-Control":   "no-cache",
		"Pragma":          "no-cache",
		"Referer":         referer,
		"User-Agent":      defaultAgent,
	}
}
