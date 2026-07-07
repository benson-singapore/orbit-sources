# 澎湃新闻

从 [澎湃新闻](https://www.thepaper.cn/) 抓取图文新闻，支持热榜、栏目列表、游标分页与文章详情。

## 内置频道

| channel | 说明 |
|---------|------|
| `hot` | 澎湃热榜 |
| `finance` | 澎湃财讯 |
| `morning` | 早晚报 |
| `china-politics` | 中国政库 |
| `headline` | 浦江头条 |
| `opinion` | 澎湃评论 |
| `international` | 澎湃国际 |
| `onsite` | 直击现场 |
| `anti-corruption` | 打虎记 |

栏目列表使用 `startTime` 游标分页；存在下一页时返回 `hasMore` 与 `next.startTime`。

## 本地测试

```bash
# 澎湃热榜
make test-native CHANNEL=hot PARAMS='{"section":"hotNews"}'

# 中国政库第 1 页
make test-native CHANNEL=china-politics ROUTE=/thepaper/list PARAMS='{"node_id":"25462"}'

# 文章详情（id 为 contId 或完整 URL）
echo '{"action":"fetch","data":{"channelId":"hot","route":"/thepaper/detail/:id","params":{"id":"33530511"}}}' | go run .
```

## 构建

```bash
make build
make package
```

## 技术说明

- 热榜取自 `cache.thepaper.cn/contentapi/wwwIndex/rightSidebar`
- 栏目列表取自 `api.thepaper.cn/contentapi/nodeCont/getByNodeIdPortal`
- 详情解析文章页内嵌的 `__NEXT_DATA__` → `detailData.contentDetail`

## 注意事项

- API 请求需携带 `Origin` / `Referer`，否则可能被 WAF 拦截
- 建议 `refreshInterval` 不低于 1800 秒
- 本插件为非官方实现，请遵守澎湃新闻使用条款与版权声明
