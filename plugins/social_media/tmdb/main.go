package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/orbit-tauri-tools/plugin-sdk"
	"github.com/orbit-tauri-tools/plugin-sdk/host"
)

const (
	apiBase      = "https://api.themoviedb.org/3"
	imagePoster  = "https://image.tmdb.org/t/p/w500"
	imageBackdrop = "https://image.tmdb.org/t/p/w1280"
	imageProfile = "https://image.tmdb.org/t/p/w185"
	imageGallery = "https://image.tmdb.org/t/p/w780"
)

const maxGalleryImages = 36

func main() {
	sdk.Run(&TMDBPlugin{})
}

type TMDBPlugin struct{}

var channelLabels = map[string]string{
	"movie_popular":       "热门电影",
	"tv_popular":          "热门电视剧",
	"movie_top_rated":     "高分电影",
	"tv_top_rated":        "高分电视剧",
	"recent_tv_top":       "近期高分剧集",
	"recent_tv_popular":   "近期热门剧集",
	"recent_movie_top":    "近期高分电影",
	"recent_movie_popular": "近期热门电影",
	"trending_movie_day":  "今日热榜 · 电影",
	"trending_movie_week": "本周热榜 · 电影",
	"trending_tv_day":     "今日热榜 · 剧集",
	"trending_tv_week":    "本周热榜 · 剧集",
	"trending_all_day":    "今日热榜 · 全类型",
	"trending_all_week":   "本周热榜 · 全类型",
	"movie_upcoming":      "即将上映",
	"movie_now_playing":   "正在上映",
	"anime_tv":            "动漫排行",
	"anime_movie":         "动漫电影",
	"top_anime":           "高分动漫",
	"cn_animation":         "国漫排行",
	"cn_animation_top":     "高分国漫",
	"recent_cn_animation":  "近期热门国漫",
	"recent_cn_top":        "近期高分国漫",
	"korean_drama":         "韩剧排行",
	"korean_drama_top":     "高分韩剧",
	"recent_korean_drama":  "近期热门韩剧",
	"recent_korean_top":    "近期高分韩剧",
	"japanese_drama":       "日剧排行",
	"chinese_drama":        "华语剧集",
	"western_drama":        "欧美剧集",
	"netflix_tv":           "Netflix 剧集",
	"netflix_tv_top":       "Netflix 高分剧集",
	"recent_netflix_tv":    "近期 Netflix 剧集",
	"recent_netflix_top":   "近期 Netflix 高分",
	"netflix_movie":        "Netflix 电影",
	"disney_tv":           "Disney+ 剧集",
	"prime_tv":            "Prime Video 剧集",
	"sci_fi_tv":           "科幻奇幻剧集",
	"action_movie":        "动作电影",
	"search":              "搜索",
}

var channelEndpoints = map[string]string{
	"movie_popular":       "movie/popular",
	"tv_popular":          "tv/popular",
	"movie_top_rated":     "movie/top_rated",
	"tv_top_rated":        "tv/top_rated",
	"trending_movie_day":  "trending/movie/day",
	"trending_movie_week": "trending/movie/week",
	"trending_all_day":    "trending/all/day",
	"trending_all_week":   "trending/all/week",
	"movie_upcoming":      "movie/upcoming",
	"movie_now_playing":   "movie/now_playing",
	"trending_tv_day":     "trending/tv/day",
	"trending_tv_week":    "trending/tv/week",
}

var listReservedParams = map[string]bool{
	"endpoint":         true,
	"page":             true,
	"use_recent_since": true,
	"use_min_votes":    true,
}

