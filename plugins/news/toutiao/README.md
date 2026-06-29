# 今日头条 (toutiao)

今日头条资讯插件，支持热榜与分类频道信息流、图文详情解析。

## 频道

| 频道 | 说明 |
|------|------|
| `hot-board` | 实时热榜 |
| `news-hot` | 要闻 |
| `news-tech` | 科技 |
| `news-finance` | 财经 |
| `news-sports` | 体育 |
| `news-entertainment` | 娱乐 |

## 数据源

- 分类列表：`https://www.toutiao.com/api/pc/feed/`
- 热榜：`https://www.toutiao.com/hot-event/hot-board/`（详情复用移动端 `m.toutiao.com/i{id}/info/`，约 2/3 话题有独立正文，其余展示跳转提示）
- 文章详情：`https://m.toutiao.com/i{id}/info/`（移动端接口，规避 PC 反爬）

## 本地测试

```bash
make test-native
make test-native CHANNEL=hot-board ROUTE=/toutiao/hot-board PARAMS='{}'
make test-native ROUTE=/toutiao/detail/7656813195675402786 PARAMS='{"id":"7656813195675402786"}'
```

## 限制

- **不支持翻页**：官网首页推荐流使用 `/api/pc/list/feed`，需 `msToken`、`a_bogus` 及浏览器 Cookie，WASM 插件无法稳定复现；当前使用的 `/api/pc/feed/` 虽可拉首屏，但 `max_behot_time` 游标翻页会大量重复，已禁用分页
- **热榜详情**：通过移动端接口获取；部分话题无独立正文，详情页会提示跳转头条话题页
- 搜索、个性化推荐等接口需字节系鉴权，暂未支持
- 非官方 API，可能随平台反爬策略变更而失效
