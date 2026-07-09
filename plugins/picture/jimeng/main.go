package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	baseURL     = "https://jimeng.jianying.com"
	explorePath = "/mweb/v1/get_explore"

	appID       = "513695"
	appVersion  = "8.4.0"
	platform    = "7"
	webVersion  = "7.5.0"
	daVersion   = "3.3.9"
	apiPageSize = 40 // 官方固定每页 40，next_offset = offset + 40
)

var channelLabels = map[string]string{
	"trending":       "即梦 · 热门趋势",
	"discover-image": "即梦 · 发现图片",
}

var coverSizePrefs = []string{"2400", "1080", "720", "480", "360"}

func main() {
	sdk.Run(&JimengPlugin{})
}

type JimengPlugin struct{}

func (p *JimengPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/jimeng/explore" || strings.HasPrefix(req.Route, "/jimeng/explore"):
		return fetchExplore(req)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type exploreResponse struct {
	Ret    string `json:"ret"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		HasMore    bool          `json:"has_more"`
		NextOffset int           `json:"next_offset"`
		ItemList   []exploreItem `json:"item_list"`
	} `json:"data"`
}

type exploreItem struct {
	CommonAttr struct {
		ID          string            `json:"id"`
		Title       string            `json:"title"`
		Description string            `json:"description"`
		CoverURL    string            `json:"cover_url"`
		CoverURLMap map[string]string `json:"cover_url_map"`
		CreateTime  int64             `json:"create_time"`
		EffectType  int               `json:"effect_type"`
	} `json:"common_attr"`
	Author struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	} `json:"author"`
	Image struct {
		LargeImages []struct {
			ImageURL string `json:"image_url"`
			Width    int    `json:"width"`
			Height   int    `json:"height"`
		} `json:"large_images"`
	} `json:"image"`
	AIGCImageParams struct {
		Text2ImageParams struct {
			Prompt string `json:"prompt"`
		} `json:"text2image_params"`
	} `json:"aigc_image_params"`
	Statistic struct {
		FavoriteNum int `json:"favorite_num"`
		UsageNum    int `json:"usage_num"`
	} `json:"statistic"`
}

func fetchExplore(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	categoryID := strings.TrimSpace(req.Params["categoryId"])
	if categoryID == "" {
		categoryID = "11222"
	}
	page := parseInt(req.Params["page"], 1)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * apiPageSize
	seenIDs := parseIDList(req.Params["seenIds"])
	if page == 1 {
		seenIDs = nil
	}
	seenSet := make(map[string]bool, len(seenIDs))
	for _, id := range seenIDs {
		seenSet[id] = true
	}

	feedRefer := "feed_loadmore"
	if page == 1 && len(seenIDs) == 0 {
		feedRefer = "feed_refresh"
	}

	workTypes := parseWorkTypes(req.Params["workType"])
	body := map[string]any{
		"count":       apiPageSize,
		"offset":      offset,
		"category_id": parseInt(categoryID, 11222),
		"filter": map[string]any{
			"work_type_list": workTypes,
		},
		"image_info": exploreImageInfo(),
		"feed_refer": feedRefer,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	apiURL := buildExploreURL()
	headers := buildHeaders(explorePath)

	respBody, status, err := host.HTTPPost(apiURL, headers, string(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("fetch explore failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("explore http status %d: %s", status, truncate(string(respBody), 200))
	}
	if len(respBody) == 0 {
		return nil, fmt.Errorf("explore returned empty body (check sign or network)")
	}

	var resp exploreResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parse explore response: %w", err)
	}
	if resp.Ret != "" && resp.Ret != "0" {
		return nil, fmt.Errorf("explore api error ret=%s errmsg=%s", resp.Ret, resp.Errmsg)
	}

	allSeen := append([]string{}, seenIDs...)
	items := make([]sdk.FeedItem, 0, len(resp.Data.ItemList))
	for _, raw := range resp.Data.ItemList {
		item, ok := itemToFeed(raw)
		if !ok || seenSet[item.ID] {
			continue
		}
		seenSet[item.ID] = true
		allSeen = append(allSeen, item.ID)
		items = append(items, item)
	}

	title := channelLabels[req.ChannelID]
	if title == "" {
		title = "即梦 AI"
	}

	hasMore := resp.Data.HasMore && len(items) > 0
	var next map[string]string
	if hasMore {
		next = copyParams(req.Params)
		next["page"] = strconv.Itoa(page + 1)
		next["seenIds"] = joinIDList(allSeen)
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "即梦发现页热门 AI 图片与生成提示词",
		Items:       items,
		HasMore:     hasMore,
		Next:        next,
	}, nil
}

func itemToFeed(raw exploreItem) (sdk.FeedItem, bool) {
	id := strings.TrimSpace(raw.CommonAttr.ID)
	if id == "" {
		return sdk.FeedItem{}, false
	}

	prompt := strings.TrimSpace(raw.AIGCImageParams.Text2ImageParams.Prompt)
	if prompt == "" {
		prompt = strings.TrimSpace(raw.CommonAttr.Title)
	}
	if prompt == "" {
		prompt = strings.TrimSpace(raw.CommonAttr.Description)
	}

	imageURL := pickImageURL(raw)
	if imageURL == "" {
		return sdk.FeedItem{}, false
	}

	title := promptTitle(prompt, raw.CommonAttr.Title)
	author := strings.TrimSpace(raw.Author.Name)
	publishedAt := time.Unix(raw.CommonAttr.CreateTime, 0).UTC().Format(time.RFC3339)
	if raw.CommonAttr.CreateTime <= 0 {
		publishedAt = time.Now().UTC().Format(time.RFC3339)
	}

	summary := buildSummary(prompt, author, raw.Statistic.FavoriteNum, raw.Statistic.UsageNum)

	return sdk.FeedItem{
		ID:           id,
		Title:        title,
		URL:          workURL(id),
		Summary:      summary,
		Content:      buildContent(imageURL, prompt),
		Author:       author,
		AuthorAvatar: strings.TrimSpace(raw.Author.AvatarURL),
		Cover:        imageURL,
		Image:        imageURL,
		PublishedAt:  publishedAt,
	}, true
}

func buildExploreURL() string {
	values := url.Values{
		"aid":                     {appID},
		"device_platform":         {"web"},
		"region":                  {"cn"},
		"da_version":              {daVersion},
		"os":                      {"windows"},
		"web_component_open_flag": {"1"},
		"web_version":             {webVersion},
		"aigc_features":           {"app_lip_sync"},
	}
	return baseURL + explorePath + "?" + values.Encode()
}

func buildHeaders(uri string) map[string]string {
	deviceTime := host.NowUnix()
	sign := computeSign(uri, deviceTime)
	return map[string]string{
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "zh-CN,zh;q=0.9",
		"Content-Type":    "application/json",
		"Appvr":           appVersion,
		"Pf":              platform,
		"Appid":           appID,
		"App-Sdk-Version": "48.0.0",
		"Device-Time":     strconv.FormatInt(deviceTime, 10),
		"Lan":             "zh-Hans",
		"Loc":             "cn",
		"Origin":          baseURL,
		"Referer":         baseURL + "/ai-tool/home/?type=image",
		"Sign":            sign,
		"Sign-Ver":        "1",
		"Tdid":            "",
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36",
	}
}

func computeSign(uri string, deviceTime int64) string {
	suffix := uri
	if len(suffix) > 7 {
		suffix = suffix[len(suffix)-7:]
	}
	payload := fmt.Sprintf("9e2c|%s|%s|%s|%d||11ac", suffix, platform, appVersion, deviceTime)
	sum := md5.Sum([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func pickImageURL(raw exploreItem) string {
	if len(raw.Image.LargeImages) > 0 {
		if u := strings.TrimSpace(raw.Image.LargeImages[0].ImageURL); u != "" {
			return u
		}
	}
	for _, size := range coverSizePrefs {
		if u := strings.TrimSpace(raw.CommonAttr.CoverURLMap[size]); u != "" {
			return u
		}
	}
	return strings.TrimSpace(raw.CommonAttr.CoverURL)
}

func promptTitle(prompt, fallback string) string {
	if t := strings.TrimSpace(fallback); t != "" {
		return truncateRunes(t, 80)
	}
	if prompt == "" {
		return "即梦 AI 作品"
	}
	line := prompt
	if idx := strings.IndexAny(prompt, "\n\r。！？"); idx > 0 {
		line = prompt[:idx]
	}
	return truncateRunes(strings.TrimSpace(line), 80)
}

func buildSummary(prompt, author string, favorites, usage int) string {
	parts := make([]string, 0, 3)
	if prompt != "" {
		parts = append(parts, truncateRunes(prompt, 120))
	}
	if author != "" {
		parts = append(parts, author)
	}
	if favorites > 0 || usage > 0 {
		parts = append(parts, fmt.Sprintf("收藏 %d · 使用 %d", favorites, usage))
	}
	return strings.Join(parts, " · ")
}

func buildContent(imageURL, prompt string) string {
	if imageURL == "" && prompt == "" {
		return ""
	}

	var b strings.Builder
	if imageURL != "" {
		alt := prompt
		if alt == "" {
			alt = "即梦 AI"
		}
		b.WriteString(`<figure><img src="`)
		b.WriteString(html.EscapeString(imageURL))
		b.WriteString(`" alt="`)
		b.WriteString(html.EscapeString(truncateRunes(alt, 120)))
		b.WriteString(`"/></figure>`)
	}
	if prompt != "" {
		b.WriteString("<p>")
		b.WriteString(html.EscapeString(prompt))
		b.WriteString("</p>")
	}
	return b.String()
}

