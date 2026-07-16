package main

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL   = "https://bing.gifposter.com"
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var slugPattern = regexp.MustCompile(`/((?:column|wallpaper)-\d+-[^/]+)\.html`)

var channelLabels = map[string]string{
	"new-desc-classic": "最新壁纸",
	"new-asc-classic":  "最新壁纸（升序）",
	"new-desc-slide":   "最新壁纸（幻灯片）",
	"hot-desc-classic": "热门壁纸",
	"hot-asc-classic":  "热门壁纸（升序）",
	"hot-desc-slide":   "热门壁纸（幻灯片）",
		"phone-new":        "手机壁纸（最新）",
	"phone-hot":        "手机壁纸（热门）",
}

func main() {
	sdk.Run(&BingPlugin{})
}

type BingPlugin struct{}

func (p *BingPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/bing/list":
		return fetchList(req.Params, req.ChannelID)
	case req.Route == "/bing/phone":
		return fetchPhone(req.Params, req.ChannelID)
	case req.Route == "/bing/detail/:id":
		return fetchDetail(req.Params["id"])
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(params map[string]string, channelID string) (*sdk.FeedResult, error) {
	category := params["category"]
	if category == "" {
		category = "new"
	}
	sort := params["sort"]
	if sort == "" {
		sort = "desc"
	}
	layout := params["layout"]
	if layout == "" {
		layout = "classic"
	}
	page := parsePage(params["page"])

	listURL := fmt.Sprintf("%s/list/%s/%s/%s.html", baseURL, category, sort, layout)
	if page > 1 {
		listURL = fmt.Sprintf("%s?p=%d", listURL, page)
	}

	body, status, err := httpGet(listURL)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("list fetch failed: status %d", status)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse list html failed: %w", err)
	}

	var items []sdk.FeedItem
	if layout == "slide" {
		items = parseSlideItems(doc)
	} else {
		items = parseClassicItems(doc)
	}

	result := &sdk.FeedResult{
		Title:       listTitle(channelID, category, sort, layout),
		Description: "Bing daily wallpapers from Bing Wallpaper Gallery",
		Items:       items,
	}

	if hasNextPage(doc, page) {
		result.HasMore = true
		result.Next = map[string]string{
			"category": category,
			"sort":     sort,
			"layout":   layout,
			"page":     strconv.Itoa(page + 1),
		}
	}

	return result, nil
}

func fetchPhone(params map[string]string, channelID string) (*sdk.FeedResult, error) {
	order := params["order"]
	if order == "" {
		order = "new"
	}
	page := parsePage(params["page"])

	var phoneURL string
	switch order {
	case "hot":
		phoneURL = baseURL + "/phone/hot/desc.html"
	default:
		phoneURL = baseURL + "/phone.html"
	}
	if page > 1 {
		phoneURL = fmt.Sprintf("%s?page=%d", phoneURL, page)
	}

	body, status, err := httpGet(phoneURL)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("phone fetch failed: status %d", status)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse phone html failed: %w", err)
	}

	items := parsePhoneItems(doc)
	result := &sdk.FeedResult{
		Title:       channelLabels[channelID],
		Description: "Bing mobile wallpapers (608x1080)",
		Items:       items,
	}

	if hasPhoneNextPage(doc, page) {
		result.HasMore = true
		result.Next = map[string]string{
			"order": order,
			"page":  strconv.Itoa(page + 1),
		}
	}

	return result, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("missing wallpaper id")
	}

	detailURL := absoluteURL(id + ".html")

	body, status, err := httpGet(detailURL)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("detail fetch failed: status %d", status)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse detail html failed: %w", err)
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find(".title").First().Text())
	}
	if title == "" {
		title = strings.TrimSpace(doc.Find("meta[property='og:title']").AttrOr("content", "Bing Wallpaper"))
	}

	summary := strings.TrimSpace(doc.Find("article p").First().Text())
	if summary == "" {
		summary = strings.TrimSpace(doc.Find(".description").First().Text())
	}
	if summary == "" {
		summary = strings.TrimSpace(doc.Find("meta[property='og:description']").AttrOr("content", ""))
	}

	image := strings.TrimSpace(doc.Find("#bing_wallpaper").AttrOr("src", ""))
	if image == "" {
		image = strings.TrimSpace(doc.Find("img.mainimg").AttrOr("src", ""))
	}
	if image == "" {
		image = strings.TrimSpace(doc.Find("meta[property='og:image']").AttrOr("content", ""))
	}
	image = toDesktopImageURL(image)
	publishedAt := parsePublishedAt(strings.TrimSpace(doc.Find(".date time, time[itemprop='startTime']").First().Text()))

	item := sdk.FeedItem{
		ID:          id,
		Title:       html.UnescapeString(title),
		URL:         detailURL,
		Image:       image,
		Cover:       image,
		Summary:     html.UnescapeString(summary),
		PublishedAt: publishedAt,
		Content:     html.UnescapeString(summary),
	}

	return &sdk.FeedResult{
		Title: item.Title,
		Items: []sdk.FeedItem{item},
	}, nil
}

func parseClassicItems(doc *goquery.Document) []sdk.FeedItem {
	var items []sdk.FeedItem
	doc.Find("ul.imglist.lists li").Each(func(_ int, li *goquery.Selection) {
		item, ok := parseClassicItem(li)
		if ok {
			items = append(items, item)
		}
	})
	return items
}

