# TV-如意资源 Plugin

基于苹果 CMS 资源站开放接口的视频插件，默认接入 `https://cj.rycjapi.com/api.php/provide/vod`。

采用 **三层 chapters 导航**（适合剧集/综艺等多集内容）：

```
feed（作品列表）→ chapters（选集列表）→ episode（单集播放）
```

## 功能

- 最新更新（全站）
- 按二级分类浏览，如「电影 · 动作片」「剧集 · 国产剧」
- 关键词搜索
- 点击作品进入选集列表，再点击单集直接播放
- 单集页默认使用 `<video>` 播放 m3u8/mp4
- 多线路时在播放页内切换（如 rym3u8 / ruyi）

## 配置

可选变量：

- `apiURL`: 资源站 API 地址
- `siteName`: 页面展示名称

默认无需额外配置即可使用。

## 本地测试

```bash
# 分类列表
CHANNEL=series-cn ROUTE=/rycjapi/list PARAMS='{"t":"13","page":"1"}' make test-native

# 选集列表
CHANNEL=series-cn ROUTE=/rycjapi/chapters/:id PARAMS='{"id":"79534"}' make test-native

# 单集播放（第 1 集，默认线路）
CHANNEL=series-cn ROUTE=/rycjapi/episode/:chapterId PARAMS='{"id":"79534","chapterId":"0"}' make test-native

# 搜索
CHANNEL=search ROUTE=/rycjapi/search PARAMS='{"query":"流浪地球","page":"1"}' make test-native
```

## 接口说明

使用的是常见资源站 JSON API：

- 列表: `?ac=list&pg=1`
- 二级分类: `?ac=list&t=6&pg=1`（`t` 为二级 `type_id`）
- 搜索: `?ac=list&wd=关键词&pg=1`
- 作品详情/选集: `?ac=videolist&ids=视频ID`（解析 `vod_play_from` / `vod_play_url`）
