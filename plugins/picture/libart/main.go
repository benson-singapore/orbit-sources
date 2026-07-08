package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

func main() {
	sdk.Run(&LibArtPlugin{})
}

type LibArtPlugin struct{}

const apiBaseURL = "https://api2.liblib.art/api/www/img/group/search"

func (p *LibArtPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/libart/search":
		tagIDStr := req.Params["tagV2Ids"]
		if tagIDStr == "" {
			tagIDStr = "550005"
		}
		page := req.Params["page"]
		if page == "" {
			page = "1"
		}
		return fetchSearch(tagIDStr, page)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchSearch(tagIDStr string, pageStr string) (*sdk.FeedResult, error) {
	pageNum, _ := strconv.Atoi(pageStr)
	if pageNum < 1 {
		pageNum = 1
	}

	var tagIDs []int
	if tagIDStr != "" {
		parts := strings.Split(tagIDStr, ",")
		for _, p := range parts {
			if id, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
	}

	if len(tagIDs) == 0 {
		tagIDs = []int{550005}
	}

	cid := fmt.Sprintf("1783506326162uxclgmiy")
	requestID := fmt.Sprintf("req-%d-%d", time.Now().Unix(), pageNum)

	payload := map[string]interface{}{
		"cid":          cid,
		"requestId":    requestID,
		"page":         pageNum,
		"pageSize":     30,
		"sort":         0,
		"followed":     0,
		"resolution":   0,
		"types":        []int{},
		"imageFuncs":   []int{},
		"createTools":  []int{},
		"imageSources": []int{},
		"clientType":   "pc",
		"tagV2Ids":     tagIDs,
		"liked":        0,
		"imageCapability": []int{},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	url := fmt.Sprintf("%s?timestamp=%s", apiBaseURL, timestamp)

	resp, status, err := host.HTTPPost(url, map[string]string{
		"accept":         "application/json, text/plain, */*",
		"accept-language": "zh,en;q=0.9,zh-CN;q=0.8",
		"cache-control":  "no-cache",
		"content-type":   "application/json",
		"origin":         "https://www.liblib.art",
		"pragma":         "no-cache",
		"referer":        "https://www.liblib.art/inspiration",
		"user-agent":     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
	}, string(body))

	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var apiResp struct {
		Code int `json:"code"`
		Data struct {
			Page     int `json:"page"`
			Total    int `json:"total"`
			PageSize int `json:"pageSize"`
			Data     []struct {
				UUID      string `json:"uuid"`
				Title     string `json:"title"`
				ImageURL  string `json:"imageUrl"`
				WebpURL   string `json:"webpUrl"`
				Nickname  string `json:"nickname"`
				LikeCount int    `json:"likeCount"`
				CreateTime string `json:"createTime"`
				Width     int    `json:"width"`
				Height    int    `json:"height"`
			} `json:"data"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("api error: code %d", apiResp.Code)
	}

	var items []sdk.FeedItem
	tagLabel := getTagLabel(tagIDs[0])

	for _, img := range apiResp.Data.Data {
		if img.UUID == "" || img.Title == "" {
			continue
		}

		imgURL := img.ImageURL
		if imgURL == "" {
			imgURL = img.WebpURL
		}
		if imgURL == "" {
			continue
		}

		pubAt := parseTime(img.CreateTime)

		items = append(items, sdk.FeedItem{
			ID:          img.UUID,
			Title:       img.Title,
			URL:         fmt.Sprintf("https://www.liblib.art/inspiration?id=%s", img.UUID),
			Cover:       imgURL,
			Image:       imgURL,
			Author:      img.Nickname,
			PublishedAt: pubAt,
			Summary:     fmt.Sprintf("❤️ %d", img.LikeCount),
		})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no images found")
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("Liblib - %s", tagLabel),
		Description: "AI创意图库",
		Items:       items,
	}

	if len(items) >= 30 {
		result.HasMore = true
		result.Next = map[string]string{
			"page":      strconv.Itoa(pageNum + 1),
			"tagV2Ids":  tagIDStr,
		}
	}

	return result, nil
}

func getTagLabel(tagID int) string {
	labels := map[int]string{
		550005: "摄影写真",
		560045: "风格插画",
		560024: "平面设计",
		560032: "动漫游戏",
		600018: "电商产品",
		600020: "短剧漫剧",
		560012: "电商营销",
		540024: "建筑室内",
		560061: "创意玩法",
		560019: "文创周边",
		560035: "小说推文",
	}
	if label, ok := labels[tagID]; ok {
		return label
	}
	return fmt.Sprintf("分类 %d", tagID)
}

func parseTime(timeStr string) string {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format(time.RFC3339)
		}
	}

	return time.Now().Format(time.RFC3339)
}