func (p *TMDBPlugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	apiKey := req.Var("apiKey")
	if apiKey == "" {
		return nil, fmt.Errorf("TMDB API key required (configure variable apiKey in plugin settings)")
	}
	language := req.Var("language")
	if language == "" {
		language = "zh-CN"
	}
	watchRegion := req.Var("watchRegion")
	if watchRegion == "" {
		watchRegion = "US"
	}
	recentSince := req.Var("recentSince")
	if recentSince == "" {
		recentSince = "2023-01-01"
	}
	minVoteCount := req.Var("minVoteCount")
	if minVoteCount == "" {
		minVoteCount = "200"
	}

	switch {
	case req.Route == "/tmdb/list":
		endpoint := strings.TrimSpace(req.Params["endpoint"])
		if endpoint == "" {
			if ep, ok := channelEndpoints[req.ChannelID]; ok {
				endpoint = ep
				req.Params["endpoint"] = endpoint
			}
		}
		if endpoint == "" {
			return nil, fmt.Errorf("missing endpoint parameter")
		}
		page := pageNum(req.Params)
		return fetchList(req.Params, page, apiKey, language, watchRegion, recentSince, minVoteCount, req.ChannelID)
	case req.Route == "/tmdb/search" || strings.HasPrefix(req.Route, "/tmdb/search"):
		query := strings.TrimSpace(req.Params["query"])
		if query == "" {
			return nil, fmt.Errorf("missing query parameter")
		}
		page := pageNum(req.Params)
		searchType := strings.TrimSpace(req.Params["type"])
		if searchType == "" {
			searchType = "multi"
		}
		return fetchSearch(query, page, searchType, apiKey, language)
	case req.Route == "/tmdb/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id, apiKey, language)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

type listResponse struct {
	Page         int          `json:"page"`
	Results      []listItem   `json:"results"`
	TotalPages   int          `json:"total_pages"`
	TotalResults int          `json:"total_results"`
}

type listItem struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	ReleaseDate   string  `json:"release_date"`
	FirstAirDate  string  `json:"first_air_date"`
	MediaType     string  `json:"media_type"`
	KnownForDept  string  `json:"known_for_department"`
}

func fetchList(params map[string]string, page int, apiKey, language, watchRegion, recentSince, minVoteCount, channelID string) (*sdk.FeedResult, error) {
	endpoint := strings.TrimSpace(params["endpoint"])
	query := url.Values{
		"language": {language},
		"page":     {strconv.Itoa(page)},
	}
	for key, value := range params {
		if listReservedParams[key] || strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}
	applyDiscoverFilters(query, params, recentSince, minVoteCount)
	if query.Get("with_watch_providers") != "" && query.Get("watch_region") == "" {
		query.Set("watch_region", watchRegion)
	}

	rawURL, headers := buildTMDBRequest(endpoint, query, apiKey)

	body, status, err := host.HTTPGet(rawURL, headers)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("empty list data")
	}

	defaultMedia := inferMediaType(endpoint)
	items := make([]sdk.FeedItem, 0, len(resp.Results))
	for _, item := range resp.Results {
		feedItem, ok := listItemToFeed(item, defaultMedia)
		if !ok {
			continue
		}
		items = append(items, feedItem)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no valid items in list")
	}

	label := channelLabels[channelID]
	if label == "" {
		label = endpoint
	}

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("TMDB · %s", label),
		Description: listDescription(params, recentSince, resp.Page, resp.TotalResults),
		Items:       items,
	}
	if resp.Page < resp.TotalPages {
		result.HasMore = true
		result.Next = map[string]string{
			"page": strconv.Itoa(resp.Page + 1),
		}
		for key, value := range params {
			if key == "page" {
				continue
			}
			result.Next[key] = value
		}
	}
	return result, nil
}

func applyDiscoverFilters(query url.Values, params map[string]string, recentSince, minVoteCount string) {
	switch params["use_recent_since"] {
	case "tv":
		query.Set("first_air_date.gte", recentSince)
	case "movie":
		query.Set("primary_release_date.gte", recentSince)
	}
	if params["use_min_votes"] == "true" && query.Get("vote_count.gte") == "" {
		query.Set("vote_count.gte", minVoteCount)
	}
}

func listDescription(params map[string]string, recentSince string, page, total int) string {
	if params["use_recent_since"] != "" {
		return fmt.Sprintf("自 %s 起 · 第 %d 页，共 %d 条", recentSince, page, total)
	}
	return fmt.Sprintf("第 %d 页，共 %d 条", page, total)
}

