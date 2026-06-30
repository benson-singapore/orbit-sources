# 纽约时报中文网

从 [纽约时报中文网](https://m.cn.nytimes.com/) 抓取简体新闻，支持分类频道列表与文章详情阅读。

## 内置频道

| section | 频道 |
|---------|------|
| `home` | 首页（默认） |
| `world` | 国际 |
| `china` | 中国 |
| `business` | 商业与经济 |
| `technology` | 科技 |
| `science` | 科学 |
| `health` | 健康 |
| `education` | 教育 |
| `culture` | 文化 |
| `style` | 风尚 |
| `travel` | 旅游 |
| `real-estate` | 房地产 |
| `opinion` | 观点与评论 |

「镜头」频道页面为图集结构，当前不支持抓取，已排除。

各频道约返回最新 20 条，**不支持翻页**。

## 本地测试

```bash
# 首页列表
make test-native CHANNEL=home PARAMS='{"section":"home"}'

# 中国频道
make test-native CHANNEL=china PARAMS='{"section":"china"}'

# 文章详情
echo '{"action":"fetch","data":{"channelId":"china","route":"/nyt-chinese/detail/:id","params":{"id":"china/20260627/china-plane-crash-beijing"}}}' | go run .
```

## 构建

```bash
make build
make package
```

## 注意事项

- 部分频道列表会混入其他栏目文章，与官网展示一致
- 科技等频道偶发标题为空，插件会从 URL slug 兜底生成标题
- 建议 `refreshInterval` 不低于 1800 秒
