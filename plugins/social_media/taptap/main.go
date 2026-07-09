package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&TapTapPlugin{})
}

type TapTapPlugin struct{}

const (
	baseURL     = "https://www.taptap.cn"
	apiURL      = "https://www.taptap.cn/webapiv2/app-top/v2/hits"
	detailAPI   = "https://www.taptap.cn/webapiv2/app/v2/detail-by-id"
	pageSize    = 10
	defaultUA   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	defaultXUA  = "V=1&PN=WebApp&LANG=zh_CN&VN_CODE=102&LOC=CN&PLT=PC&DS=Android&UID=11111111-1111-4111-8111-111111111111&DT=PC"
	fallbackNow = "2026-01-01T00:00:00Z"
)

var rankingLabels = map[string]string{
	"hot":                  "热门榜",
	"reserve":              "预约榜",
	"pop":                  "热玩榜",
	"new":                  "新品榜",
	"sell":                 "热卖榜",
	"in_app_event_reserve": "新版本榜",
	"exclusive":            "独家榜",
	"action":               "动作榜",
	"strategy":             "策略榜",
	"idle":                 "放置榜",
	"single":               "单机榜",
	"casual":               "休闲榜",
	"sandbox_survival":     "沙盒生存榜",
	"management":           "模拟经营榜",
	"unriddle":             "解谜榜",
	"shooter":              "射击榜",
	"multiplayer":          "多人对战榜",
	"acgn":                 "二次元榜",
	"music":                "音乐节奏榜",
	"scenario":             "剧情榜",
	"swordsman":            "武侠榜",
	"otome":                "女性向榜",
	"independent":          "独立游戏榜",
	"roguelike":            "Roguelike榜",
}

type rankingResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

type rankingData struct {
	List []rankingEntry `json:"list"`
}

type rankingEntry struct {
	Identification string    `json:"identification"`
	Type           string    `json:"type"`
	App            taptapApp `json:"app"`
}

type taptapApp struct {
	ID            int         `json:"id"`
	Title         string      `json:"title"`
	Identifier    string      `json:"identifier"`
	ITunesID      string      `json:"itunes_id"`
	Icon          imageInfo   `json:"icon"`
	Banner        imageInfo   `json:"banner"`
	TopBanner     imageInfo   `json:"top_banner"`
	ReleasedTime  int64       `json:"released_time"`
	UpdateDate    string      `json:"update_date"`
	RecText       string      `json:"rec_text"`
	ReadableID    string      `json:"readable_id"`
	Description   textInfo    `json:"description"`
	WhatsNew      textInfo    `json:"whatsnew"`
	Stat          statInfo    `json:"stat"`
	Tags          []tagInfo   `json:"tags"`
	PlatformInfo  platformSet `json:"platform_info"`
	TitleLabelsV2 []labelInfo `json:"title_labels_v2"`
	Screenshots   []imageInfo `json:"screenshots"`
	Videos        []videoInfo `json:"videos"`
	Developers    []creator   `json:"developers"`
}

type imageInfo struct {
	URL       string `json:"url"`
	MediumURL string `json:"medium_url"`
	SmallURL  string `json:"small_url"`
	LargeURL  string `json:"large_url"`
}

type textInfo struct {
	Text string `json:"text"`
}

type detailResponse struct {
	Success bool      `json:"success"`
	Data    taptapApp `json:"data"`
}

type videoInfo struct {
	VideoID   int       `json:"video_id"`
	Thumbnail imageInfo `json:"thumbnail"`
	URL       string    `json:"url"`
	PlayURL   string    `json:"play_url"`
	HLSURL    string    `json:"hls_url"`
	M3U8URL   string    `json:"m3u8_url"`
}

type creator struct {
	Name string `json:"name"`
}

type statInfo struct {
	Rating      ratingInfo `json:"rating"`
	HitsTotal   int        `json:"hits_total"`
	PlayTotal   int        `json:"play_total"`
	Reserve     int        `json:"reserve_count"`
	ReviewCount int        `json:"review_count"`
	FansCount   int        `json:"fans_count"`
	WishCount   int        `json:"wish_count"`
}

type ratingInfo struct {
	Score string `json:"score"`
	Max   int    `json:"max"`
}

type tagInfo struct {
	Value string `json:"value"`
}

type labelInfo struct {
	Label string `json:"label"`
}

type platformSet struct {
	CurrentPlatform platformInfo   `json:"current_platform"`
	Supported       []platformInfo `json:"supported_platforms"`
}

type platformInfo struct {
	Key string `json:"key"`
}

