package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&BBCChinesePlugin{})
}

type BBCChinesePlugin struct{}

const (
	baseURL       = "https://www.bbc.com"
	defaultUA     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	imageWidth    = "480"
	nextDataReStr = `<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`
)

var reNextData = regexp.MustCompile(nextDataReStr)

var sectionPaths = map[string]string{
	"home":      "/zhongwen/simp",
	"world":     "/zhongwen/topics/c83plve5vmjt/simp",
	"china":     "/zhongwen/topics/ckr7mn6r003t/simp",
	"hong-kong": "/zhongwen/topics/cezw73jk755t/simp",
	"taiwan":    "/zhongwen/topics/cd6qem06z92t/simp",
	"uk":        "/zhongwen/topics/c1ez1k4emn0t/simp",
	"business":  "/zhongwen/topics/cq8nqywy37yt/simp",
	"video":     "/zhongwen/topics/cgvl47l38e1t/simp",
}

var sectionLabels = map[string]string{
	"home":      "主页",
	"world":     "国际",
	"china":     "中国",
	"hong-kong": "香港",
	"taiwan":    "台湾",
	"uk":        "英国",
	"business":  "财经",
	"video":     "影片",
}

func (p *BBCChinesePlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/bbc-chinese/list":
		return fetchList(req.Params)
	case req.Route == "/bbc-chinese/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type nextData struct {
	Props struct {
		PageProps struct {
			PageData pageData `json:"pageData"`
		} `json:"pageProps"`
	} `json:"props"`
}

type pageData struct {
	Title      string     `json:"title"`
	Curations  []curation `json:"curations"`
	ActivePage int        `json:"activePage"`
	PageCount  int        `json:"pageCount"`
	Content    *content   `json:"content"`
	Metadata   articleMeta `json:"metadata"`
	Promo      articlePromo `json:"promo"`
}

type curation struct {
	Title     string    `json:"title"`
	Summaries []summary `json:"summaries"`
}

