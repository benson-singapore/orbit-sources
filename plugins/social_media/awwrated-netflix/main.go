package main

import (
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
	graphqlURL   = "https://cloud.awwrated.com/graphql"
	siteBase     = "https://awwrated.com/zh-tw"
	pageSize     = 30
	categoryID   = 2
	platformPath = "netflix"
	pluginName   = "awwrated · Netflix"
)

func main() {
	sdk.Run(&Plugin{})
}

type Plugin struct{}

var channelLabels = map[string]string{
	"type_movie":             "电影",
	"type_series":            "电视影集",
	"type_ani":               "动画",
	"type_ani_movie":         "动画电影",
	"sort_new":               "新上架",
	"sort_recent_top":        "近期好评",
	"sort_top_rated":         "最好评",
	"sort_aww_hot":           "热门 aww 评",
	"sort_popular":           "最热门",
	"sort_latest":            "最新",
	"sort_oldest":            "最旧",
	"sort_worst":             "最…!!?",
	"region_us":              "美国",
	"region_tw":              "台湾",
	"region_kr":              "韩国",
	"region_jp":              "日本",
	"region_uk":              "英国",
	"region_cn":              "中国",
	"region_hk":              "香港",
	"region_in":              "印度",
	"region_th":              "泰国",
	"genre_edchoice":         "编辑推荐",
	"genre_drama":            "剧情",
	"genre_horror":           "恐怖",
	"genre_thriller":         "惊悚",
	"genre_mystery":          "悬疑",
	"genre_comedy":           "喜剧",
	"genre_crime":            "犯罪",
	"genre_scifi":            "科幻",
	"genre_romance":          "爱情",
	"genre_action":           "动作",
	"genre_fantasy":          "奇幻",
	"genre_documentary":      "纪录片",
	"genre_war":              "战争",
	"genre_family":           "合家欢",
	"genre_reality":          "实境节目",
	"genre_short":            "短片集",
	"genre_history":          "历史",
	"genre_talkshow":         "脱口秀",
	"genre_music":            "音乐",
	"genre_biography":        "传记",
	"genre_sport":            "运动",
	"genre_based_on_game":    "改编自游戏",
	"genre_based_on_comic":   "改编自漫画",
	"genre_based_on_book":    "改编自小说",
	"genre_based_on_true":    "改编自真实故事",
	"year_2025":              "2025",
	"year_2024":              "2024",
	"year_2023":              "2023",
	"year_2022":              "2022",
	"year_2021":              "2021",
	"year_2020":              "2020",
	"year_2010s":             "2010-2019",
	"year_2000s":             "2000-2009",
	"year_1980_1999":         "1980-1999",
	"leaving_soon":           "即将下架",
	"search":                 "搜索",
}

func (p *Plugin) Fetch(req *sdk.FetchRequest) (*sdk.FeedResult, error) {
	switch {
	case req.Route == "/awwrated-netflix/list":
		return fetchList(req.ChannelID, req.Params)
	case req.Route == "/awwrated-netflix/search" || strings.HasPrefix(req.Route, "/awwrated-netflix/search"):
		query := strings.TrimSpace(req.Params["query"])
		if query == "" {
			return nil, fmt.Errorf("missing query parameter")
		}
		return fetchList(req.ChannelID, req.Params)
	case req.Route == "/awwrated-netflix/detail/:id":
		id := strings.TrimSpace(req.Params["id"])
		if id == "" {
			return nil, fmt.Errorf("missing id parameter")
		}
		return fetchDetail(id)
	default:
		return nil, fmt.Errorf("unknown route: %s", req.Route)
	}
}

