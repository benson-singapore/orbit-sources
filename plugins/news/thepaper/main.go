package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL     = "https://www.thepaper.cn"
	listAPI     = "https://api.thepaper.cn/contentapi/nodeCont/getByNodeIdPortal"
	sidebarAPI  = "https://cache.thepaper.cn/contentapi/wwwIndex/rightSidebar"
	defaultUA   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var (
	reNextData  = regexp.MustCompile(`<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`)
	reContIDURL = regexp.MustCompile(`newsDetail_forward_(\d+)`)
	reContIDNum = regexp.MustCompile(`^\d+$`)
)

func main() {
	sdk.Run(&ThepaperPlugin{})
}

type ThepaperPlugin struct{}

var nodeLabelMap = map[string]string{
	"25462": "中国政库",
	"25422": "浦江头条",
	"27224": "澎湃评论",
	"25429": "澎湃国际",
	"25428": "直击现场",
	"25490": "打虎记",
}

var sidebarSectionMap = map[string]string{
	"hotNews":                  "澎湃热榜",
	"financialInformationNews": "澎湃财讯",
	"morningEveningNews":       "早晚报",
}

func (p *ThepaperPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/thepaper/list":
		return fetchList(req.Params)
	case req.Route == "/thepaper/sidebar":
		return fetchSidebar(req.Params)
	case req.Route == "/thepaper/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type listAPIResponse struct {
	Code int `json:"code"`
	Data struct {
		HasNext   bool           `json:"hasNext"`
		StartTime int64          `json:"startTime"`
		List      []contListItem `json:"list"`
		NodeInfo  *nodeInfo      `json:"nodeInfo"`
	} `json:"data"`
}

type sidebarAPIResponse struct {
	ResultCode int `json:"resultCode"`
	Data       struct {
		HotNews                  []contListItem `json:"hotNews"`
		FinancialInformationNews []contListItem `json:"financialInformationNews"`
		MorningEveningNews       []contListItem `json:"morningEveningNews"`
	} `json:"data"`
}

type contListItem struct {
	ContID      string    `json:"contId"`
	Name        string    `json:"name"`
	Link        string    `json:"link"`
	Pic         string    `json:"pic"`
	SharePic    string    `json:"sharePic"`
	PubTimeLong int64     `json:"pubTimeLong"`
	Summary     string    `json:"summary"`
	Author      string    `json:"author"`
	NodeInfo    *nodeInfo `json:"nodeInfo"`
}

type nodeInfo struct {
	NodeID int    `json:"nodeId"`
	Name   string `json:"name"`
}

type nextPageData struct {
	Props struct {
		PageProps struct {
			DetailData struct {
				ContentDetail *contentDetail `json:"contentDetail"`
				LiveDetail    *contentDetail `json:"liveDetail"`
				SpecialDetail *struct {
					SpecialInfo *contentDetail `json:"specialInfo"`
				} `json:"specialDetail"`
			} `json:"detailData"`
		} `json:"pageProps"`
	} `json:"props"`
}

type contentDetail struct {
	ContID      flexString `json:"contId"`
	Name        string     `json:"name"`
	ShareName   string     `json:"shareName"`
	Summary     string     `json:"summary"`
	Desc        string     `json:"desc"`
	Content     string     `json:"content"`
	Author      string     `json:"author"`
	PubTime     string     `json:"pubTime"`
	PublishTime int64      `json:"publishTime"`
	Pic         string     `json:"pic"`
	BigPic      string     `json:"bigPic"`
	SharePic    string     `json:"sharePic"`
	TagList     []struct {
		Tag string `json:"tag"`
	} `json:"tagList"`
	NodeInfo *nodeInfo `json:"nodeInfo"`
}

type flexString string

func (s *flexString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = ""
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*s = flexString(str)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*s = flexString(n.String())
	return nil
}