func parseClassicItem(li *goquery.Selection) (sdk.FeedItem, bool) {
	link := li.Find("a").First()
	href, _ := link.Attr("href")
	id := extractSlug(href)
	if id == "" {
		return sdk.FeedItem{}, false
	}

	thumb, _ := link.Find("img").Attr("src")
	title := strings.TrimSpace(link.Find("img").AttrOr("alt", ""))
	if title == "" {
		title = strings.TrimSpace(li.Find("span").Last().Text())
	}

	dateText := strings.TrimSpace(li.Find("time").First().Text())
	summary := ""
	if views := strings.TrimSpace(li.Find(".icon-view").Parent().Text()); views != "" {
		summary = views + " views"
	}

	image := toDesktopImageURL(thumb)
	if image == "" {
		image = thumb
	}

	return sdk.FeedItem{
		ID:          id,
		Title:       html.UnescapeString(title),
		URL:         absoluteURL(href),
		Image:       image,
		Cover:       image,
		Summary:     summary,
		PublishedAt: parsePublishedAt(dateText),
	}, true
}

func parseSlideItems(doc *goquery.Document) []sdk.FeedItem {
	var items []sdk.FeedItem
	doc.Find(".imglist figure").Each(func(_ int, figure *goquery.Selection) {
		link := figure.Find("a[itemprop='contentUrl'], a").First()
		image, _ := link.Attr("href")
		title := strings.TrimSpace(link.Find("img").AttrOr("alt", ""))
		if title == "" {
			title = strings.TrimSpace(figure.Find("figcaption").Text())
		}
		dateText := strings.TrimSpace(link.Find("time").Text())

		detailHref, _ := figure.Find("a[href*='/column-'], a[href*='/wallpaper-']").Attr("href")
		id := extractSlug(detailHref)
		if id == "" {
			id = slugID(title, image)
		}
		itemURL := absoluteURL(detailHref)
		if itemURL == baseURL+"/" || itemURL == "" {
			itemURL = image
		}
		image = toDesktopImageURL(image)

		items = append(items, sdk.FeedItem{
			ID:          id,
			Title:       html.UnescapeString(title),
			URL:         itemURL,
			Image:       image,
			Cover:       image,
			PublishedAt: parsePublishedAt(dateText),
		})
	})
	return items
}

func parsePhoneItems(doc *goquery.Document) []sdk.FeedItem {
	var items []sdk.FeedItem
	doc.Find(".imglist figure").Each(func(_ int, figure *goquery.Selection) {
		link := figure.Find("a[itemprop='contentUrl'], a").First()
		image, _ := link.Attr("href")
		title := strings.TrimSpace(link.Find("img").AttrOr("alt", ""))
		if title == "" {
			title = strings.TrimSpace(figure.Find("figcaption").Text())
		}
		dateText := strings.TrimSpace(link.Find("time").Text())
		summary := strings.TrimSpace(figure.Find("figcaption").Text())

		detailHref, _ := figure.Find("a[href*='/wallpaper-']").Attr("href")
		id := extractSlug(detailHref)
		if id == "" {
			id = slugID(title, image)
		}

		itemURL := absoluteURL(detailHref)
		if itemURL == baseURL+"/" || itemURL == "" {
			itemURL = image
		}
		image = toDesktopImageURL(image)

		items = append(items, sdk.FeedItem{
			ID:          id,
			Title:       html.UnescapeString(title),
			URL:         itemURL,
			Image:       image,
			Cover:       image,
			Summary:     html.UnescapeString(summary),
			PublishedAt: parsePublishedAt(dateText),
		})
	})
	return items
}

func hasNextPage(doc *goquery.Document, page int) bool {
	nextPage := page + 1
	found := false
	doc.Find(".pagination a").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.Contains(href, fmt.Sprintf("p=%d", nextPage)) {
			found = true
		}
	})
	return found
}

func hasPhoneNextPage(doc *goquery.Document, page int) bool {
	nextPage := page + 1
	found := false
	doc.Find(".pagination a").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if strings.Contains(href, fmt.Sprintf("page=%d", nextPage)) {
			found = true
		}
	})
	return found
}

func listTitle(channelID, category, sort, layout string) string {
	if label := channelLabels[channelID]; label != "" {
		return "Bing · " + label
	}
	return fmt.Sprintf("Bing · %s/%s/%s", category, sort, layout)
}

func httpGet(rawURL string) ([]byte, int, error) {
	return host.HTTPGet(rawURL, map[string]string{
		"User-Agent": userAgent,
		"Referer":    baseURL + "/",
	})
}

func parsePage(page string) int {
	n, err := strconv.Atoi(strings.TrimSpace(page))
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func extractSlug(href string) string {
	match := slugPattern.FindStringSubmatch(href)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func slugID(title, image string) string {
	base := title
	if base == "" {
		base = image
	}
	base = strings.ToLower(base)
	base = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(base, "-")
	return strings.Trim(base, "-")
}

func absoluteURL(href string) string {
	if href == "" {
		return baseURL
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

func toDesktopImageURL(thumb string) string {
	thumb = strings.TrimSpace(thumb)
	if thumb == "" {
		return ""
	}
	for _, suffix := range []string{"_sm", "_mb"} {
		if strings.HasSuffix(thumb, suffix) {
			return strings.TrimSuffix(thumb, suffix)
		}
	}
	return thumb
}

func parsePublishedAt(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}

	formats := []string{
		"Jan 2, 2006",
		"January 2, 2006",
		"2006-01-02",
		"2006/01/02",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return time.Now().UTC().Format(time.RFC3339)
}