func fetchList(channelID string, params map[string]string) (*sdk.FeedResult, error) {
	// page=1 为刷新/首屏：忽略上一轮残留的 pageToken（与 jimeng seenIds 约定一致）
	after := ""
	if pageNum(params) > 1 {
		after = strings.TrimSpace(params["pageToken"])
	}
	where := buildWhere(params)

	query := `
query VideosQuery($first: Int, $after: String, $where: RootQueryToPostConnectionWhereArgs) {
  posts(first: $first, after: $after, where: $where) {
    pageInfo { hasNextPage endCursor }
    edges {
      node {
        databaseId
        awwratedData {
          awwratedId
          averageReview
          totalVotes
          poster
          releaseDate
          leavingSoonDate
          edchoice
          genre
          country
          lang {
            zhtw { title }
            zhcn { title }
            en { title }
            jp { title }
          }
        }
      }
    }
  }
}`

	variables := map[string]interface{}{
		"first": pageSize,
		"where": where,
	}
	if after != "" {
		variables["after"] = after
	}

	var resp listResponse
	if err := graphql(query, variables, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("graphql: %s", resp.Errors[0].Message)
	}

	items := make([]sdk.FeedItem, 0, len(resp.Data.Posts.Edges))
	for _, edge := range resp.Data.Posts.Edges {
		if item, ok := toFeedItem(edge.Node); ok {
			items = append(items, item)
		}
	}

	title := channelLabels[channelID]
	if title == "" {
		title = pluginName
	}
	if q := strings.TrimSpace(params["query"]); q != "" {
		title = fmt.Sprintf("搜索：%s", q)
	}

	result := &sdk.FeedResult{
		Title:       title,
		Description: pluginName + " 评分",
		Items:       items,
	}
	if resp.Data.Posts.PageInfo.HasNextPage && resp.Data.Posts.PageInfo.EndCursor != "" {
		result.HasMore = true
		result.Next = copyParams(params)
		result.Next["page"] = strconv.Itoa(pageNum(params) + 1)
		result.Next["pageToken"] = resp.Data.Posts.PageInfo.EndCursor
	}
	return result, nil
}

func buildWhere(params map[string]string) map[string]interface{} {
	where := map[string]interface{}{
		"categoryId": categoryID,
	}

	if tag := strings.TrimSpace(params["tag"]); tag != "" {
		where["tagSlugIn"] = []string{tag}
	}
	if q := strings.TrimSpace(params["query"]); q != "" {
		where["search"] = q
	}

	var metaArray []map[string]interface{}
	switch strings.TrimSpace(params["filter"]) {
	case "leaving_soon":
		metaArray = append(metaArray, map[string]interface{}{
			"key":     "leaving_soon_date",
			"compare": "GREATER_THAN_OR_EQUAL_TO",
			"value":   strconv.FormatInt(host.NowUnix(), 10),
		})
	case "edchoice":
		metaArray = append(metaArray, map[string]interface{}{
			"key":     "edchoice",
			"compare": "EQUAL_TO",
			"value":   "1",
		})
	}

	if metaType := strings.TrimSpace(params["metaType"]); metaType != "" {
		metaArray = append(metaArray, map[string]interface{}{
			"key":     "type",
			"compare": "EQUAL_TO",
			"value":   metaType,
		})
	}
	if genreLike := strings.TrimSpace(params["genreLike"]); genreLike != "" {
		metaArray = append(metaArray, map[string]interface{}{
			"key":     "genre",
			"compare": "LIKE",
			"value":   genreLike,
		})
	}
	if country := strings.TrimSpace(params["country"]); country != "" {
		metaArray = append(metaArray, map[string]interface{}{
			"key":     "country",
			"compare": "LIKE",
			"value":   country,
		})
	}

	// 近期：release_date >= now - N days
	if daysStr := strings.TrimSpace(params["recentDays"]); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
			since := host.NowUnix() - int64(days)*86400
			metaArray = append(metaArray, map[string]interface{}{
				"key":     "release_date",
				"compare": "GREATER_THAN_OR_EQUAL_TO",
				"value":   strconv.FormatInt(since, 10),
			})
		}
	}

	// 年份区间（unix 秒，含首尾）
	if yFrom := strings.TrimSpace(params["yearFrom"]); yFrom != "" {
		metaArray = append(metaArray, map[string]interface{}{
			"key":     "release_date",
			"compare": "GREATER_THAN_OR_EQUAL_TO",
			"value":   yFrom,
		})
	}
	if yTo := strings.TrimSpace(params["yearTo"]); yTo != "" {
		metaArray = append(metaArray, map[string]interface{}{
			"key":     "release_date",
			"compare": "LESS_THAN_OR_EQUAL_TO",
			"value":   yTo,
		})
	}

	if len(metaArray) > 0 {
		metaQuery := map[string]interface{}{"metaArray": metaArray}
		if len(metaArray) > 1 {
			metaQuery["relation"] = "AND"
		}
		where["metaQuery"] = metaQuery
	}

	orderField := strings.TrimSpace(params["orderby"])
	orderDir := strings.ToUpper(strings.TrimSpace(params["order"]))
	if orderDir != "ASC" {
		orderDir = "DESC"
	}
	if orderField != "" {
		item := map[string]interface{}{
			"order": orderDir,
		}
		switch orderField {
		case "DATE", "TITLE", "MODIFIED", "COMMENT_COUNT":
			item["field"] = orderField
		default:
			// meta keys: average_review / total_votes / release_date / popular_value
			item["field"] = "META_KEY"
			item["metaKeyField"] = orderField
		}
		where["orderby"] = []map[string]interface{}{item}
	}

	return where
}