func fetchList(params map[string]string) (*sdk.FeedResult, error) {
	nodeID := strings.TrimSpace(params["node_id"])
	if nodeID == "" {
		return nil, fmt.Errorf("missing node_id parameter")
	}

	payload := map[string]interface{}{
		"nodeId": nodeID,
	}
	if startTime := strings.TrimSpace(params["startTime"]); startTime != "" {
		if ts, err := strconv.ParseInt(startTime, 10, 64); err == nil && ts > 0 {
			payload["startTime"] = ts
		}
	}

	body, status, err := apiPost(listAPI, payload)
	if err != nil {
		return nil, fmt.Errorf("fetch list failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("list http status %d", status)
	}

	var resp listAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("list api code %d", resp.Code)
	}

	items := listItemsToFeed(resp.Data.List)
	if len(items) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	label := nodeLabelMap[nodeID]
	if label == "" && resp.Data.NodeInfo != nil {
		label = strings.TrimSpace(resp.Data.NodeInfo.Name)
	}
	if label == "" {
		label = nodeID
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("澎湃新闻 · %s", label),
		Description: "澎湃新闻栏目资讯",
		Items:       items,
	}

	if resp.Data.HasNext && resp.Data.StartTime > 0 {
		result.HasMore = true
		result.Next = map[string]string{
			"node_id":   nodeID,
			"startTime": strconv.FormatInt(resp.Data.StartTime, 10),
		}
	}

	return result, nil
}

func fetchSidebar(params map[string]string) (*sdk.FeedResult, error) {
	section := strings.TrimSpace(params["section"])
	if section == "" {
		section = "hotNews"
	}

	body, status, err := host.HTTPGet(sidebarAPI, cacheHeaders())
	if err != nil {
		return nil, fmt.Errorf("fetch sidebar failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("sidebar http status %d", status)
	}

	var resp sidebarAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse sidebar response: %w", err)
	}
	if resp.ResultCode != 1 {
		return nil, fmt.Errorf("sidebar api resultCode %d", resp.ResultCode)
	}

	var rawList []contListItem
	switch section {
	case "hotNews":
		rawList = resp.Data.HotNews
	case "financialInformationNews":
		rawList = resp.Data.FinancialInformationNews
	case "morningEveningNews":
		rawList = resp.Data.MorningEveningNews
	default:
		return nil, fmt.Errorf("unknown section: %s", section)
	}

	items := listItemsToFeed(rawList)
	if len(items) == 0 {
		return nil, fmt.Errorf("no articles found")
	}

	label := sidebarSectionMap[section]
	if label == "" {
		label = section
	}

	return &sdk.FeedResult{
		Title:       fmt.Sprintf("澎湃新闻 · %s", label),
		Description: "澎湃新闻首页推荐",
		Items:       items,
	}, nil
}