func fetchSearch(query string, page int, searchType, apiKey, language string) (*sdk.FeedResult, error) {
	endpoint := "search/multi"
	defaultMedia := ""
	switch searchType {
	case "movie":
		endpoint = "search/movie"
		defaultMedia = "movie"
	case "tv":
		endpoint = "search/tv"
		defaultMedia = "tv"
	case "multi":
		endpoint = "search/multi"
	default:
		return nil, fmt.Errorf("unsupported search type: %s", searchType)
	}

	rawURL, headers := buildTMDBRequest(endpoint, url.Values{
		"query":         {query},
		"language":      {language},
		"page":          {strconv.Itoa(page)},
		"include_adult": {"false"},
	}, apiKey)

	body, status, err := host.HTTPGet(rawURL, headers)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no results for: %s", query)
	}

	items := make([]sdk.FeedItem, 0, len(resp.Results))
	for _, item := range resp.Results {
		if item.MediaType == "person" {
			continue
		}
		feedItem, ok := listItemToFeed(item, defaultMedia)
		if !ok {
			continue
		}
		items = append(items, feedItem)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no movie/tv results for: %s", query)
	}

	typeLabel := map[string]string{
		"multi": "综合",
		"movie": "电影",
		"tv":    "剧集",
	}[searchType]

	result := &sdk.FeedResult{
		Title:       fmt.Sprintf("TMDB · 搜索 · %s", query),
		Description: fmt.Sprintf("%s · 第 %d 页，共 %d 条", typeLabel, resp.Page, resp.TotalResults),
		Items:       items,
	}
	if resp.Page < resp.TotalPages {
		result.HasMore = true
		result.Next = map[string]string{
			"query": query,
			"type":  searchType,
			"page":  strconv.Itoa(resp.Page + 1),
		}
	}
	return result, nil
}

func inferMediaType(endpoint string) string {
	switch {
	case strings.HasPrefix(endpoint, "movie/"), strings.HasPrefix(endpoint, "discover/movie"):
		return "movie"
	case strings.HasPrefix(endpoint, "tv/"), strings.HasPrefix(endpoint, "discover/tv"), strings.HasPrefix(endpoint, "trending/tv"):
		return "tv"
	default:
		return ""
	}
}

func listItemToFeed(item listItem, defaultMedia string) (sdk.FeedItem, bool) {
	mediaType := item.MediaType
	if mediaType == "" {
		mediaType = defaultMedia
	}
	if mediaType == "" {
		if item.Title != "" {
			mediaType = "movie"
		} else if item.Name != "" {
			mediaType = "tv"
		}
	}
	if mediaType == "" {
		return sdk.FeedItem{}, false
	}

	title := item.Title
	if title == "" {
		title = item.Name
	}
	if strings.TrimSpace(title) == "" {
		return sdk.FeedItem{}, false
	}

	poster := imageURL(imagePoster, item.PosterPath)
	backdrop := imageURL(imageBackdrop, item.BackdropPath)
	cover := poster
	if cover == "" {
		cover = backdrop
	}

	date := item.ReleaseDate
	if date == "" {
		date = item.FirstAirDate
	}

	tags := []string{mediaTypeLabel(mediaType)}
	if item.VoteAverage > 0 {
		tags = append(tags, fmt.Sprintf("评分 %.1f", item.VoteAverage))
	}
	if date != "" {
		tags = append(tags, date)
	}
	if item.KnownForDept != "" {
		tags = append(tags, item.KnownForDept)
	}

	summary := strings.TrimSpace(item.Overview)
	if len([]rune(summary)) > 200 {
		summary = string([]rune(summary)[:200]) + "…"
	}

	return sdk.FeedItem{
		ID:      fmt.Sprintf("%s:%d", mediaType, item.ID),
		Title:   title,
		URL:     tmdbWebURL(mediaType, item.ID),
		Summary: summary,
		Cover:   cover,
		Image:   cover,
		Tags:    tags,
	}, true
}