func fetchDetail(id string) (*sdk.FeedResult, error) {
	query := `
query DetailQuery($id: String!) {
  posts(first: 1, where: {
    metaQuery: {
      metaArray: [{ key: "awwrated_id", compare: EQUAL_TO, value: $id }]
    }
  }) {
    edges {
      node {
        databaseId
        title
        content
        awwratedData {
          awwratedId
          averageReview
          totalVotes
          poster
          releaseDate
          leavingSoonDate
          edchoice
          genre
          country
          director
          actors
          rated
          imdbId
          tmdbId
          netflixId
          relatedArticleArray
          trailerArray
          musicOst
          season { currentSeason totalSeason }
          reviews {
            imdb { rating votes }
            douban { rating votes url }
            rottenTomatoes { rating votes url }
            metascore { rating votes url }
            ign { rating url }
            awwRating { rating votes }
          }
          lang {
            zhtw { title }
            zhcn { title }
            en { title }
            jp { title }
          }
        }
      }
    }
  }
}`

	var resp listResponse
	if err := graphql(query, map[string]interface{}{"id": id}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("graphql: %s", resp.Errors[0].Message)
	}
	if len(resp.Data.Posts.Edges) == 0 {
		return nil, fmt.Errorf("not found: %s", id)
	}

	node := resp.Data.Posts.Edges[0].Node
	item, ok := toFeedItem(node)
	if !ok {
		return nil, fmt.Errorf("invalid item: %s", id)
	}
	item.Content = buildDetailContent(node)
	if item.Summary == "" {
		item.Summary = truncateRunes(stripTags(node.Content), 300)
	}

	return &sdk.FeedResult{
		Title: item.Title,
		Items: []sdk.FeedItem{item},
	}, nil
}

func toFeedItem(node postNode) (sdk.FeedItem, bool) {
	data := node.AwwratedData
	id := strings.TrimSpace(data.AwwratedID)
	title := pickTitle(data.Lang, node.Title)
	if id == "" || title == "" {
		return sdk.FeedItem{}, false
	}

	itemURL := fmt.Sprintf("%s/%s/%s", siteBase, platformPath, id)

	var tags []string
	if data.Genre != "" {
		for _, g := range strings.Split(data.Genre, ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				tags = append(tags, g)
			}
		}
	}
	if data.Edchoice {
		tags = append(tags, "编辑推荐")
	}
	if data.LeavingSoonDate != "" {
		tags = append(tags, "即将下架")
	}

	summaryParts := []string{}
	if data.AverageReview != "" {
		part := fmt.Sprintf("aww %s/10", data.AverageReview)
		if data.TotalVotes != "" {
			part += fmt.Sprintf(" · %s 票", data.TotalVotes)
		}
		summaryParts = append(summaryParts, part)
	}
	if data.Genre != "" {
		summaryParts = append(summaryParts, data.Genre)
	}
	if data.Country != "" {
		summaryParts = append(summaryParts, data.Country)
	}

	return sdk.FeedItem{
		ID:          id,
		Title:       title,
		URL:         itemURL,
		Cover:       data.Poster,
		Image:       data.Poster,
		PublishedAt: unixToRFC3339(data.ReleaseDate),
		Summary:     strings.Join(summaryParts, " · "),
		Author:      "awwrated",
		Tags:        tags,
	}, true
}

