# 央视网新闻

从 [央视网新闻频道](https://news.cctv.com/) 抓取图文新闻，支持多栏目列表、分页与文章详情。

视频稿（`tv.cctv.com`、`VIDE` 链接）与图集（`photo.cctv.com`、`PHOA` 链接）在列表与详情中均会跳过。

## 内置频道

| section | 频道 |
|---------|------|
| `news` | 新闻 |
| `china` | 国内 |
| `world` | 国际 |
| `society` | 社会 |
| `law` | 法治 |
| `ent` | 文娱 |
| `tech` | 科技 |
| `life` | 生活 |

列表分页参数 `page` 从 `1` 开始，对应 `cmsdatainterface/page/{section}_{page}.jsonp`；存在下一页时返回 `hasMore` 与 `next.page`。

## 本地测试

```bash
# 国内第 1 页
make test-native CHANNEL=china PARAMS='{"section":"china","page":"1"}'

# 文章详情（id 为完整 URL）
echo '{"action":"fetch","data":{"channelId":"china","route":"/cctv/detail/:id","params":{"id":"https://news.cctv.com/2026/07/07/ARTIBSKr86fXTvHjw0bOJ3Uk260706.shtml"}}}' | go run .
```

## 构建

```bash
make build
make package
```

## 技术说明

- 列表取自 `news.cctv.com/2019/07/gaiban/cmsdatainterface/page/{section}_{page}.jsonp`
- 详情正文解析页面内嵌的 `contentdate` 变量
- 内嵌视频占位符会被移除，不解析播放器

## 注意事项

- 建议 `refreshInterval` 不低于 1800 秒
- 本插件为非官方实现，请遵守央视网使用条款与版权声明