func workURL(id string) string {
	return baseURL + "/ai-tool/home/?type=image&workId=" + url.QueryEscape(id)
}

// exploreImageInfo matches the official web client payload for cover variants.
func exploreImageInfo() map[string]any {
	return map[string]any{
		"width":  2048,
		"height": 2048,
		"format": "webp",
		"image_scene_list": []map[string]any{
			{"scene": "smart_crop", "width": 360, "height": 360, "format": "webp", "uniq_key": "smart_crop-w:360-h:360"},
			{"scene": "smart_crop", "width": 480, "height": 480, "format": "webp", "uniq_key": "smart_crop-w:480-h:480"},
			{"scene": "smart_crop", "width": 720, "height": 720, "format": "webp", "uniq_key": "smart_crop-w:720-h:720"},
			{"scene": "smart_crop", "width": 720, "height": 480, "format": "webp", "uniq_key": "smart_crop-w:720-h:480"},
			{"scene": "smart_crop", "width": 360, "height": 240, "format": "webp", "uniq_key": "smart_crop-w:360-h:240"},
			{"scene": "smart_crop", "width": 240, "height": 320, "format": "webp", "uniq_key": "smart_crop-w:240-h:320"},
			{"scene": "smart_crop", "width": 480, "height": 640, "format": "webp", "uniq_key": "smart_crop-w:480-h:640"},
			{"scene": "loss", "width": 1080, "height": 1080, "format": "webp", "uniq_key": "1080"},
			{"scene": "loss", "width": 900, "height": 900, "format": "webp", "uniq_key": "900"},
			{"scene": "loss", "width": 720, "height": 720, "format": "webp", "uniq_key": "720"},
			{"scene": "loss", "width": 480, "height": 480, "format": "webp", "uniq_key": "480"},
			{"scene": "loss", "width": 360, "height": 360, "format": "webp", "uniq_key": "360"},
			{"scene": "normal", "width": 2048, "height": 2048, "format": "webp", "uniq_key": "2048"},
		},
	}
}

func parseWorkTypes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{"video", "image", "canvas"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{"video", "image", "canvas"}
	}
	return out
}

func parseIDList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

func joinIDList(ids []string) string {
	return strings.Join(ids, ",")
}

func copyParams(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for key, value := range params {
		out[key] = value
	}
	return out
}

func parseInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