func buildDetailContent(node postNode) string {
	data := node.AwwratedData
	title := pickTitle(data.Lang, node.Title)
	var sb strings.Builder
	sb.WriteString(`<div class="awwrated-detail">`)

	if data.Poster != "" {
		sb.WriteString(fmt.Sprintf(
			`<img src="%s" style="max-width:220px;border-radius:8px;float:left;margin:0 1rem 1rem 0;" alt="poster"/>`,
			html.EscapeString(data.Poster),
		))
	}

	sb.WriteString(fmt.Sprintf(`<h2 style="margin:0 0 0.5rem;">%s</h2>`, html.EscapeString(title)))

	if alt := altTitles(data.Lang, title); alt != "" {
		sb.WriteString(fmt.Sprintf(`<p style="color:#888;margin:0 0 1rem;">%s</p>`, html.EscapeString(alt)))
	}

	sb.WriteString(`<div style="margin-bottom:1rem;">`)
	if data.AverageReview != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>awwrated:</strong> %s/10`, html.EscapeString(data.AverageReview)))
		if data.TotalVotes != "" {
			sb.WriteString(fmt.Sprintf(` (%s 票)`, html.EscapeString(data.TotalVotes)))
		}
		sb.WriteString(`</p>`)
	}
	if data.Genre != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>类型:</strong> %s</p>`, html.EscapeString(data.Genre)))
	}
	if data.Country != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>地区:</strong> %s</p>`, html.EscapeString(data.Country)))
	}
	if data.Director != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>导演:</strong> %s</p>`, html.EscapeString(data.Director)))
	}
	if data.Actors != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>演员:</strong> %s</p>`, html.EscapeString(data.Actors)))
	}
	if data.Season != nil {
		var parts []string
		if data.Season.CurrentSeason != "" {
			parts = append(parts, "当前季 "+data.Season.CurrentSeason)
		}
		if data.Season.TotalSeason != "" {
			parts = append(parts, "共 "+data.Season.TotalSeason+" 季")
		}
		if len(parts) > 0 {
			sb.WriteString(fmt.Sprintf(`<p><strong>季数:</strong> %s</p>`, html.EscapeString(strings.Join(parts, " · "))))
		}
	}
	if ts := unixToRFC3339(data.ReleaseDate); ts != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>上架:</strong> %s</p>`, html.EscapeString(ts[:10])))
	}
	if ts := unixToRFC3339(data.LeavingSoonDate); ts != "" {
		sb.WriteString(fmt.Sprintf(`<p><strong>即将下架:</strong> %s</p>`, html.EscapeString(ts[:10])))
	}
	if data.Edchoice {
		sb.WriteString(`<p><strong>编辑推荐</strong></p>`)
	}
	sb.WriteString(`</div>`)

	sb.WriteString(buildRatingsHTML(data))

	synopsis := strings.TrimSpace(node.Content)
	if synopsis != "" {
		sb.WriteString(`<div style="clear:both;margin-bottom:1.5rem;">`)
		sb.WriteString(`<p><strong>简介</strong></p>`)
		sb.WriteString(synopsis)
		sb.WriteString(`</div>`)
	}

	sb.WriteString(buildTrailersHTML(data.TrailerArray))
	sb.WriteString(buildArticlesHTML(data.RelatedArticleArray))

	if data.MusicOst != "" {
		sb.WriteString(`<div style="margin-bottom:1.5rem;">`)
		sb.WriteString(`<p><strong>原声</strong></p>`)
		if strings.Contains(data.MusicOst, "spotify.com") {
			sb.WriteString(fmt.Sprintf(
				`<iframe style="border-radius:12px" src="%s" width="100%%" height="152" frameBorder="0" allow="autoplay; clipboard-write; encrypted-media; fullscreen; picture-in-picture" loading="lazy"></iframe>`,
				html.EscapeString(data.MusicOst),
			))
		} else {
			sb.WriteString(fmt.Sprintf(`<p><a href="%s" target="_blank" rel="noopener">%s</a></p>`,
				html.EscapeString(data.MusicOst), html.EscapeString(data.MusicOst)))
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`<div style="margin-top:1rem;padding-top:1rem;border-top:1px solid #eee;">`)
	sb.WriteString(`<p><strong>外链</strong></p><p>`)
	sb.WriteString(fmt.Sprintf(`<a href="%s/%s/%s" target="_blank" rel="noopener">awwrated</a>`, siteBase, platformPath, url.PathEscape(data.AwwratedID)))
	if data.ImdbID != "" {
		sb.WriteString(fmt.Sprintf(` · <a href="https://www.imdb.com/title/%s/" target="_blank" rel="noopener">IMDb</a>`, html.EscapeString(data.ImdbID)))
	}
	if data.TmdbID != "" {
		sb.WriteString(fmt.Sprintf(` · <a href="https://www.themoviedb.org/tv/%s" target="_blank" rel="noopener">TMDB</a>`, html.EscapeString(data.TmdbID)))
	}
	if data.Reviews != nil && data.Reviews.Douban != nil && data.Reviews.Douban.URL != "" {
		sb.WriteString(fmt.Sprintf(` · <a href="%s" target="_blank" rel="noopener">豆瓣</a>`, html.EscapeString(data.Reviews.Douban.URL)))
	}
	if data.NetflixID != "" {
		sb.WriteString(fmt.Sprintf(` · <a href="https://www.netflix.com/title/%s" target="_blank" rel="noopener">Netflix</a>`, html.EscapeString(data.NetflixID)))
	}
	sb.WriteString(`</p></div>`)

	sb.WriteString(`</div>`)
	return sb.String()
}

