# BBC News 中文

从 [BBC News 中文](https://www.bbc.com/zhongwen/simp) 抓取简体新闻，支持主页与分类频道列表、文章详情阅读。

## 内置频道

| section | 频道 | 说明 |
|---------|------|------|
| `world` | 国际 | 国际新闻 |
| `china` | 中国 | 中国新闻（默认） |
| `hong-kong` | 香港 | 香港新闻 |
| `taiwan` | 台湾 | 台湾新闻 |
| `uk` | 英国 | 英国新闻 |
| `business` | 财经 | 金融财经 |
| `video` | 影片 | 视频新闻 |

分类频道支持 `page` 分页（从第 2 页起追加 `?page=N`）。详情 `content` 仅含正文段落，封面与摘要分别由 `cover` / `summary` 字段提供。

## 本地测试

```bash
# 中国分类第 1 页
make test-native CHANNEL=china PARAMS='{"section":"china","page":"1"}'

# 文章详情
echo '{"action":"fetch","data":{"channelId":"home","route":"/bbc-chinese/detail/:id","params":{"id":"crm0lenk8n8o"}}}' | go run .
```

## 构建

```bash
make build
make package
```

或在仓库根目录：

```bash
make build PLUGIN=bbc-chinese
make orbit PLUGIN=bbc-chinese
```

## 技术说明

- 解析页面内嵌的 `__NEXT_DATA__` JSON，不依赖公开 API
- 列表取自 `pageData.curations[].summaries[]`
- 详情正文由 `content.model.blocks` 渲染为 HTML
- 图片经 `ichef.bbci.co.uk` CDN，`{width}` 占位符替换为 480px

## 注意事项

- BBC 在中国大陆可能无法直接访问，需 Orbit Runtime 代理或用户侧网络支持
- 建议 `refreshInterval` 不低于 1800 秒，避免频繁请求
- 本插件为非官方实现，请遵守 BBC 使用条款
