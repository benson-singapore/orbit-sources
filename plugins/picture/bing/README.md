# Bing Wallpaper Plugin

Bing 每日精选高清壁纸，按 [Bing Wallpaper Gallery](https://bing.gifposter.com/) 的分类方式组织频道。

## Features

- 最新壁纸（降序 / 升序 / 幻灯片）
- 热门壁纸（降序 / 升序 / 幻灯片）
- 月度归档
- 手机壁纸（最新 / 热门）
- 分页加载与详情页

## Channels

| ID | Label | Source |
|---|---|---|
| `new-desc-classic` | 最新壁纸 | `/list/new/desc/classic.html` |
| `new-asc-classic` | 最新壁纸（升序） | `/list/new/asc/classic.html` |
| `new-desc-slide` | 最新壁纸（幻灯片） | `/list/new/desc/slide.html` |
| `hot-desc-classic` | 热门壁纸 | `/list/hot/desc/classic.html` |
| `hot-asc-classic` | 热门壁纸（升序） | `/list/hot/asc/classic.html` |
| `hot-desc-slide` | 热门壁纸（幻灯片） | `/list/hot/desc/slide.html` |
| `phone-new` | 手机壁纸（最新） | `/phone.html` |
| `phone-hot` | 手机壁纸（热门） | `/phone/hot/desc.html` |

## Pagination

列表与手机壁纸频道使用 `page` 参数：

- 桌面列表：`?p=2`
- 手机壁纸：`?page=2`

## Item Fields

- `id` - 页面 slug（如 `column-3669-shark-awareness-day`）
- `title` - 壁纸标题
- `url` - 详情页链接
- `image` - 高清图片 URL（1920x1080 或 608x1080）
- `cover` - 缩略图
- `summary` - 摘要或浏览量
- `published_at` - 发布日期（RFC3339）

## Testing

```bash
# 最新壁纸
make test-native

# 热门壁纸
echo '{"action":"fetch","data":{"channelId":"hot-desc-classic","route":"/bing/list","params":{"category":"hot","sort":"desc","layout":"classic","page":"1"}}}' | go run .

# 构建 WASM
make build
```