func buildRatingsHTML(data awwratedData) string {
	if data.Reviews == nil {
		return ""
	}
	type row struct {
		Name   string
		Rating string
		Votes  string
		URL    string
	}
	var rows []row
	r := data.Reviews
	if r.AwwRating != nil && r.AwwRating.Rating != "" {
		rows = append(rows, row{"aww 评分", r.AwwRating.Rating, r.AwwRating.Votes, ""})
	}
	if r.Imdb != nil && r.Imdb.Rating != "" {
		rows = append(rows, row{"IMDb", r.Imdb.Rating, r.Imdb.Votes, ""})
	}
	if r.Douban != nil && r.Douban.Rating != "" {
		rows = append(rows, row{"豆瓣", r.Douban.Rating, r.Douban.Votes, r.Douban.URL})
	}
	if r.RottenTomatoes != nil && r.RottenTomatoes.Rating != "" {
		rows = append(rows, row{"烂番茄", r.RottenTomatoes.Rating, r.RottenTomatoes.Votes, r.RottenTomatoes.URL})
	}
	if r.Metascore != nil && r.Metascore.Rating != "" {
		rows = append(rows, row{"Metascore", r.Metascore.Rating, r.Metascore.Votes, r.Metascore.URL})
	}
	if r.Ign != nil && r.Ign.Rating != "" {
		rows = append(rows, row{"IGN", r.Ign.Rating, "", r.Ign.URL})
	}
	if len(rows) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`<div style="clear:both;margin:1rem 0 1.5rem;">`)
	sb.WriteString(`<p><strong>各站评分</strong></p>`)
	sb.WriteString(`<table style="width:100%;border-collapse:collapse;font-size:14px;">`)
	sb.WriteString(`<tr style="text-align:left;border-bottom:1px solid #eee;"><th style="padding:6px;">来源</th><th style="padding:6px;">评分</th><th style="padding:6px;">票数</th></tr>`)
	for _, row := range rows {
		name := html.EscapeString(row.Name)
		if row.URL != "" {
			name = fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener">%s</a>`, html.EscapeString(row.URL), name)
		}
		votes := row.Votes
		if votes == "" {
			votes = "—"
		}
		sb.WriteString(fmt.Sprintf(
			`<tr style="border-bottom:1px solid #f3f3f3;"><td style="padding:6px;">%s</td><td style="padding:6px;font-weight:600;">%s</td><td style="padding:6px;color:#666;">%s</td></tr>`,
			name, html.EscapeString(row.Rating), html.EscapeString(votes),
		))
	}
	sb.WriteString(`</table></div>`)
	return sb.String()
}

