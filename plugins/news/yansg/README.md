# 新加坡眼插件

从 [新加坡眼](https://www.yan.sg/) 抓取即时新闻，支持新加坡、中国与东南亚三个分类。

## 频道

| channel id | 分类 | 网站路径 |
|------------|------|----------|
| `singapore` | 新加坡 | `/category/news/sgnews/` |
| `china` | 中国 | `/category/news/chinanews/` |
| `southeast-asia` | 东南亚 | `/category/news/southeastasianews/` |

## 路由

- 列表：`/yansg/list`，参数 `category`、`page`
- 详情：`/yansg/detail/:id`，`id` 为文章完整 URL

## 本地测试

```bash
make try-yansg
CHANNEL=china make try-yansg
PAGE=2 make try-yansg
```