// Detail response with append_to_response fields inlined.
type detailResponse struct {
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	Name         string  `json:"name"`
	Tagline      string  `json:"tagline"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	VoteAverage  float64 `json:"vote_average"`
	VoteCount    int     `json:"vote_count"`
	ReleaseDate  string  `json:"release_date"`
	FirstAirDate string  `json:"first_air_date"`
	Runtime      int     `json:"runtime"`
	EpisodeRunTime []int `json:"episode_run_time"`
	NumberOfSeasons int  `json:"number_of_seasons"`
	NumberOfEpisodes int `json:"number_of_episodes"`
	Status       string  `json:"status"`
	Homepage     string  `json:"homepage"`
	Biography    string  `json:"biography"`
	Birthday     string  `json:"birthday"`
	PlaceOfBirth string  `json:"place_of_birth"`
	Genres       []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
	Credits *struct {
		Cast []struct {
			Name        string `json:"name"`
			Character   string `json:"character"`
			ProfilePath string `json:"profile_path"`
			Order       int    `json:"order"`
		} `json:"cast"`
		Crew []struct {
			Name        string `json:"name"`
			Job         string `json:"job"`
			Department  string `json:"department"`
			ProfilePath string `json:"profile_path"`
		} `json:"crew"`
	} `json:"credits"`
	Images *struct {
		Backdrops []struct {
			FilePath    string  `json:"file_path"`
			VoteAverage float64 `json:"vote_average"`
		} `json:"backdrops"`
		Posters []struct {
			FilePath    string  `json:"file_path"`
			VoteAverage float64 `json:"vote_average"`
		} `json:"posters"`
	} `json:"images"`
	Reviews *struct {
		Results []struct {
			Author  string `json:"author"`
			Content string `json:"content"`
			URL     string `json:"url"`
			AuthorDetails struct {
				Rating float64 `json:"rating"`
			} `json:"author_details"`
		} `json:"results"`
	} `json:"reviews"`
	Videos *struct {
		Results []struct {
			Key  string `json:"key"`
			Site string `json:"site"`
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"results"`
	} `json:"videos"`
}

func fetchDetail(itemID, apiKey, language string) (*sdk.FeedResult, error) {
	mediaType, id, err := parseItemID(itemID)
	if err != nil {
		return nil, err
	}

	appendTo := "credits,images,reviews,videos"
	if mediaType == "person" {
		appendTo = "images,combined_credits"
	}

	rawURL, headers := buildTMDBRequest(
		fmt.Sprintf("%s/%s", mediaType, id),
		url.Values{
			"language":               {language},
			"append_to_response":     {appendTo},
			"include_image_language": {imageLanguages(language)},
		},
		apiKey,
	)

	body, status, err := host.HTTPGet(rawURL, headers)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("http status %d: %s", status, truncate(string(body), 200))
	}

	var detail detailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, fmt.Errorf("parse detail response: %w", err)
	}

	title := detail.Title
	if title == "" {
		title = detail.Name
	}
	if title == "" {
		title = "详情"
	}

	poster := imageURL(imagePoster, detail.PosterPath)
	backdrop := imageURL(imageBackdrop, detail.BackdropPath)
	cover := poster
	if cover == "" {
		cover = backdrop
	}

	content := buildDetailContent(mediaType, &detail)
	summary := strings.TrimSpace(detail.Overview)
	if summary == "" {
		summary = strings.TrimSpace(detail.Biography)
	}
	if len([]rune(summary)) > 400 {
		summary = string([]rune(summary)[:400]) + "…"
	}

	tags := []string{mediaTypeLabel(mediaType)}
	for _, g := range detail.Genres {
		if g.Name != "" {
			tags = append(tags, g.Name)
		}
	}
	if detail.VoteAverage > 0 {
		tags = append(tags, fmt.Sprintf("评分 %.1f", detail.VoteAverage))
	}

	item := sdk.FeedItem{
		ID:      itemID,
		Title:   title,
		URL:     tmdbWebURL(mediaType, detail.ID),
		Summary: summary,
		Content: content,
		Cover:   cover,
		Image:   cover,
		Tags:    tags,
	}

	return &sdk.FeedResult{
		Title:       title,
		Description: "TMDB 详情",
		Items:       []sdk.FeedItem{item},
	}, nil
}