type summary struct {
	Type           string          `json:"type"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Link           string          `json:"link"`
	ImageURL       string          `json:"imageUrl"`
	FirstPublished json.RawMessage `json:"firstPublished"`
	LastPublished  json.RawMessage `json:"lastPublished"`
	ID             string          `json:"id"`
}

type articleMeta struct {
	FirstPublished json.RawMessage `json:"firstPublished"`
	LastPublished  json.RawMessage `json:"lastPublished"`
	Type           string          `json:"type"`
}

type articlePromo struct {
	Headlines struct {
		SEOHeadline string `json:"seoHeadline"`
	} `json:"headlines"`
	Images struct {
		DefaultPromoImage imageBlock `json:"defaultPromoImage"`
	} `json:"images"`
}

type content struct {
	Model struct {
		Blocks []block `json:"blocks"`
	} `json:"model"`
}

type block struct {
	Type  string          `json:"type"`
	Model json.RawMessage `json:"model"`
}

type blockModel struct {
	Text   string  `json:"text"`
	Blocks []block `json:"blocks"`
}

type imageBlock struct {
	Blocks []block `json:"blocks"`
}

type rawImageModel struct {
	Locator    string `json:"locator"`
	OriginCode string `json:"originCode"`
	AltText    string `json:"altText"`
}

func fetchList(params map[string]string) (*sdk.FeedResult, error) {
	section := strings.TrimSpace(params["section"])
	if section == "" {
		section = "home"
	}
	path, ok := sectionPaths[section]
	if !ok {
		return nil, fmt.Errorf("unknown section: %s", section)
	}

	page := parsePositiveInt(params["page"], 1)
	fetchURL := baseURL + path
	if section != "home" && page > 1 {
		fetchURL = fmt.Sprintf("%s?page=%d", fetchURL, page)
	}

	body, status, err := host.HTTPGet(fetchURL, map[string]string{
		"User-Agent": defaultUA,
		"Accept":     "text/html,application/xhtml+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch list page failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("list page http status %d", status)
	}

	pd, err := parsePageData(body)
	if err != nil {
		return nil, err
	}

	label := sectionLabels[section]
	if label == "" {
		label = section
	}

	items := summariesToItems(pd.Curations)
	if len(items) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("BBC News 中文 - %s", label),
		Description: "BBC中文网面向全球华人的新闻资讯",
		Items:       items,
	}

	if section != "home" && pd.PageCount > 0 && pd.ActivePage < pd.PageCount {
		result.HasMore = true
		result.Next = map[string]string{
			"page": strconv.Itoa(pd.ActivePage + 1),
		}
	}

	return result, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	id = strings.Trim(id, "/")
	if strings.Contains(id, "http") {
		id = articleIDFromURL(id)
	}
	if id == "" {
		return nil, fmt.Errorf("invalid article id")
	}

	articleURL := fmt.Sprintf("%s/zhongwen/articles/%s/simp", baseURL, id)
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

	pd, err := parsePageData(body)
	if err != nil {
		return nil, err
	}

	item, err := pageDataToItem(id, articleURL, pd)
	if err != nil {
		return nil, err
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{*item},
	}, nil
}

func parsePageData(body []byte) (*pageData, error) {
	match := reNextData.FindSubmatch(body)
	if len(match) < 2 {
		return nil, fmt.Errorf("__NEXT_DATA__ not found")
	}

	var data nextData
	if err := json.Unmarshal(match[1], &data); err != nil {
		return nil, fmt.Errorf("parse __NEXT_DATA__: %w", err)
	}

	return &data.Props.PageProps.PageData, nil
}

func summariesToItems(curations []curation) []sdk.FeedItem {
	seen := make(map[string]struct{})
	var items []sdk.FeedItem

	for _, c := range curations {
		for _, s := range c.Summaries {
			title := strings.TrimSpace(s.Title)
			link := strings.TrimSpace(s.Link)
			if title == "" || link == "" {
				continue
			}

			id := strings.TrimSpace(s.ID)
			if id == "" {
				id = articleIDFromURL(link)
			}
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}

			published := parsePublishedTime(s.FirstPublished)
			if published == "" {
				published = parsePublishedTime(s.LastPublished)
			}

			items = append(items, sdk.FeedItem{
				ID:          id,
				Title:       title,
				URL:         link,
				Summary:     strings.TrimSpace(s.Description),
				Cover:       normalizeImageURL(s.ImageURL),
				Image:       normalizeImageURL(s.ImageURL),
				PublishedAt: published,
			})
		}
	}

	return items
}

func pageDataToItem(id, articleURL string, pd *pageData) (*sdk.FeedItem, error) {
	title := strings.TrimSpace(pd.Promo.Headlines.SEOHeadline)
	if title == "" {
		title = strings.TrimSpace(pd.Title)
	}
	if title == "" {
		return nil, fmt.Errorf("no title found")
	}

	published := parsePublishedTime(pd.Metadata.FirstPublished)
	if published == "" {
		published = parsePublishedTime(pd.Metadata.LastPublished)
	}
	if published == "" {
		published = time.Now().UTC().Format(time.RFC3339)
	}

	cover := promoImageURL(pd.Promo.Images.DefaultPromoImage)
	summary := extractSummary(pd)

	var content string
	if pd.Content != nil {
		coverLocator := promoImageLocator(pd.Promo.Images.DefaultPromoImage)
		content = renderBody(pd.Content.Model.Blocks, summary, coverLocator)
	}

	return &sdk.FeedItem{
		ID:          id,
		Title:       title,
		URL:         articleURL,
		Summary:     summary,
		Cover:       cover,
		Image:       cover,
		Content:     content,
		PublishedAt: published,
	}, nil
}

func extractSummary(pd *pageData) string {
	if pd.Content == nil {
		return ""
	}
	for _, b := range pd.Content.Model.Blocks {
		if b.Type != "text" {
			continue
		}
		var model blockModel
		if err := json.Unmarshal(b.Model, &model); err != nil {
			continue
		}
		for _, inner := range model.Blocks {
			if inner.Type != "paragraph" {
				continue
			}
			var para blockModel
			if err := json.Unmarshal(inner.Model, &para); err != nil {
				continue
			}
			text := strings.TrimSpace(para.Text)
			if text != "" {
				return text
			}
		}
	}
	return ""
}

var skipBodyBlocks = map[string]struct{}{
	"headline":        {},
	"disclaimer":      {},
	"timestamp":       {},
	"continueReading": {},
	"mpu":             {},
	"wsoj":            {},
	"social":          {},
	"relatedContent":  {},
	"embed":           {},
	"video":           {},
	"group":           {},
	"links":           {},
	"link":            {},
	"byline":          {},
	"caption":         {},
	"altText":         {},
}

func renderBody(blocks []block, summary, coverLocator string) string {
	summary = strings.TrimSpace(summary)
	coverLocator = strings.TrimSpace(coverLocator)
	var sb strings.Builder
	skippedSummary := false
	skippedHeroImage := false
	for _, b := range blocks {
		if _, skip := skipBodyBlocks[b.Type]; skip {
			continue
		}
		switch b.Type {
		case "image":
			locator := imageLocatorFromBlock(b.Model)
			if !skippedHeroImage && coverLocator != "" && locator == coverLocator {
				skippedHeroImage = true
				continue
			}
			if img := renderImageBlock(b.Model); img != "" {
				sb.WriteString(img)
			}
		case "text", "subheadline":
			html, skipped := renderTextBlock(b.Model, summary, skippedSummary)
			if skipped {
				skippedSummary = true
			}
			sb.WriteString(html)
		case "heading":
			var model blockModel
			if err := json.Unmarshal(b.Model, &model); err == nil {
				text := strings.TrimSpace(model.Text)
				if text != "" {
					sb.WriteString(fmt.Sprintf("<h3>%s</h3>\n", escapeHTML(text)))
				}
			}
		}
	}
	return sb.String()
}

func renderTextBlock(raw json.RawMessage, summary string, skippedSummary bool) (string, bool) {
	var model blockModel
	if err := json.Unmarshal(raw, &model); err != nil {
		return "", skippedSummary
	}

	var sb strings.Builder
	for _, inner := range model.Blocks {
		switch inner.Type {
		case "paragraph":
			var para blockModel
			if err := json.Unmarshal(inner.Model, &para); err != nil {
				continue
			}
			text := strings.TrimSpace(para.Text)
			if text == "" {
				continue
			}
			if !skippedSummary && summary != "" && text == summary {
				skippedSummary = true
				continue
			}
			sb.WriteString(fmt.Sprintf("<p>%s</p>\n", escapeHTML(text)))
		case "unorderedList", "orderedList":
			sb.WriteString(renderListBlock(inner))
		}
	}
	return sb.String(), skippedSummary
}

func renderListBlock(b block) string {
	var model blockModel
	if err := json.Unmarshal(b.Model, &model); err != nil {
		return ""
	}

	tag := "ul"
	if b.Type == "orderedList" {
		tag = "ol"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<%s>\n", tag))
	for _, item := range model.Blocks {
		if item.Type != "listItem" {
			continue
		}
		var itemModel blockModel
		if err := json.Unmarshal(item.Model, &itemModel); err != nil {
			continue
		}
		text := collectBlockText(itemModel.Blocks)
		if text != "" {
			sb.WriteString(fmt.Sprintf("<li>%s</li>\n", escapeHTML(text)))
		}
	}
	sb.WriteString(fmt.Sprintf("</%s>\n", tag))
	return sb.String()
}

func collectBlockText(blocks []block) string {
	var parts []string
	for _, b := range blocks {
		if b.Type == "paragraph" {
			var para blockModel
			if err := json.Unmarshal(b.Model, &para); err == nil {
				if t := strings.TrimSpace(para.Text); t != "" {
					parts = append(parts, t)
				}
			}
		}
	}
	return strings.Join(parts, " ")
}

func imageLocatorFromBlock(raw json.RawMessage) string {
	var model blockModel
	if err := json.Unmarshal(raw, &model); err != nil {
		return ""
	}
	for _, inner := range model.Blocks {
		if inner.Type != "rawImage" {
			continue
		}
		var img rawImageModel
		if err := json.Unmarshal(inner.Model, &img); err != nil {
			continue
		}
		return strings.TrimSpace(img.Locator)
	}
	return ""
}

func renderImageBlock(raw json.RawMessage) string {
	var model blockModel
	if err := json.Unmarshal(raw, &model); err != nil {
		return ""
	}
	for _, inner := range model.Blocks {
		if inner.Type != "rawImage" {
			continue
		}
		var img rawImageModel
		if err := json.Unmarshal(inner.Model, &img); err != nil {
			continue
		}
		src := ichefURL(img.OriginCode, img.Locator)
		if src == "" {
			continue
		}
		alt := escapeHTML(strings.TrimSpace(img.AltText))
		return fmt.Sprintf("<figure><img src=\"%s\" alt=\"%s\" style=\"max-width:100%%;\"/></figure>\n", src, alt)
	}
	return ""
}

func promoImageLocator(img imageBlock) string {
	for _, b := range img.Blocks {
		if b.Type != "rawImage" {
			continue
		}
		var model rawImageModel
		if err := json.Unmarshal(b.Model, &model); err != nil {
			continue
		}
		return strings.TrimSpace(model.Locator)
	}
	return ""
}

func promoImageURL(img imageBlock) string {
	for _, b := range img.Blocks {
		if b.Type != "rawImage" {
			continue
		}
		var model rawImageModel
		if err := json.Unmarshal(b.Model, &model); err != nil {
			continue
		}
		return ichefURL(model.OriginCode, model.Locator)
	}
	return ""
}

func ichefURL(originCode, locator string) string {
	locator = strings.TrimSpace(locator)
	if locator == "" {
		return ""
	}
	if strings.HasPrefix(locator, "http") {
		return normalizeImageURL(locator)
	}
	if originCode == "" {
		originCode = "cpsprodpb"
	}
	return fmt.Sprintf("https://ichef.bbci.co.uk/ace/ws/%s/%s/%s", imageWidth, originCode, locator)
}

func normalizeImageURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	return strings.ReplaceAll(raw, "{width}", imageWidth)
}

func articleIDFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		raw = u.Path
	}
	parts := strings.Split(strings.Trim(raw, "/"), "/")
	for i, part := range parts {
		if part == "articles" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func parsePublishedTime(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var ms int64
	if err := json.Unmarshal(raw, &ms); err == nil && ms > 0 {
		sec := ms
		if ms > 1_000_000_000_000 {
			sec = ms / 1000
		}
		return time.Unix(sec, 0).UTC().Format(time.RFC3339)
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil && strings.TrimSpace(s) != "" {
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, s); err == nil {
				return t.UTC().Format(time.RFC3339)
			}
		}
	}

	return ""
}

func parsePositiveInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return replacer.Replace(s)
}