func buildTrailersHTML(raw string) string {
	keys := parseTrailerKeys(raw)
	if len(keys) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<div style="clear:both;margin-bottom:1.5rem;">`)
	sb.WriteString(fmt.Sprintf(`<p><strong>预告片</strong> <span style="opacity:0.65;">(%d)</span></p>`, len(keys)))
	limit := 3
	if len(keys) < limit {
		limit = len(keys)
	}
	for i := 0; i < limit; i++ {
		key := keys[i]
		embed := "https://www.youtube.com/embed/" + key
		watch := "https://www.youtube.com/watch?v=" + key
		sb.WriteString(fmt.Sprintf(
			`<div style="margin-bottom:1rem;"><p><a href="%s" target="_blank" rel="noopener">YouTube</a></p>`+
				`<div style="position:relative;padding-bottom:56.25%%;height:0;overflow:hidden;border-radius:8px;">`+
				`<iframe src="%s" style="position:absolute;top:0;left:0;width:100%%;height:100%%;border:0;" allowfullscreen loading="lazy"></iframe>`+
				`</div></div>`,
			html.EscapeString(watch), html.EscapeString(embed),
		))
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

func buildArticlesHTML(raw string) string {
	articles := parseRelatedArticles(raw)
	if len(articles) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<div style="clear:both;margin-bottom:1.5rem;">`)
	sb.WriteString(`<p><strong>相关图文</strong></p>`)
	for _, a := range articles {
		sb.WriteString(`<div style="margin-bottom:1rem;overflow:hidden;">`)
		if a.Image != "" {
			sb.WriteString(fmt.Sprintf(
				`<img src="%s" style="max-width:160px;border-radius:6px;float:left;margin:0 12px 8px 0;" alt="" loading="lazy"/>`,
				html.EscapeString(a.Image),
			))
		}
		title := a.Title
		if title == "" {
			title = a.URL
		}
		if a.URL != "" {
			sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 6px;"><a href="%s" target="_blank" rel="noopener">%s</a></p>`,
				html.EscapeString(a.URL), html.EscapeString(title)))
		} else {
			sb.WriteString(fmt.Sprintf(`<p style="margin:0 0 6px;font-weight:600;">%s</p>`, html.EscapeString(title)))
		}
		if a.Summary != "" {
			sb.WriteString(fmt.Sprintf(`<p style="margin:0;color:#555;line-height:1.6;">%s</p>`, html.EscapeString(a.Summary)))
		}
		sb.WriteString(`<div style="clear:both;"></div></div>`)
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

func parseTrailerKeys(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	var items []struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if k := strings.TrimSpace(it.Key); k != "" {
			out = append(out, k)
		}
	}
	return out
}

type relatedArticle struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Image   string `json:"image"`
	Summary string `json:"summary"`
	Cover   string `json:"cover"`
	Link    string `json:"link"`
	Desc    string `json:"description"`
}

func parseRelatedArticles(raw string) []relatedArticle {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	var items []relatedArticle
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	out := make([]relatedArticle, 0, len(items))
	for _, a := range items {
		if a.Image == "" {
			a.Image = a.Cover
		}
		if a.URL == "" {
			a.URL = a.Link
		}
		if a.Summary == "" {
			a.Summary = a.Desc
		}
		if a.Title == "" && a.URL == "" && a.Image == "" {
			continue
		}
		out = append(out, a)
	}
	return out
}

func pickTitle(lang *langBlock, fallback string) string {
	if lang != nil {
		for _, t := range []string{
			lang.Zhtw.Title, lang.Zhcn.Title, lang.En.Title, lang.Jp.Title,
		} {
			if strings.TrimSpace(t) != "" {
				return strings.TrimSpace(t)
			}
		}
	}
	return strings.TrimSpace(fallback)
}

