package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	defaultAPIURL   = "https://cj.rycjapi.com/api.php/provide/vod"
	defaultSiteName = "TV-如意资源"
)

var reStripHTML = regexp.MustCompile(`(?s)<[^>]+>`)

func main() {
	sdk.Run(&RycjAPIPlugin{})
}

type RycjAPIPlugin struct{}

var channelLabels = map[string]string{
	"latest": "最新更新",
	"search": "搜索",
}

// typeLabels maps CMS secondary category IDs (type_id) to display titles.
var typeLabels = map[string]string{
	"6":  "电影 · 动作片",
	"7":  "电影 · 喜剧片",
	"8":  "电影 · 爱情片",
	"9":  "电影 · 科幻片",
	"10": "电影 · 恐怖片",
	"11": "电影 · 剧情片",
	"12": "电影 · 战争片",
	"20": "电影 · 记录片",
	"34": "电影 · 伦理片",
	"45": "电影 · 预告片",
	"47": "电影 · 动画电影",
	"13": "剧集 · 国产剧",
	"14": "剧集 · 香港剧",
	"15": "剧集 · 韩国剧",
	"16": "剧集 · 欧美剧",
	"21": "剧集 · 台湾剧",
	"22": "剧集 · 日本剧",
	"23": "剧集 · 海外剧",
	"24": "剧集 · 泰国剧",
	"46": "剧集 · 短剧",
	"25": "综艺 · 大陆综艺",
	"26": "综艺 · 港台综艺",
	"27": "综艺 · 日韩综艺",
	"28": "综艺 · 欧美综艺",
	"29": "动漫 · 国产动漫",
	"30": "动漫 · 日韩动漫",
	"31": "动漫 · 欧美动漫",
	"32": "动漫 · 港台动漫",
	"33": "动漫 · 海外动漫",
	"37": "体育 · 足球",
	"38": "体育 · 篮球",
	"39": "体育 · 网球",
	"40": "体育 · 斯诺克",
}

type apiResponse struct {
	Code      int       `json:"code"`
	Msg       string    `json:"msg"`
	Page      int       `json:"page"`
	PageCount int       `json:"pagecount"`
	Limit     int       `json:"limit"`
	Total     int       `json:"total"`
	List      []vodItem `json:"list"`
}

type vodItem struct {
	VodID          int    `json:"vod_id"`
	VodName        string `json:"vod_name"`
	VodSub         string `json:"vod_sub"`
	TypeID         int    `json:"type_id"`
	TypeName       string `json:"type_name"`
	VodRemarks     string `json:"vod_remarks"`
	VodTime        string `json:"vod_time"`
	VodPic         string `json:"vod_pic"`
	VodActor       string `json:"vod_actor"`
	VodDirector    string `json:"vod_director"`
	VodWriter      string `json:"vod_writer"`
	VodArea        string `json:"vod_area"`
	VodLang        string `json:"vod_lang"`
	VodYear        string `json:"vod_year"`
	VodScore       string `json:"vod_score"`
	VodDoubanScore string `json:"vod_douban_score"`
	VodPubdate     string `json:"vod_pubdate"`
	VodClass       string `json:"vod_class"`
	VodBlurb       string `json:"vod_blurb"`
	VodContent     string `json:"vod_content"`
	VodPlayFrom    string `json:"vod_play_from"`
	VodPlayURL     string `json:"vod_play_url"`
	VodPlayNote    string `json:"vod_play_note"`
	VodDuration    string `json:"vod_duration"`
	VodTotal       int    `json:"vod_total"`
	VodSerial      string `json:"vod_serial"`
	VodIsEnd       int    `json:"vod_isend"`
}

type playGroup struct {
	Name    string
	Entries []playEntry
}

type playEntry struct {
	Name string
	URL  string
}

