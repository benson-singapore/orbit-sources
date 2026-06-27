# Pixabay 图片与视频插件

[Pixabay](https://pixabay.com/zh/) 提供免版权的图片与视频资源。本插件通过官方 API 订阅分类流、编辑精选与关键词搜索。

## 前置配置

在 [Pixabay API 文档](https://pixabay.com/api/docs/) 注册并获取 API Key，填入 Orbit 插件设置的 `apiKey` 字段。

可选变量：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `lang` | `zh` | 搜索语言（zh、en、ja、ko 等） |
| `safesearch` | `true` | 安全搜索，仅返回适合全年龄内容 |

## 频道配置

所有频道通过 `features.pagination` 声明翻页能力：

```json
"features": {
  "pagination": {
    "style": "offset",
    "param": "page",
    "default": "1",
    "sizeParam": "size",
    "defaultSize": 50
  }
}
```

### 图片频道

| 频道 ID | 频道名称 | 主要参数 |
|--------|--------|--------|
| `editors-choice-photo` | 编辑精选 · 照片 | `editors_choice=true` |
| `latest-photo` | 最新照片 | `order=latest` |
| `popular-photo` | 热门照片 | `order=popular` |
| `illustration` | 插画 | `image_type=illustration` |
| `vector` | 矢量图 | `image_type=vector` |
| `photo-nature` | 自然 | `category=nature` |
| `photo-animals` | 动物 | `category=animals` |
| `photo-travel` | 旅行 | `category=travel` |
| `photo-buildings` | 建筑 | `category=buildings` |
| `photo-places` | 地点 | `category=places` |
| `photo-food` | 美食 | `category=food` |
| `photo-sports` | 运动 | `category=sports` |
| `photo-people` | 人物 | `category=people` |
| `photo-backgrounds` | 背景 | `category=backgrounds` |
| `photo-fashion` | 时尚 | `category=fashion` |
| `photo-science` | 科学 | `category=science` |
| `photo-computer` | 科技 | `category=computer` |
| `photo-business` | 商业 | `category=business` |
| `photo-music` | 音乐 | `category=music` |
| `photo-transportation` | 交通 | `category=transportation` |
| `photo-health` | 健康 | `category=health` |
| `photo-education` | 教育 | `category=education` |
| `photo-feelings` | 情感 | `category=feelings` |
| `photo-religion` | 宗教 | `category=religion` |
| `photo-industry` | 工业 | `category=industry` |
| `photo-horizontal` | 横向照片 | `orientation=horizontal` |
| `photo-vertical` | 纵向照片 | `orientation=vertical` |

### 视频频道

| 频道 ID | 频道名称 | 主要参数 |
|--------|--------|--------|
| `editors-choice-video` | 编辑精选 · 视频 | `editors_choice=true` |
| `latest-video` | 最新视频 | `order=latest` |
| `popular-video` | 热门视频 | `order=popular` |
| `video-nature` | 自然视频 | `category=nature` |
| `video-travel` | 旅行视频 | `category=travel` |
| `video-animals` | 动物视频 | `category=animals` |

### 搜索频道

| 频道 ID | 频道名称 | 路由 |
|--------|--------|------|
| `search` | 搜索 | `/pixabay/search/:query` |

搜索时可通过 `media` 参数切换结果类型：`image`（默认）或 `video`。

## 查询参数

频道 `params` 中的 Pixabay API 参数会原样透传，常用参数如下：

| 参数 | 默认值 | 说明 |
|-----|------|------|
| `page` | `1` | 页码，从 1 开始 |
| `size` | `50` | 每页数量，映射为 API 的 `per_page`（3–200） |
| `q` / `query` | — | 搜索关键词（搜索频道使用 `query`） |
| `image_type` | `all` | 图片类型：`all`、`photo`、`illustration`、`vector` |
| `video_type` | `all` | 视频类型：`all`、`film`、`animation` |
| `category` | — | 分类（nature、animals、travel 等） |
| `orientation` | `all` | 方向：`horizontal`、`vertical` |
| `colors` | — | 颜色筛选（red、blue、green 等） |
| `editors_choice` | `false` | 编辑精选：`true` / `false` |
| `order` | `popular` | 排序：`popular` 或 `latest` |
| `min_width` | `0` | 最小宽度 |
| `min_height` | `0` | 最小高度 |

## 本地测试

```bash
# 自然分类
make test-native-pixabay CHANNEL=photo-nature ROUTE=/pixabay/images PARAMS='{"category":"nature","page":"1","size":"5"}' API_KEY=YOUR_KEY

# 关键词搜索
echo '{"action":"fetch","data":{"channelId":"search","route":"/pixabay/search/:query","params":{"query":"sunset","page":"1","size":"5"},"vars":{"apiKey":"YOUR_KEY"}}}' | go run .

# 视频
make test-native-pixabay CHANNEL=video-nature ROUTE=/pixabay/videos PARAMS='{"category":"nature","page":"1","size":"5"}' API_KEY=YOUR_KEY
```

定时刷新拉取第一页并落库；加载更多时通过 `page` 翻页。每页条数可通过 `size` 参数调整（3–200，默认 50）。

## 数据字段

每条图片包含：标题、作者、标签、高清图链接（`Cover` / `Image` 均使用 `largeImageURL`，最大边约 1280px）、Pixabay 详情页 URL、浏览/下载/点赞统计。

每条视频包含：标题、作者、标签、视频直链（`content`）、封面缩略图、时长与统计信息。

## API 限制

- 免费账户默认每分钟 100 次请求
- 单次查询最多返回 500 条结果（`totalHits` 上限）
- 图片 URL 有效期约 24 小时，仅供临时展示；长期使用请下载到自有服务器
- 官方要求：展示搜索结果时需注明图片来自 Pixabay
