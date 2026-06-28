package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	apiBase = "https://apiv1.oschina.net/oschinapi"
)

func main() {
	sdk.Run(&OSChinaPlugin{})
}

type OSChinaPlugin struct{}

func (p *OSChinaPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch req.Route {
	case "/oschina/list":
		return fetchList(req.Params)
	case "/oschina/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type listResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  struct {
		Count int        `json:"count"`
		List  []listItem `json:"list"`
	} `json:"result"`
}

type listItem struct {
	ObjID      int64  `json:"objId"`
	ObjType    int    `json:"objType"`
	ObjTitle   string `json:"objTitle"`
	Detail     string `json:"detail"`
	Image      string `json:"image"`
	CreateTime string `json:"createTime"`
	UserVO     struct {
		Name string `json:"name"`
	} `json:"userVo"`
}

type detailResponse struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  struct {
		ID      int64  `json:"id"`
		Title   string `json:"title"`
		Detail  string `json:"detail"`
		PubTime string `json:"pubTime"`
		UserVO  struct {
			Name string `json:"name"`
		} `json:"userVo"`
		ShareImage string `json:"shareImage"`
	} `json:"result"`
}

func fetchList(params map[string]string) (*sdk.FeedResult, error) {
	typeStr := strings.TrimSpace(params["type"])
	if typeStr == "" {
		typeStr = "2"
	}

	pageNum := parsePositiveInt(params["pageNum"], 1)
	pageSize := parsePositiveInt(params["pageSize"], 15)

	listURL := fmt.Sprintf("%s/home/consult?pageNum=%d&pageSize=%d&type=%s",
		apiBase,
		pageNum,
		pageSize,
		url.QueryEscape(typeStr),
	)

	body, status, err := httpGetJSON(listURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode list response: %w", err)
	}
	if !resp.Success || resp.Code != 200 {
		msg := strings.TrimSpace(resp.Message)
		if msg == "" {
			msg = "request failed"
		}
		return nil, fmt.Errorf("oschina api error: %s", msg)
	}

	items := make([]sdk.FeedItem, 0, len(resp.Result.List))
	for _, it := range resp.Result.List {
		idStr := strconv.FormatInt(it.ObjID, 10)
		title := strings.TrimSpace(it.ObjTitle)
		if idStr == "0" || title == "" {
			continue
		}

		publishedAt := parseOSCTime(it.CreateTime)
		summary := strings.TrimSpace(it.Detail)
		if summary == "" {
			summary = title
		}

		items = append(items, sdk.FeedItem{
			ID:          idStr,
			Title:       title,
			URL:         oschinaNewsURL(idStr),
			Summary:     summary,
			Author:      strings.TrimSpace(it.UserVO.Name),
			Cover:       strings.TrimSpace(it.Image),
			Image:       strings.TrimSpace(it.Image),
			PublishedAt: publishedAt,
		})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no items found")
	}

	title := "开源资讯"
	if typeStr == "1" {
		title = "软件资讯"
	}

	result := &sdk.FeedResult{
		Title:       title,
		Description: "开源中国资讯列表",
		Items:       items,
	}

	total := resp.Result.Count
	if total <= 0 {
		total = pageNum*pageSize + 1
	}
	if pageNum*pageSize < total {
		result.HasMore = true
		result.Next = map[string]string{
			"pageNum":  strconv.Itoa(pageNum + 1),
			"pageSize": strconv.Itoa(pageSize),
			"type":     typeStr,
		}
	}

	return result, nil
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	detailURL := fmt.Sprintf("%s/new/detail?id=%s", apiBase, url.QueryEscape(id))

	body, status, err := httpGetJSON(detailURL)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d", status)
	}

	var resp detailResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode detail response: %w", err)
	}
	if !resp.Success || resp.Code != 200 {
		msg := strings.TrimSpace(resp.Message)
		if msg == "" {
			msg = "request failed"
		}
		return nil, fmt.Errorf("oschina api error: %s", msg)
	}

	title := strings.TrimSpace(resp.Result.Title)
	content := strings.TrimSpace(resp.Result.Detail)
	if title == "" {
		title = "正文"
	}
	if content == "" {
		return nil, fmt.Errorf("article content not found")
	}

	item := sdk.FeedItem{
		ID:          id,
		Title:       title,
		URL:         oschinaNewsURL(id),
		Content:     content,
		Author:      strings.TrimSpace(resp.Result.UserVO.Name),
		Cover:       strings.TrimSpace(resp.Result.ShareImage),
		Image:       strings.TrimSpace(resp.Result.ShareImage),
		PublishedAt: parseOSCTime(resp.Result.PubTime),
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "开源中国正文",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func parsePositiveInt(s string, fallback int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func parseOSCTime(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now().Format(time.RFC3339)
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		return time.Now().Format(time.RFC3339)
	}
	return t.Format(time.RFC3339)
}

func oschinaNewsURL(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "https://www.oschina.net/"
	}
	return "https://www.oschina.net/news/" + url.PathEscape(id)
}

func httpGetJSON(rawURL string) ([]byte, int, error) {
	body, status, err := host.HTTPGet(rawURL, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, 0, fmt.Errorf("http get failed: %w", err)
	}
	return body, status, nil
}