func (p *RycjAPIPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	apiURL := strings.TrimSpace(req.Var("apiURL"))
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	siteName := strings.TrimSpace(req.Var("siteName"))
	if siteName == "" {
		siteName = defaultSiteName
	}

	switch {
	case req.Route == "/rycjapi/list" || strings.HasPrefix(req.Route, "/rycjapi/list"):
		return fetchList(apiURL, siteName, req.ChannelID, req.Params)
	case req.Route == "/rycjapi/search" || strings.HasPrefix(req.Route, "/rycjapi/search"):
		query := strings.TrimSpace(req.Params["query"])
		if query == "" {
			return nil, fmt.Errorf("missing query parameter")
		}
		return fetchSearch(apiURL, siteName, query, req.Params)
	case req.Route == "/rycjapi/chapters/:id" || strings.HasPrefix(req.Route, "/rycjapi/chapters"):
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchChapters(apiURL, siteName, id)
	case req.Route == "/rycjapi/episode/:chapterId" || strings.HasPrefix(req.Route, "/rycjapi/episode"):
		parentID := strings.TrimSpace(req.Params["id"])
		chapterID := strings.TrimSpace(req.Params["chapterId"])
		if parentID == "" || chapterID == "" {
			return nil, fmt.Errorf("missing id or chapterId parameter")
		}
		sourceIdx := sourceIndex(req.Params["source"])
		return fetchEpisode(apiURL, siteName, parentID, chapterID, sourceIdx)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(apiURL, siteName, channelID string, params map[string]string) (*sdk.FeedResult, error) {
	page := pageNum(params)
	query := url.Values{
		"ac": {"list"},
		"pg": {strconv.Itoa(page)},
	}
	if typeID := strings.TrimSpace(params["t"]); typeID != "" {
		query.Set("t", typeID)
	}

	resp, err := requestAPI(apiURL, query)
	if err != nil {
		return nil, err
	}
	if len(resp.List) == 0 {
		return nil, fmt.Errorf("empty list data")
	}

	richItems := enrichWithVideoList(apiURL, resp.List)

	title := siteName + " · " + channelTitle(channelID, params["t"])
	result := &sdk.FeedResult{
		Title:       title,
		Description: fmt.Sprintf("第 %d 页，共 %d 条", page, resp.Total),
		Items:       vodItemsToFeedItems(richItems),
	}
	if resp.PageCount > page {
		result.HasMore = true
		result.Next = copyParams(params)
		result.Next["page"] = strconv.Itoa(page + 1)
	}
	return result, nil
}

func fetchSearch(apiURL, siteName, keyword string, params map[string]string) (*sdk.FeedResult, error) {
	page := pageNum(params)
	resp, err := requestAPI(apiURL, url.Values{
		"ac": {"list"},
		"wd": {keyword},
		"pg": {strconv.Itoa(page)},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.List) == 0 {
		return nil, fmt.Errorf("no results for: %s", keyword)
	}

	richItems := enrichWithVideoList(apiURL, resp.List)
	result := &sdk.FeedResult{
		Title:       siteName + " · 搜索",
		Description: fmt.Sprintf("关键词「%s」· 第 %d 页，共 %d 条", keyword, page, resp.Total),
		Items:       vodItemsToFeedItems(richItems),
	}
	if resp.PageCount > page {
		result.HasMore = true
		result.Next = map[string]string{
			"query": keyword,
			"page":  strconv.Itoa(page + 1),
		}
	}
	return result, nil
}

func fetchChapters(apiURL, siteName, vodID string) (*sdk.FeedResult, error) {
	item, groups, err := fetchVodPlayInfo(apiURL, vodID)
	if err != nil {
		return nil, err
	}
	entries := defaultPlayEntries(groups)
	if len(entries) == 0 {
		return nil, fmt.Errorf("no playable episodes for: %s", vodID)
	}

	items := make([]sdk.FeedItem, 0, len(entries))
	for i, entry := range entries {
		items = append(items, sdk.FeedItem{
			ID:    strconv.Itoa(i),
			Title: entry.Name,
			Tags:  []string{defaultRouteName(groups)},
		})
	}

	desc := itemSummary(item)
	if len(groups) > 1 {
		if desc != "" {
			desc += " · "
		}
		desc += fmt.Sprintf("%d 条线路", len(groups))
	}

	return &sdk.FeedResult{
		Title:       itemTitle(item),
		Description: desc,
		Items:       items,
	}, nil
}

func fetchEpisode(apiURL, siteName, parentID, chapterID string, sourceIdx int) (*sdk.FeedResult, error) {
	item, groups, err := fetchVodPlayInfo(apiURL, parentID)
	if err != nil {
		return nil, err
	}
	episodeIdx, err := strconv.Atoi(chapterID)
	if err != nil || episodeIdx < 0 {
		return nil, fmt.Errorf("invalid chapterId: %s", chapterID)
	}

	entries := defaultPlayEntries(groups)
	if episodeIdx >= len(entries) {
		return nil, fmt.Errorf("episode not found: %s", chapterID)
	}
	episode := entries[episodeIdx]

	if sourceIdx < 0 || sourceIdx >= len(groups) {
		sourceIdx = 0
	}
	playURL := episodeURLAt(groups, sourceIdx, episodeIdx)
	if playURL == "" {
		return nil, fmt.Errorf("play url not found")
	}

	title := itemTitle(item) + " · " + episode.Name
	feedItem := sdk.FeedItem{
		ID:          chapterID,
		Title:       title,
		URL:         playURL,
		Summary:     itemSummary(item),
		Content:     buildEpisodePlayerHTML(item, episode, groups, episodeIdx, sourceIdx),
		Author:      strings.TrimSpace(item.VodActor),
		Cover:       strings.TrimSpace(item.VodPic),
		Image:       strings.TrimSpace(item.VodPic),
		PublishedAt: strings.TrimSpace(item.VodTime),
		Tags:        itemTags(item, groups),
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: siteName + " · 播放",
		Items:       []sdk.FeedItem{feedItem},
	}, nil
}

func fetchVodPlayInfo(apiURL, vodID string) (vodItem, []playGroup, error) {
	resp, err := requestAPI(apiURL, url.Values{
		"ac":  {"videolist"},
		"ids": {vodID},
	})
	if err != nil {
		return vodItem{}, nil, err
	}
	if len(resp.List) == 0 {
		return vodItem{}, nil, fmt.Errorf("vod not found: %s", vodID)
	}
	item := resp.List[0]
	groups := parsePlayGroups(item.VodPlayFrom, item.VodPlayURL)
	if len(groups) == 0 {
		return vodItem{}, nil, fmt.Errorf("no play sources for: %s", vodID)
	}
	return item, groups, nil
}

func requestAPI(apiURL string, query url.Values) (*apiResponse, error) {
	rawURL := strings.TrimRight(apiURL, "?") + "?" + query.Encode()
	body, status, err := host.HTTPGet(rawURL, map[string]string{
		"Accept":     "application/json",
		"User-Agent": "OrbitPlugins/1.0",
	})
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("api error: %s", strings.TrimSpace(resp.Msg))
	}
	return &resp, nil
}

func vodItemsToFeedItems(list []vodItem) []sdk.FeedItem {
	items := make([]sdk.FeedItem, 0, len(list))
	for _, item := range list {
		items = append(items, sdk.FeedItem{
			ID:          strconv.Itoa(item.VodID),
			Title:       itemTitle(item),
			Summary:     itemSummary(item),
			Author:      strings.TrimSpace(item.VodActor),
			Cover:       strings.TrimSpace(item.VodPic),
			Image:       strings.TrimSpace(item.VodPic),
			PublishedAt: strings.TrimSpace(item.VodTime),
			Tags:        itemTags(item, nil),
		})
	}
	return items
}

func itemTitle(item vodItem) string {
	title := strings.TrimSpace(item.VodName)
	if title == "" {
		return "未命名视频"
	}
	return title
}

func itemSummary(item vodItem) string {
	parts := make([]string, 0, 6)

	if typeName := strings.TrimSpace(item.TypeName); typeName != "" {
		parts = append(parts, typeName)
	}
	if remarks := strings.TrimSpace(item.VodRemarks); remarks != "" {
		parts = append(parts, remarks)
	}
	if item.VodTotal > 0 {
		parts = append(parts, fmt.Sprintf("共 %d 集", item.VodTotal))
	} else if dur := strings.TrimSpace(item.VodDuration); dur != "" {
		parts = append(parts, dur)
	}
	if year := strings.TrimSpace(item.VodYear); year != "" {
		parts = append(parts, year)
	}
	if area := strings.TrimSpace(item.VodArea); area != "" {
		parts = append(parts, area)
	}

	if len(parts) == 0 {
		if blurb := strings.TrimSpace(item.VodBlurb); blurb != "" {
			return truncate(blurb, 120)
		}
		return ""
	}
	return strings.Join(parts, " · ")
}

func itemTags(item vodItem, playGroups []playGroup) []string {
	var tags []string
	appendTag := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" {
			tags = append(tags, v)
		}
	}
	appendTag(item.TypeName)
	appendTag(item.VodRemarks)
	if item.VodTotal > 0 {
		appendTag(fmt.Sprintf("共 %d 集", item.VodTotal))
	} else if dur := strings.TrimSpace(item.VodDuration); dur != "" {
		appendTag(dur)
	}
	appendTag(item.VodClass)
	appendTag(item.VodLang)
	appendTag(item.VodArea)
	if score := formatScore(item.VodScore); score != "" {
		appendTag("评分 " + score)
	}
	if score := formatScore(item.VodDoubanScore); score != "" {
		appendTag("豆瓣 " + score)
	}
	if len(playGroups) > 0 {
		appendTag(fmt.Sprintf("%d 条线路", len(playGroups)))
	}
	return tags
}