func (p *TapTapPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch req.Route {
	case "/taptap/ranking":
		return fetchRanking(req.Params)
	case "/taptap/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id, req.Params["platform"])
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchRanking(params map[string]string) (*sdk.FeedResult, error) {
	rankingType := strings.TrimSpace(params["type"])
	if rankingType == "" {
		rankingType = "hot"
	}
	label := rankingLabels[rankingType]
	if label == "" {
		label = rankingType
	}

	page := parsePage(params["page"])
	platform := strings.TrimSpace(params["platform"])
	items, err := fetchRankingItems(rankingType, platform, page)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("empty ranking data")
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("TapTap - %s", label),
		Description: "TapTap 游戏排行榜数据",
		Items:       items,
	}
	if len(items) >= pageSize {
		result.HasMore = true
		result.Next = map[string]string{"page": strconv.Itoa(page + 1)}
	}
	return result, nil
}

func fetchRankingItems(rankingType, platform string, page int) ([]sdk.FeedItem, error) {
	from := (page - 1) * pageSize
	query := fmt.Sprintf("type_name=%s", rankingType)
	if platform != "" {
		query += "&platform=" + platform
	}
	if from > 0 {
		query += fmt.Sprintf("&from=%d&limit=%d", from, pageSize)
	}

	body, status, err := host.HTTPGet(apiURL+"?"+query, requestHeaders(platform))
	if err != nil {
		return nil, fmt.Errorf("fetch ranking failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("ranking http status %d", status)
	}
	return parseRanking(body, page)
}

func parseRanking(body []byte, page int) ([]sdk.FeedItem, error) {
	var response rankingResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	entries, err := parseEntries(response.Data)
	if err != nil {
		return nil, err
	}

	items := make([]sdk.FeedItem, 0, len(entries))
	for index, entry := range entries {
		app := entry.App
		if app.ID == 0 || strings.TrimSpace(app.Title) == "" {
			continue
		}

		rank := (page-1)*pageSize + index + 1
		cover := firstNonEmpty(app.Icon.URL, app.Icon.LargeURL, app.Icon.MediumURL, app.Icon.SmallURL)
		summary := buildSummary(rank, app)
		publishedAt := fallbackNow
		if app.ReleasedTime > 0 {
			publishedAt = time.Unix(app.ReleasedTime, 0).UTC().Format(time.RFC3339)
		}

		items = append(items, sdk.FeedItem{
			ID:          strconv.Itoa(app.ID),
			Title:       app.Title,
			URL:         fmt.Sprintf("%s/app/%d", baseURL, app.ID),
			Summary:     summary,
			Cover:       cover,
			Image:       cover,
			PublishedAt: publishedAt,
			Tags:        buildTags(rank, app),
		})
	}
	return items, nil
}

func parseEntries(raw json.RawMessage) ([]rankingEntry, error) {
	var entries []rankingEntry
	if err := json.Unmarshal(raw, &entries); err == nil {
		return entries, nil
	}

	var data rankingData
	if err := json.Unmarshal(raw, &data); err == nil {
		return data.List, nil
	}

	return nil, fmt.Errorf("unsupported ranking data shape")
}

func buildSummary(rank int, app taptapApp) string {
	parts := []string{fmt.Sprintf("#%d", rank)}
	if app.Stat.Rating.Score != "" {
		parts = append(parts, fmt.Sprintf("评分 %s/%d", app.Stat.Rating.Score, nonZero(app.Stat.Rating.Max, 10)))
	}
	if app.Stat.HitsTotal > 0 {
		parts = append(parts, fmt.Sprintf("热度 %d", app.Stat.HitsTotal))
	}
	if app.RecText != "" {
		parts = append(parts, html.UnescapeString(stripHTML(app.RecText)))
	} else if app.Description.Text != "" {
		parts = append(parts, html.UnescapeString(stripHTML(app.Description.Text)))
	}
	return strings.Join(parts, " · ")
}

func buildContent(rank int, app taptapApp, banner string) string {
	var builder strings.Builder
	if banner != "" {
		builder.WriteString(fmt.Sprintf("<img src=\"%s\" style=\"max-width: 100%%; border-radius: 8px; margin-bottom: 1rem;\"/>\n", html.EscapeString(banner)))
	}
	builder.WriteString(fmt.Sprintf("<p><strong>排名:</strong> #%d</p>\n", rank))
	if app.Stat.Rating.Score != "" {
		builder.WriteString(fmt.Sprintf("<p><strong>评分:</strong> %s/%d</p>\n", html.EscapeString(app.Stat.Rating.Score), nonZero(app.Stat.Rating.Max, 10)))
	}
	builder.WriteString(fmt.Sprintf("<p><strong>热度:</strong> %d</p>\n", app.Stat.HitsTotal))
	if app.Stat.FansCount > 0 {
		builder.WriteString(fmt.Sprintf("<p><strong>关注:</strong> %d</p>\n", app.Stat.FansCount))
	}
	if app.Stat.ReviewCount > 0 {
		builder.WriteString(fmt.Sprintf("<p><strong>评价:</strong> %d</p>\n", app.Stat.ReviewCount))
	}
	if app.Identifier != "" {
		builder.WriteString(fmt.Sprintf("<p><strong>包名:</strong> %s</p>\n", html.EscapeString(app.Identifier)))
	}
	if app.Description.Text != "" {
		builder.WriteString(fmt.Sprintf("<p>%s</p>\n", html.UnescapeString(app.Description.Text)))
	}
	return builder.String()
}

func fetchDetail(id, platform string) (*sdk.FeedResult, error) {
	body, status, err := host.HTTPGet(detailAPI+"/"+url.PathEscape(id), requestHeaders(platform))
	if err != nil {
		return nil, fmt.Errorf("fetch detail failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("detail http status %d", status)
	}

	var response detailResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse detail response: %w", err)
	}
	if !response.Success || response.Data.ID == 0 {
		return nil, fmt.Errorf("detail data not found")
	}

	app := response.Data
	cover := firstNonEmpty(app.Icon.URL, app.Icon.LargeURL, app.Icon.MediumURL, app.Icon.SmallURL)
	banner := firstNonEmpty(app.Banner.URL, app.Banner.LargeURL, cover)
	publishedAt := fallbackNow
	if app.UpdateDate != "" {
		if parsed, err := time.Parse("2006-01-02", app.UpdateDate); err == nil {
			publishedAt = parsed.UTC().Format(time.RFC3339)
		}
	}

	item := sdk.FeedItem{
		ID:          strconv.Itoa(app.ID),
		Title:       app.Title,
		URL:         fmt.Sprintf("%s/app/%d", baseURL, app.ID),
		Summary:     html.UnescapeString(stripHTML(app.Description.Text)),
		Content:     buildDetailContent(app, banner),
		Author:      firstDeveloper(app.Developers),
		Cover:       cover,
		Image:       firstScreenshot(app.Screenshots, cover),
		PublishedAt: publishedAt,
		Tags:        buildTags(0, app),
		Media:       buildDetailMedia(app),
	}

	return &sdk.FeedResult{
		Title:       app.Title,
		Description: "TapTap 游戏详情",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func buildDetailContent(app taptapApp, banner string) string {
	var builder strings.Builder
	if banner != "" {
		builder.WriteString(fmt.Sprintf("<img src=\"%s\" style=\"max-width: 100%%; border-radius: 8px; margin-bottom: 1rem;\"/>", html.EscapeString(banner)))
	}
	builder.WriteString(fmt.Sprintf("<h2>%s</h2>", html.EscapeString(app.Title)))
	if app.Stat.Rating.Score != "" {
		builder.WriteString(fmt.Sprintf("<p><strong>评分:</strong> %s/%d</p>", html.EscapeString(app.Stat.Rating.Score), nonZero(app.Stat.Rating.Max, 10)))
	}
	if app.Stat.HitsTotal > 0 {
		builder.WriteString(fmt.Sprintf("<p><strong>热度:</strong> %d</p>", app.Stat.HitsTotal))
	}
	if app.Stat.FansCount > 0 {
		builder.WriteString(fmt.Sprintf("<p><strong>关注:</strong> %d</p>", app.Stat.FansCount))
	}
	if app.Stat.ReviewCount > 0 {
		builder.WriteString(fmt.Sprintf("<p><strong>评价:</strong> %d</p>", app.Stat.ReviewCount))
	}
	if app.Identifier != "" {
		builder.WriteString(fmt.Sprintf("<p><strong>包名:</strong> %s</p>", html.EscapeString(app.Identifier)))
	}
	writeScreenshots(&builder, app.Screenshots)
	writeVideos(&builder, app.Videos)
	writeTextSection(&builder, "游戏介绍", app.Description.Text)
	writeTextSection(&builder, "更新说明", app.WhatsNew.Text)
	writeBottomScreenshots(&builder, app.Screenshots)
	return builder.String()
}

func writeVideos(builder *strings.Builder, videos []videoInfo) {
	if len(videos) == 0 {
		return
	}
	builder.WriteString("<h3>视频</h3>")
	for _, video := range videos {
		playURL := firstNonEmpty(video.M3U8URL, video.HLSURL, video.PlayURL, video.URL)
		thumb := firstNonEmpty(video.Thumbnail.URL, video.Thumbnail.LargeURL, video.Thumbnail.MediumURL)
		if playURL != "" {
			builder.WriteString(fmt.Sprintf("<video controls preload=\"metadata\" poster=\"%s\" style=\"max-width: 100%%; border-radius: 8px; margin: 0.5rem 0 1rem;\"><source src=\"%s\" type=\"application/vnd.apple.mpegurl\"/></video>", html.EscapeString(thumb), html.EscapeString(playURL)))
			continue
		}
		if thumb != "" {
			builder.WriteString(fmt.Sprintf("<img src=\"%s\" style=\"max-width: 100%%; border-radius: 8px; margin: 0.5rem 0 1rem;\"/>", html.EscapeString(thumb)))
		}
	}
}

func writeScreenshots(builder *strings.Builder, screenshots []imageInfo) {
	if len(screenshots) == 0 {
		return
	}
	builder.WriteString("<h3>游戏图片</h3><div style=\"display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 10px; margin: 0.5rem 0 1rem;\">")
	for _, shot := range screenshots {
		imageURL := firstNonEmpty(shot.URL, shot.LargeURL, shot.MediumURL, shot.SmallURL)
		if imageURL != "" {
			builder.WriteString(fmt.Sprintf("<img src=\"%s\" style=\"max-width: 100%%; border-radius: 8px; margin: 0.5rem 0;\"/>", html.EscapeString(imageURL)))
		}
	}
	builder.WriteString("</div>")
}

func writeBottomScreenshots(builder *strings.Builder, screenshots []imageInfo) {
	if len(screenshots) == 0 {
		return
	}
	builder.WriteString("<h3>更多游戏图片</h3>")
	for _, shot := range screenshots {
		imageURL := firstNonEmpty(shot.URL, shot.LargeURL, shot.MediumURL, shot.SmallURL)
		if imageURL == "" {
			continue
		}
		builder.WriteString(fmt.Sprintf("<p><img src=\"%s\" style=\"max-width: 100%%; width: 100%%; border-radius: 8px; display: block; margin: 0 0 12px;\"/></p>", html.EscapeString(imageURL)))
	}
}

func writeTextSection(builder *strings.Builder, title, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	builder.WriteString(fmt.Sprintf("<h3>%s</h3><div>%s</div>", html.EscapeString(title), normalizeTapHTML(value)))
}

func normalizeTapHTML(value string) string {
	replacer := strings.NewReplacer("<br class=\"bbcode-paragraph-br\"/>", "<br/>", "<br class=\"bbcode-paragraph-br\">", "<br/>")
	return html.UnescapeString(replacer.Replace(value))
}

func buildDetailMedia(app taptapApp) []sdk.SocialMedia {
	media := make([]sdk.SocialMedia, 0, len(app.Screenshots)+len(app.Videos))
	for _, shot := range app.Screenshots {
		imageURL := firstNonEmpty(shot.URL, shot.LargeURL, shot.MediumURL, shot.SmallURL)
		if imageURL != "" {
			media = append(media, sdk.SocialMedia{Type: "image", URL: imageURL, Thumbnail: firstNonEmpty(shot.SmallURL, shot.MediumURL)})
		}
	}
	return media
}

func firstDeveloper(developers []creator) string {
	for _, developer := range developers {
		if strings.TrimSpace(developer.Name) != "" {
			return strings.TrimSpace(developer.Name)
		}
	}
	return ""
}

func buildTags(rank int, app taptapApp) []string {
	tags := make([]string, 0, 1+len(app.TitleLabelsV2)+len(app.Tags))
	if rank > 0 {
		tags = append(tags, fmt.Sprintf("排名 #%d", rank))
	}
	if app.Stat.Rating.Score != "" {
		tags = append(tags, "评分 "+app.Stat.Rating.Score)
	}
	if app.PlatformInfo.CurrentPlatform.Key != "" {
		tags = append(tags, app.PlatformInfo.CurrentPlatform.Key)
	}
	for _, label := range app.TitleLabelsV2 {
		if label.Label != "" {
			tags = append(tags, label.Label)
		}
	}
	for _, tag := range app.Tags {
		if tag.Value != "" {
			tags = append(tags, tag.Value)
		}
	}
	return tags
}

func requestHeaders(platform string) map[string]string {
	xua := defaultXUA
	if strings.EqualFold(platform, "ios") {
		xua = strings.ReplaceAll(defaultXUA, "DS=Android", "DS=iOS")
	}
	return map[string]string{
		"Accept":     "application/json, text/plain, */*",
		"Referer":    baseURL + "/top/download",
		"User-Agent": defaultUA,
		"X-UA":       xua,
	}
}

func parsePage(value string) int {
	page, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstScreenshot(screenshots []imageInfo, fallback string) string {
	for _, shot := range screenshots {
		imageURL := firstNonEmpty(shot.URL, shot.LargeURL, shot.MediumURL, shot.SmallURL)
		if imageURL != "" {
			return imageURL
		}
	}
	return fallback
}

func stripHTML(value string) string {
	replacer := strings.NewReplacer("<br class=\"bbcode-paragraph-br\"/>", " ", "<br class=\"bbcode-paragraph-br\">", " ", "<br/>", " ", "<br>", " ", "<br />", " ", "\n", " ", "\t", " ")
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func nonZero(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}
