# 红果果短剧 Plugin

抓取 [红果果短剧网](https://www.hongguoguo.tv/) 的短剧列表、分集与播放地址。

站点基于苹果 CMS（MacCMS）模板，列表走分类 HTML，分集与播放地址解析播放页内嵌的 `player_aaaa`。

采用 **三层 chapters 导航**：

```
feed（作品列表）→ chapters（选集列表）→ episode（单集播放）
```

## 功能

- 精选 / 穿越 / 古装 / 逆袭 / 都市 分类浏览
- 福利短剧（`status: disabled`，默认隐藏，可在频道设置中开启）
- 关键词搜索
- 分页加载
- 选集列表（支持「全集」单条或按集拆分）
- 单集页解析明文 / Base64 的 m3u8、mp4 地址

## 配置

可选变量：

- `baseURL`: 站点根地址，默认 `https://www.hongguoguo.tv`

## 本地测试

```bash
# 分类列表
CHANNEL=jingxuan ROUTE=/hongguoguo/list/:type PARAMS='{"type":"jingxuanduanju","page":"1"}' make test-native

# 选集列表
CHANNEL=jingxuan ROUTE=/hongguoguo/chapters/:id PARAMS='{"id":"1122"}' make test-native

# 单集播放（nid=1）
CHANNEL=jingxuan ROUTE=/hongguoguo/episode/:chapterId PARAMS='{"id":"1122","chapterId":"1"}' make test-native

# 搜索
CHANNEL=search ROUTE=/hongguoguo/search/:query PARAMS='{"query":"扬名立万","page":"1"}' make test-native
```

## URL 约定

| 用途 | 路径 |
|------|------|
| 分类 | `/vod/show/id/{type}/page/{n}.html` |
| 搜索 | `/vod/search/wd/{query}/page/{n}.html` |
| 播放 | `/vod/play/id/{id}/sid/{sid}/nid/{nid}.html` |