func enrichWithVideoList(apiURL string, base []vodItem) []vodItem {
	if len(base) == 0 {
		return base
	}

	ids := make([]string, 0, len(base))
	for _, v := range base {
		if v.VodID > 0 {
			ids = append(ids, strconv.Itoa(v.VodID))
		}
	}
	if len(ids) == 0 {
		return base
	}

	richResp, err := requestAPI(apiURL, url.Values{
		"ac":  {"videolist"},
		"ids": {strings.Join(ids, ",")},
	})
	if err != nil || len(richResp.List) == 0 {
		return base
	}

	richMap := make(map[int]vodItem, len(richResp.List))
	for _, v := range richResp.List {
		if v.VodID > 0 {
			richMap[v.VodID] = v
		}
	}

	ordered := make([]vodItem, 0, len(base))
	for _, v := range base {
		if rich, ok := richMap[v.VodID]; ok {
			ordered = append(ordered, rich)
		} else {
			ordered = append(ordered, v)
		}
	}
	return ordered
}

func parsePlayGroups(rawFrom, rawURLs string) []playGroup {
	fromParts := strings.Split(rawFrom, "$$$")
	urlParts := strings.Split(rawURLs, "$$$")
	count := len(fromParts)
	if len(urlParts) > count {
		count = len(urlParts)
	}

	groups := make([]playGroup, 0, count)
	for i := 0; i < count; i++ {
		groupName := fmt.Sprintf("线路 %d", i+1)
		if i < len(fromParts) && strings.TrimSpace(fromParts[i]) != "" {
			groupName = strings.TrimSpace(fromParts[i])
		}

		var entries []playEntry
		if i < len(urlParts) {
			for _, block := range strings.Split(urlParts[i], "#") {
				block = strings.TrimSpace(block)
				if block == "" {
					continue
				}
				name, link := parsePlayEntry(block)
				if link == "" {
					continue
				}
				entries = append(entries, playEntry{Name: name, URL: link})
			}
		}
		if len(entries) == 0 {
			continue
		}
		groups = append(groups, playGroup{Name: groupName, Entries: entries})
	}
	return groups
}