func fetchDetail(rawID string) (*sdk.FeedResult, error) {
	contID := extractContID(rawID)
	if contID == "" {
		return nil, fmt.Errorf("invalid article id")
	}

	articleURL := articleURL(contID)
	body, status, err := host.HTTPGet(articleURL, pageHeaders())
	if err != nil {
		return nil, fmt.Errorf("fetch article failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("article http status %d", status)
	}

	detail, err := parseDetailFromHTML(string(body))
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(detail.Name)
	if title == "" {
		title = strings.TrimSpace(detail.ShareName)
	}
	if title == "" {
		return nil, fmt.Errorf("no title found")
	}

	cover := firstNonEmpty(detail.Pic, detail.BigPic, detail.SharePic)
	summary := strings.TrimSpace(detail.Summary)
	if summary == "" {
		summary = strings.TrimSpace(detail.Desc)
	}
	contentHTML := strings.TrimSpace(detail.Content)
	if contentHTML == "" {
		contentHTML = summary
	}

	item := sdk.FeedItem{
		ID:          contID,
		Title:       title,
		URL:         articleURL,
		Summary:     summary,
		Author:      strings.TrimSpace(detail.Author),
		Cover:       cover,
		Image:       cover,
		Content:     buildArticleContent(cover, summary, contentHTML),
		PublishedAt: formatPublishedAt(detail.PublishTime, detail.PubTime),
	}

	for _, t := range detail.TagList {
		tag := strings.TrimSpace(t.Tag)
		if tag != "" {
			item.Tags = append(item.Tags, tag)
		}
	}
	if detail.NodeInfo != nil && strings.TrimSpace(detail.NodeInfo.Name) != "" {
		item.Tags = append(item.Tags, detail.NodeInfo.Name)
	}

	return &sdk.FeedResult{
		Title:       item.Title,
		Description: item.Summary,
		Items:       []sdk.FeedItem{item},
	}, nil
}

func listItemsToFeed(rawList []contListItem) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, len(rawList))
	for _, entry := range rawList {
		item := listItemToFeedItem(entry)
		if item.Title == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func listItemToFeedItem(entry contListItem) sdk.FeedItem {
	title := strings.TrimSpace(entry.Name)
	link := strings.TrimSpace(entry.Link)

	var itemURL string
	var itemID string

	if link != "" && strings.HasPrefix(link, "http") {
		itemURL = link
		itemID = link
	} else {
		contID := strings.TrimSpace(entry.ContID)
		if contID == "" {
			return sdk.FeedItem{}
		}
		itemID = contID
		itemURL = articleURL(contID)
	}

	cover := firstNonEmpty(entry.Pic, entry.SharePic)
	item := sdk.FeedItem{
		ID:          itemID,
		Title:       title,
		URL:         itemURL,
		Summary:     strings.TrimSpace(entry.Summary),
		Author:      strings.TrimSpace(entry.Author),
		Cover:       cover,
		Image:       cover,
		PublishedAt: formatPublishedAt(entry.PubTimeLong, ""),
	}

	if entry.NodeInfo != nil && strings.TrimSpace(entry.NodeInfo.Name) != "" {
		item.Tags = append(item.Tags, entry.NodeInfo.Name)
	}

	return item
}

func parseDetailFromHTML(html string) (*contentDetail, error) {
	match := reNextData.FindStringSubmatch(html)
	if len(match) < 2 {
		return nil, fmt.Errorf("no __NEXT_DATA__ found")
	}

	var pageData nextPageData
	if err := json.Unmarshal([]byte(match[1]), &pageData); err != nil {
		return nil, fmt.Errorf("parse __NEXT_DATA__: %w", err)
	}

	detailData := pageData.Props.PageProps.DetailData
	detail := detailData.ContentDetail
	if detail == nil {
		detail = detailData.LiveDetail
	}
	if detail == nil && detailData.SpecialDetail != nil {
		detail = detailData.SpecialDetail.SpecialInfo
	}
	if detail == nil {
		return nil, fmt.Errorf("no article detail found")
	}

	return detail, nil
}

func extractContID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if reContIDNum.MatchString(raw) {
		return raw
	}
	if matches := reContIDURL.FindStringSubmatch(raw); len(matches) > 1 {
		return matches[1]
	}
	if strings.Contains(raw, "/detail/") {
		parts := strings.Split(strings.TrimSuffix(raw, "/"), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
}

func articleURL(contID string) string {
	return fmt.Sprintf("%s/newsDetail_forward_%s", baseURL, contID)
}

func buildArticleContent(cover, summary, content string) string {
	var sb strings.Builder
	if cover != "" {
		sb.WriteString(fmt.Sprintf(`<img src="%s" style="max-width:100%%;border-radius:8px;margin-bottom:1rem;"/>`, cover))
		sb.WriteString("\n")
	}
	if summary != "" {
		sb.WriteString(fmt.Sprintf("<p><strong>%s</strong></p>\n", summary))
	}
	if content != "" {
		sb.WriteString(content)
	}
	return sb.String()
}

func formatPublishedAt(pubTimeLong int64, pubTime string) string {
	if pubTimeLong > 0 {
		ms := pubTimeLong
		if ms > 1_000_000_000_000 {
			return time.UnixMilli(ms).Format(time.RFC3339)
		}
		return time.Unix(ms, 0).Format(time.RFC3339)
	}

	pubTime = strings.TrimSpace(pubTime)
	if pubTime == "" {
		return time.Now().Format(time.RFC3339)
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.ParseInLocation(format, pubTime, time.Local); err == nil {
			return t.Format(time.RFC3339)
		}
	}

	return time.Now().Format(time.RFC3339)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func apiPost(url string, payload interface{}) ([]byte, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	return host.HTTPPost(url, apiHeaders(), string(body))
}

func apiHeaders() map[string]string {
	return map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
		"Origin":       baseURL,
		"Referer":      baseURL + "/",
		"User-Agent":   defaultUA,
	}
}

func cacheHeaders() map[string]string {
	return map[string]string{
		"Accept":     "application/json",
		"Referer":    baseURL + "/",
		"User-Agent": defaultUA,
	}
}

func pageHeaders() map[string]string {
	return map[string]string{
		"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Referer":    baseURL + "/",
		"User-Agent": defaultUA,
	}
}