func buildDetailContent(mediaType string, d *detailResponse) string {
	var sb strings.Builder
	sb.WriteString(`<div class="tmdb-detail">`)

	backdrop := imageURL(imageBackdrop, d.BackdropPath)
	if backdrop != "" {
		sb.WriteString(fmt.Sprintf(
			`<div style="margin-bottom:1rem;border-radius:8px;overflow:hidden;"><img src="%s" style="width:100%%;display:block;" alt="backdrop"/></div>`,
			backdrop,
		))
	}

	poster := imageURL(imagePoster, d.PosterPath)
	if poster != "" {
		sb.WriteString(fmt.Sprintf(
			`<img src="%s" style="max-width:220px;border-radius:8px;float:left;margin:0 1rem 1rem 0;" alt="poster"/>`,
			poster,
		))
	}

	title := d.Title
	if title == "" {
		title = d.Name
	}
	sb.WriteString(fmt.Sprintf(`<h2 style="margin:0 0 0.5rem;">%s</h2>`, htmlEscape(title)))

	if d.Tagline != "" {
		sb.WriteString(fmt.Sprintf(`<p style="font-style:italic;color:#888;margin:0 0 1rem;">%s</p>`, htmlEscape(d.Tagline)))
	}

	sb.WriteString(`<div style="margin-bottom:1rem;">`)
	if d.VoteAverage > 0 {
		sb.WriteString(fmt.Sprintf(`<p><strong>评分:</strong> %.1f/10`, d.VoteAverage))
		if d.VoteCount > 0 {
			sb.WriteString(fmt.Sprintf(` (%d 票)`, d.VoteCount))
		}
		sb.WriteString(`</p>`)
	}

	date := d.ReleaseDate
	if date == "" {
		date = d.FirstAirDate
	}
	if date != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>日期:</strong> %s</p>`, htmlEscape(date)))
	}
	if d.Runtime > 0 {
		sb.WriteString(fmt.Sprintf(`<p><strong>时长:</strong> %d 分钟</p>`, d.Runtime))
	}
	if len(d.EpisodeRunTime) > 0 {
		sb.WriteString(fmt.Sprintf(`<p><strong>单集时长:</strong> %d 分钟</p>`, d.EpisodeRunTime[0]))
	}
	if d.NumberOfSeasons > 0 {
		sb.WriteString(fmt.Sprintf(`<p><strong>季数:</strong> %d · <strong>集数:</strong> %d</p>`, d.NumberOfSeasons, d.NumberOfEpisodes))
	}
	if d.Status != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>状态:</strong> %s</p>`, htmlEscape(d.Status)))
	}
	if len(d.Genres) > 0 {
		names := make([]string, 0, len(d.Genres))
		for _, g := range d.Genres {
			if g.Name != "" {
				names = append(names, g.Name)
			}
		}
		if len(names) > 0 {
			sb.WriteString(fmt.Sprintf(`<p><strong>类型:</strong> %s</p>`, htmlEscape(strings.Join(names, " / "))))
		}
	}
	if mediaType == "person" {
		if d.Birthday != "" {
			sb.WriteString(fmt.Sprintf(`<p><strong>生日:</strong> %s</p>`, htmlEscape(d.Birthday)))
		}
		if d.PlaceOfBirth != "" {
			sb.WriteString(fmt.Sprintf(`<p><strong>出生地:</strong> %s</p>`, htmlEscape(d.PlaceOfBirth)))
		}
	}
	if d.Homepage != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>官网:</strong> <a href="%s" target="_blank" rel="noopener">%s</a></p>`, htmlEscape(d.Homepage), htmlEscape(d.Homepage)))
	}
	sb.WriteString(`</div>`)

	overview := d.Overview
	if overview == "" {
		overview = d.Biography
	}
	if overview != "" {
		sb.WriteString(`<div style="clear:both;margin-bottom:1.5rem;">`)
		sb.WriteString(`<p><strong>简介</strong></p>`)
		sb.WriteString(fmt.Sprintf(`<p style="line-height:1.7;color:#444;">%s</p>`, htmlEscape(overview)))
		sb.WriteString(`</div>`)
	}

	if d.Videos != nil {
		var trailers []string
		for _, v := range d.Videos.Results {
			if strings.EqualFold(v.Site, "YouTube") && v.Key != "" {
				embed := fmt.Sprintf(`https://www.youtube.com/embed/%s`, v.Key)
				trailers = append(trailers, fmt.Sprintf(
					`<div style="margin-bottom:1rem;"><p><strong>%s</strong> (%s)</p><div style="position:relative;padding-bottom:56.25%%;height:0;overflow:hidden;border-radius:8px;"><iframe src="%s" style="position:absolute;top:0;left:0;width:100%%;height:100%%;border:0;" allowfullscreen></iframe></div></div>`,
					htmlEscape(v.Name), htmlEscape(v.Type), embed,
				))
			}
		}
		if len(trailers) > 0 {
			sb.WriteString(`<div style="margin-bottom:1.5rem;">`)
			sb.WriteString(`<p><strong>预告片</strong></p>`)
			for _, t := range trailers {
				sb.WriteString(t)
			}
			sb.WriteString(`</div>`)
		}
	}

	if d.Credits != nil {
		directors := filterCrew(d.Credits.Crew, "Director")
		if len(directors) > 0 {
			sb.WriteString(`<div style="margin-bottom:1.5rem;">`)
			sb.WriteString(fmt.Sprintf(`<p><strong>导演</strong></p><p>%s</p>`, htmlEscape(strings.Join(directors, "、"))))
			sb.WriteString(`</div>`)
		}

		castLimit := 12
		if len(d.Credits.Cast) > 0 {
			sb.WriteString(`<div style="margin-bottom:1.5rem;">`)
			sb.WriteString(`<p><strong>演员</strong></p>`)
			sb.WriteString(`<div style="display:flex;flex-wrap:wrap;gap:12px;">`)
			for i, c := range d.Credits.Cast {
				if i >= castLimit {
					break
				}
				profile := imageURL(imageProfile, c.ProfilePath)
				sb.WriteString(`<div style="width:90px;text-align:center;font-size:12px;">`)
				if profile != "" {
					sb.WriteString(fmt.Sprintf(`<img src="%s" style="width:72px;height:72px;border-radius:50%%;object-fit:cover;display:block;margin:0 auto 4px;" alt="%s"/>`, profile, htmlEscape(c.Name)))
				}
				sb.WriteString(fmt.Sprintf(`<div style="font-weight:600;">%s</div>`, htmlEscape(c.Name)))
				if c.Character != "" {
					sb.WriteString(fmt.Sprintf(`<div style="color:#888;">%s</div>`, htmlEscape(c.Character)))
				}
				sb.WriteString(`</div>`)
			}
			sb.WriteString(`</div></div>`)
		}
	}

	if gallery := collectGalleryImages(d, maxGalleryImages); len(gallery) > 0 {
		sb.WriteString(`<div style="margin-bottom:1.5rem;">`)
		sb.WriteString(fmt.Sprintf(`<p><strong>剧照</strong> <span style="opacity:0.65;">(%d)</span></p>`, len(gallery)))
		sb.WriteString(`<div style="display:flex;flex-wrap:wrap;gap:8px;">`)
		for _, path := range gallery {
			img := imageURL(imageGallery, path)
			if img != "" {
				sb.WriteString(fmt.Sprintf(
					`<img src="%s" style="flex:1 1 160px;max-width:calc(50%% - 4px);border-radius:6px;display:block;" alt="still" loading="lazy"/>`,
					img,
				))
			}
		}
		sb.WriteString(`</div></div>`)
	}

	if d.Reviews != nil && len(d.Reviews.Results) > 0 {
		sb.WriteString(`<div style="margin-top:1.5rem;padding-top:1rem;border-top:1px solid #eee;">`)
		sb.WriteString(`<p><strong>影评</strong></p>`)
		limit := 5
		if len(d.Reviews.Results) < limit {
			limit = len(d.Reviews.Results)
		}
		for i := 0; i < limit; i++ {
			r := d.Reviews.Results[i]
			sb.WriteString(`<div style="margin-bottom:1rem;">`)
			sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 6px;font-weight:600;">%s`, htmlEscape(r.Author)))
			if r.AuthorDetails.Rating > 0 {
				sb.WriteString(fmt.Sprintf(` <span style="color:#e67e22;">★ %.0f</span>`, r.AuthorDetails.Rating))
			}
			sb.WriteString(`</p>`)
			content := r.Content
			if len([]rune(content)) > 500 {
				content = string([]rune(content)[:500]) + "…"
			}
			sb.WriteString(fmt.Sprintf(`<p style="margin:0;line-height:1.6;color:#555;">%s</p>`, htmlEscape(content)))
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

func filterCrew(crew []struct {
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}, job string) []string {
	seen := make(map[string]bool)
	var names []string
	for _, c := range crew {
		if strings.EqualFold(c.Job, job) && c.Name != "" && !seen[c.Name] {
			seen[c.Name] = true
			names = append(names, c.Name)
		}
	}
	return names
}

func parseItemID(itemID string) (mediaType, id string, err error) {
	parts := strings.SplitN(itemID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid item id format (expected movie:123 or tv:456): %s", itemID)
	}
	switch parts[0] {
	case "movie", "tv", "person":
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("unsupported media type: %s", parts[0])
	}
}

// buildTMDBRequest constructs a TMDB API URL and headers.
// Supports v3 API Key (query param) and Read Access Token (Bearer JWT).
func buildTMDBRequest(path string, params url.Values, credential string) (string, map[string]string) {
	headers := map[string]string{"Accept": "application/json"}
	if isBearerToken(credential) {
		headers["Authorization"] = "Bearer " + credential
	} else {
		params.Set("api_key", credential)
	}
	rawURL := apiBase + "/" + strings.TrimPrefix(path, "/")
	if encoded := params.Encode(); encoded != "" {
		rawURL += "?" + encoded
	}
	return rawURL, headers
}

func imageLanguages(language string) string {
	langs := []string{"null", "en"}
	if code := primaryLanguageCode(language); code != "" && code != "en" {
		langs = append(langs, code)
	}
	return strings.Join(langs, ",")
}

func primaryLanguageCode(language string) string {
	language = strings.TrimSpace(language)
	if language == "" {
		return ""
	}
	if i := strings.Index(language, "-"); i > 0 {
		return strings.ToLower(language[:i])
	}
	return strings.ToLower(language)
}

type rankedImage struct {
	FilePath    string
	VoteAverage float64
}

func collectGalleryImages(d *detailResponse, limit int) []string {
	if d.Images == nil || limit <= 0 {
		return nil
	}

	seen := make(map[string]bool)
	paths := make([]string, 0, limit)

	appendRanked := func(items []rankedImage) {
		sort.Slice(items, func(i, j int) bool {
			return items[i].VoteAverage > items[j].VoteAverage
		})
		for _, item := range items {
			if item.FilePath == "" || seen[item.FilePath] {
				continue
			}
			seen[item.FilePath] = true
			paths = append(paths, item.FilePath)
			if len(paths) >= limit {
				return
			}
		}
	}

	backdrops := make([]rankedImage, 0, len(d.Images.Backdrops))
	for _, b := range d.Images.Backdrops {
		backdrops = append(backdrops, rankedImage{FilePath: b.FilePath, VoteAverage: b.VoteAverage})
	}
	appendRanked(backdrops)
	if len(paths) >= limit {
		return paths
	}

	posters := make([]rankedImage, 0, len(d.Images.Posters))
	for _, p := range d.Images.Posters {
		posters = append(posters, rankedImage{FilePath: p.FilePath, VoteAverage: p.VoteAverage})
	}
	appendRanked(posters)

	return paths
}

func isBearerToken(credential string) bool {
	return strings.HasPrefix(credential, "eyJ")
}

func pageNum(params map[string]string) int {
	page, _ := strconv.Atoi(strings.TrimSpace(params["page"]))
	if page < 1 {
		return 1
	}
	return page
}

func imageURL(base, path string) string {
	if path == "" {
		return ""
	}
	return base + path
}

func tmdbWebURL(mediaType string, id int) string {
	switch mediaType {
	case "movie":
		return fmt.Sprintf("https://www.themoviedb.org/movie/%d", id)
	case "tv":
		return fmt.Sprintf("https://www.themoviedb.org/tv/%d", id)
	case "person":
		return fmt.Sprintf("https://www.themoviedb.org/person/%d", id)
	default:
		return fmt.Sprintf("https://www.themoviedb.org/%s/%d", mediaType, id)
	}
}

func mediaTypeLabel(mediaType string) string {
	switch mediaType {
	case "movie":
		return "电影"
	case "tv":
		return "剧集"
	case "person":
		return "人物"
	default:
		return mediaType
	}
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
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