func parsePlayEntry(raw string) (string, string) {
	parts := strings.SplitN(raw, "$", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "正片", strings.TrimSpace(raw)
}

func defaultPlayEntries(groups []playGroup) []playEntry {
	if len(groups) == 0 {
		return nil
	}
	return groups[0].Entries
}

func defaultRouteName(groups []playGroup) string {
	if len(groups) == 0 {
		return ""
	}
	return groups[0].Name
}

func episodeURLAt(groups []playGroup, sourceIdx, episodeIdx int) string {
	if sourceIdx < 0 || sourceIdx >= len(groups) {
		return ""
	}
	entries := groups[sourceIdx].Entries
	if episodeIdx < 0 || episodeIdx >= len(entries) {
		return ""
	}
	return strings.TrimSpace(entries[episodeIdx].URL)
}

func episodeURLsAt(groups []playGroup, episodeIdx int) []string {
	urls := make([]string, 0, len(groups))
	for _, group := range groups {
		if episodeIdx >= 0 && episodeIdx < len(group.Entries) {
			urls = append(urls, strings.TrimSpace(group.Entries[episodeIdx].URL))
		} else {
			urls = append(urls, "")
		}
	}
	return urls
}

func buildEpisodePlayerHTML(item vodItem, episode playEntry, groups []playGroup, episodeIdx, activeSource int) string {
	urls := episodeURLsAt(groups, episodeIdx)
	if activeSource < 0 || activeSource >= len(urls) || urls[activeSource] == "" {
		activeSource = 0
	}
	playURL := urls[activeSource]
	if playURL == "" {
		for i, u := range urls {
			if u != "" {
				activeSource = i
				playURL = u
				break
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(`<article class="rycjapi-player" style="color:#1f2937;line-height:1.6;">`)

	if len(groups) > 1 {
		sb.WriteString(`<div style="margin-bottom:12px;display:flex;flex-wrap:wrap;gap:8px;">`)
		for i, group := range groups {
			u := urls[i]
			if u == "" {
				continue
			}
			style := "padding:6px 12px;border-radius:999px;border:1px solid #e5e7eb;background:#fff;color:inherit;cursor:pointer;font-size:13px;"
			if i == activeSource {
				style = "padding:6px 12px;border-radius:999px;border:1px solid #e11d48;background:#fff1f2;color:#be123c;cursor:pointer;font-size:13px;"
			}
			sb.WriteString(fmt.Sprintf(
				`<button type="button" style="%s" onclick="rycjSwitchSource(%d)">%s</button>`,
				style, i, htmlEscape(group.Name),
			))
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`<video id="rycj-player" controls playsinline preload="metadata" style="width:100%;max-width:100%;border-radius:12px;background:#000;display:block;">`)
	sb.WriteString(`<source id="rycj-player-source" src="`)
	sb.WriteString(htmlEscape(playURL))
	sb.WriteString(`" type="`)
	sb.WriteString(videoMIME(playURL))
	sb.WriteString(`"></video>`)

	writeEpisodeInfoSection(&sb, item, episode, groups, episodeIdx, activeSource)

	sb.WriteString(`</article>`)

	if len(groups) > 1 {
		urlJSON, _ := json.Marshal(urls)
		sb.WriteString(`<script>`)
		sb.WriteString(`(function(){`)
		sb.WriteString(`var rycjSources=`)
		sb.WriteString(string(urlJSON))
		sb.WriteString(`;window.rycjSwitchSource=function(idx){`)
		sb.WriteString(`var url=rycjSources[idx];if(!url){return;}`)
		sb.WriteString(`var player=document.getElementById("rycj-player");`)
		sb.WriteString(`if(!player){return;}`)
		sb.WriteString(`player.src=url;player.play();`)
		sb.WriteString(`var buttons=document.querySelectorAll(".rycjapi-player button");`)
		sb.WriteString(`buttons.forEach(function(btn,i){`)
		sb.WriteString(`btn.style.borderColor=i===idx?"#e11d48":"#e5e7eb";`)
		sb.WriteString(`btn.style.background=i===idx?"#fff1f2":"#fff";`)
		sb.WriteString(`btn.style.color=i===idx?"#be123c":"inherit";`)
		sb.WriteString(`});`)
		sb.WriteString(`};})();`)
		sb.WriteString(`</script>`)
	}

	return sb.String()
}

func writeEpisodeInfoSection(sb *strings.Builder, item vodItem, episode playEntry, groups []playGroup, episodeIdx, activeSource int) {
	sb.WriteString(`<section class="rycjapi-info" style="margin-top:20px;padding-top:18px;border-top:1px solid #eef2f7;">`)

	sb.WriteString(`<div style="display:flex;gap:16px;align-items:flex-start;margin-bottom:18px;">`)
	if cover := strings.TrimSpace(item.VodPic); cover != "" {
		sb.WriteString(fmt.Sprintf(
			`<img src="%s" alt="%s" style="width:108px;min-width:108px;height:152px;object-fit:cover;border-radius:10px;background:#f3f4f6;"/>`,
			htmlEscape(cover), htmlEscape(itemTitle(item)),
		))
	}
	sb.WriteString(`<div style="flex:1;min-width:0;">`)
	sb.WriteString(fmt.Sprintf(`<h1 style="margin:0 0 6px;font-size:1.25rem;line-height:1.35;">%s</h1>`, htmlEscape(itemTitle(item))))
	if sub := strings.TrimSpace(item.VodSub); sub != "" {
		sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 10px;color:#6b7280;font-size:14px;">%s</p>`, htmlEscape(sub)))
	}
	writeTagChips(sb, item, groups)
	sb.WriteString(`</div></div>`)

	sb.WriteString(`<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:10px 18px;margin-bottom:18px;">`)
	writeMetaCell(sb, "当前播放", episode.Name)
	writeMetaCell(sb, "分类", item.TypeName)
	writeMetaCell(sb, "更新状态", item.VodRemarks)
	writeMetaCell(sb, "年份", item.VodYear)
	writeMetaCell(sb, "地区", item.VodArea)
	writeMetaCell(sb, "语言", item.VodLang)
	writeMetaCell(sb, "时长", item.VodDuration)
	writeMetaCell(sb, "上映", item.VodPubdate)
	writeMetaCell(sb, "最近更新", item.VodTime)
	if item.VodTotal > 0 {
		writeMetaCell(sb, "总集数", fmt.Sprintf("%d 集", item.VodTotal))
		writeMetaCell(sb, "当前集数", fmt.Sprintf("第 %d / %d 集", episodeIdx+1, len(defaultPlayEntries(groups))))
	}
	if serial := strings.TrimSpace(item.VodSerial); serial != "" {
		writeMetaCell(sb, "连载进度", serial)
	}
	if item.VodIsEnd == 1 {
		writeMetaCell(sb, "完结状态", "已完结")
	}
	if score := formatScore(item.VodScore); score != "" {
		writeMetaCell(sb, "评分", score)
	}
	if score := formatScore(item.VodDoubanScore); score != "" {
		writeMetaCell(sb, "豆瓣", score)
	}
	if len(groups) > 0 {
		writeMetaCell(sb, "播放线路", groups[activeSource].Name)
		if len(groups) > 1 {
			writeMetaCell(sb, "可用线路", fmt.Sprintf("%d 条", len(groups)))
		}
	}
	sb.WriteString(`</div>`)

	if director := strings.TrimSpace(item.VodDirector); director != "" {
		writeInfoBlock(sb, "导演", director)
	}
	if writer := strings.TrimSpace(item.VodWriter); writer != "" {
		writeInfoBlock(sb, "编剧", writer)
	}
	if actor := strings.TrimSpace(item.VodActor); actor != "" {
		writeInfoBlock(sb, "主演", actor)
	}

	if synopsis := vodSynopsis(item); synopsis != "" {
		sb.WriteString(`<div style="margin-top:16px;">`)
		sb.WriteString(`<h2 style="margin:0 0 8px;font-size:15px;">简介</h2>`)
		sb.WriteString(fmt.Sprintf(`<p style="margin:0;color:#374151;white-space:pre-wrap;">%s</p>`, htmlEscape(synopsis)))
		sb.WriteString(`</div>`)
	}

	if note := strings.TrimSpace(item.VodPlayNote); note != "" && note != "$$$" {
		sb.WriteString(`<p style="margin:16px 0 0;color:#9ca3af;font-size:13px;">播放提示：`)
		sb.WriteString(htmlEscape(note))
		sb.WriteString(`</p>`)
	}

	sb.WriteString(`</section>`)
}

func writeTagChips(sb *strings.Builder, item vodItem, groups []playGroup) {
	tags := make([]string, 0, 8)
	appendChip := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" {
			tags = append(tags, v)
		}
	}
	appendChip(item.TypeName)
	appendChip(item.VodRemarks)
	for _, part := range strings.Split(item.VodClass, ",") {
		appendChip(part)
	}
	if item.VodTotal > 0 {
		appendChip(fmt.Sprintf("共 %d 集", item.VodTotal))
	} else if dur := strings.TrimSpace(item.VodDuration); dur != "" {
		appendChip(dur)
	}
	if len(groups) > 1 {
		appendChip(fmt.Sprintf("%d 线路", len(groups)))
	}
	if len(tags) == 0 {
		return
	}
	sb.WriteString(`<div style="display:flex;flex-wrap:wrap;gap:6px;">`)
	for _, tag := range tags {
		sb.WriteString(fmt.Sprintf(
			`<span style="display:inline-block;padding:4px 10px;border-radius:999px;background:#f3f4f6;color:#4b5563;font-size:12px;">%s</span>`,
			htmlEscape(tag),
		))
	}
	sb.WriteString(`</div>`)
}

func writeMetaCell(sb *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	sb.WriteString(`<div style="min-width:0;">`)
	sb.WriteString(fmt.Sprintf(`<div style="color:#9ca3af;font-size:12px;margin-bottom:2px;">%s</div>`, htmlEscape(label)))
	sb.WriteString(fmt.Sprintf(`<div style="font-size:14px;word-break:break-word;">%s</div>`, htmlEscape(value)))
	sb.WriteString(`</div>`)
}

func writeInfoBlock(sb *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	sb.WriteString(`<div style="margin-bottom:10px;">`)
	sb.WriteString(fmt.Sprintf(`<div style="color:#9ca3af;font-size:12px;margin-bottom:4px;">%s</div>`, htmlEscape(label)))
	sb.WriteString(fmt.Sprintf(`<div style="font-size:14px;line-height:1.7;">%s</div>`, htmlEscape(value)))
	sb.WriteString(`</div>`)
}

func vodSynopsis(item vodItem) string {
	if text := cleanVodText(item.VodContent); text != "" {
		return text
	}
	return cleanVodText(item.VodBlurb)
}

func cleanVodText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = reStripHTML.ReplaceAllString(raw, " ")
	raw = html.UnescapeString(raw)
	raw = strings.ReplaceAll(raw, "\u00a0", " ")
	return strings.Join(strings.Fields(raw), " ")
}

func formatScore(score string) string {
	score = strings.TrimSpace(score)
	if score == "" || score == "0" || score == "0.0" {
		return ""
	}
	return score
}

func videoMIME(playURL string) string {
	lower := strings.ToLower(playURL)
	if strings.Contains(lower, ".m3u8") {
		return "application/x-mpegURL"
	}
	return "video/mp4"
}

func sourceIndex(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	idx, err := strconv.Atoi(raw)
	if err != nil || idx < 0 {
		return 0
	}
	return idx
}

func pageNum(params map[string]string) int {
	page, err := strconv.Atoi(strings.TrimSpace(params["page"]))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func channelTitle(channelID, typeID string) string {
	if title := strings.TrimSpace(channelLabels[channelID]); title != "" {
		return title
	}
	if title := strings.TrimSpace(typeLabels[typeID]); title != "" {
		return title
	}
	if typeID != "" {
		return "分类 " + typeID
	}
	return "列表"
}

func copyParams(params map[string]string) map[string]string {
	next := make(map[string]string, len(params))
	for k, v := range params {
		next[k] = v
	}
	return next
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

func truncate(s string, n int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= n {
		return string(runes)
	}
	return string(runes[:n]) + "…"
}