func altTitles(lang *langBlock, primary string) string {
	if lang == nil {
		return ""
	}
	seen := map[string]bool{primary: true}
	var alts []string
	for _, t := range []string{lang.En.Title, lang.Jp.Title, lang.Zhcn.Title} {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		alts = append(alts, t)
	}
	return strings.Join(alts, " / ")
}

func unixToRFC3339(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return ""
	}
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil || sec <= 0 {
		return ""
	}
	return time.Unix(sec, 0).UTC().Format(time.RFC3339)
}

func truncateRunes(s string, n int) string {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) <= n {
		return string(rs)
	}
	return string(rs[:n]) + "…"
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return html.UnescapeString(strings.Join(strings.Fields(b.String()), " "))
}

func graphql(query string, variables map[string]interface{}, out interface{}) error {
	payload, err := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return err
	}
	body, status, err := host.HTTPPost(graphqlURL, map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
		"Origin":       "https://awwrated.com",
		"Referer":      "https://awwrated.com/zh-tw/" + platformPath,
		"User-Agent":   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	}, string(payload))
	if err != nil {
		return err
	}
	if status != 200 {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return fmt.Errorf("graphql HTTP %d: %s", status, snippet)
	}
	return json.Unmarshal(body, out)
}

type listResponse struct {
	Data   listData     `json:"data"`
	Errors []graphQLErr `json:"errors"`
}

type graphQLErr struct {
	Message string `json:"message"`
}

type listData struct {
	Posts struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Edges []struct {
			Node postNode `json:"node"`
		} `json:"edges"`
	} `json:"posts"`
}

type postNode struct {
	DatabaseID   int          `json:"databaseId"`
	Title        string       `json:"title"`
	Content      string       `json:"content"`
	AwwratedData awwratedData `json:"awwratedData"`
}

type awwratedData struct {
	AwwratedID          string        `json:"awwratedId"`
	AverageReview       string        `json:"averageReview"`
	TotalVotes          string        `json:"totalVotes"`
	Poster              string        `json:"poster"`
	ReleaseDate         string        `json:"releaseDate"`
	LeavingSoonDate     string        `json:"leavingSoonDate"`
	Edchoice            bool          `json:"edchoice"`
	Genre               string        `json:"genre"`
	Country             string        `json:"country"`
	Director            string        `json:"director"`
	Actors              string        `json:"actors"`
	Rated               string        `json:"rated"`
	ImdbID              string        `json:"imdbId"`
	TmdbID              string        `json:"tmdbId"`
	NetflixID           string        `json:"netflixId"`
	RelatedArticleArray string        `json:"relatedArticleArray"`
	TrailerArray        string        `json:"trailerArray"`
	MusicOst            string        `json:"musicOst"`
	Season              *seasonInfo   `json:"season"`
	Reviews             *reviewsBlock `json:"reviews"`
	Lang                *langBlock    `json:"lang"`
}

type seasonInfo struct {
	CurrentSeason string `json:"currentSeason"`
	TotalSeason   string `json:"totalSeason"`
}

type reviewsBlock struct {
	Imdb           *siteRating `json:"imdb"`
	Douban         *siteRating `json:"douban"`
	RottenTomatoes *siteRating `json:"rottenTomatoes"`
	Metascore      *siteRating `json:"metascore"`
	Ign            *siteRating `json:"ign"`
	AwwRating      *siteRating `json:"awwRating"`
}

type siteRating struct {
	Rating string `json:"rating"`
	Votes  string `json:"votes"`
	URL    string `json:"url"`
}

type langBlock struct {
	Zhtw langTitle `json:"zhtw"`
	Zhcn langTitle `json:"zhcn"`
	En   langTitle `json:"en"`
	Jp   langTitle `json:"jp"`
}

type langTitle struct {
	Title string `json:"title"`
}

func pageNum(params map[string]string) int {
	page := 1
	if s := strings.TrimSpace(params["page"]); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			page = n
		}
	}
	return page
}

func copyParams(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}
