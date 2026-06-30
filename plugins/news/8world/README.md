# 8视界

从 [8视界 (8world)](https://www.8world.com/) 抓取中文文字新闻，支持分类列表、分页与文章详情。

视频类新闻（`data-media="Video"`、`/videos/`、`/in-depth/` 等）在列表与详情中均会跳过，不纳入 feed。

## 内置频道

| section | 频道 |
|---------|------|
| `realtime` | 即时 |
| `singapore` | 新加坡 |
| `southeast-asia` | 东南亚 |
| `greater-china` | 中港台 |
| `world` | 国际 |
| `finance` | 财经 |
| `sports` | 体育 |

列表分页参数 `page` 从 `0` 开始；存在下一页时返回 `hasMore` 与 `next.page`。

## 本地测试

```bash
# 新加坡第 1 页
make test-native CHANNEL=singapore PARAMS='{"section":"singapore","page":"0"}'

# 文章详情（id 为完整 URL，或数字 ID 经 /node/{id} 跳转）
echo '{"action":"fetch","data":{"channelId":"singapore","route":"/8world/detail/:id","params":{"id":"https://www.8world.com/singapore/kpod-driver-crash-jail-3198996"}}}' | go run .
```

## 构建

```bash
make build
make package
```

## 技术说明

- 列表解析 `article.contour` 卡片，跳过 `data-media="Video"`
- 详情取自 JSON-LD `NewsArticle` 与 `.article-content` 正文
- 含 Brightcove 播放器或 `videoad` 配置的页面视为视频稿，不抓取

## 注意事项

- 建议 `refreshInterval` 不低于 1800 秒
- 本插件为非官方实现，请遵守 8world 使用条款
