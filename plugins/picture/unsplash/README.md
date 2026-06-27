# Unsplash 图片插件

[Unsplash](https://unsplash.com/) 高质量免版权摄影。本插件通过站内 **napi** 接口拉取主题分类与搜索结果。

## 前置配置

在 Orbit 插件设置中填入 `cookie`（必填）：

1. 浏览器登录 [unsplash.com](https://unsplash.com/)
2. 打开任意主题页（如 `/t/nature`），在开发者工具 Network 中找到 `napi/topics/.../photos` 请求
3. 复制请求头中的完整 `Cookie` 值（需包含 `un_sesh`、`uuid` 等字段）

Cookie 会过期，失效后需重新复制。

## 接口说明

与抓取文档一致：

| 类型 | 接口 |
|------|------|
| 主题分类 | `GET https://unsplash.com/napi/topics/{topic}/photos?page=1&per_page=20` |
| 分页 | `page` + `per_page`（默认 20） |
| 关键词搜索 | `GET https://unsplash.com/napi/search/photos?page=1&per_page=20&query=...` |

主题接口返回 JSON 数组；搜索接口返回 `{ total, total_pages, results }`。

## 频道

频道 ID 与 Unsplash Topic slug 一致：

| 频道 ID | 名称 |
|--------|------|
| `nature` | 自然 |
| `animals` | 动物 |
| `travel` | 旅行 |
| `wallpapers` | 壁纸 |
| `architecture-interior` | 建筑与室内 |
| `people` | 人物 |
| `food-drink` | 美食 |
| `film` | 胶片 |
| `textures-patterns` | 纹理与图案 |
| `street-photography` | 街拍 |
| `business-work` | 商业与工作 |
| `fashion-beauty` | 时尚与美妆 |
| `3d-renders` | 3D 渲染 |
| `experimental` | 实验摄影 |
| `arts-culture` | 艺术与文化（搜索 `arts culture`，topic 已下线） |
| `current-events` | 时事（默认禁用） |
| `spring` | 春天 |
| `optimism` | 乐观 |
| `search` | 搜索 |

## 查询参数

| 参数 | 默认值 | 说明 |
|-----|------|------|
| `page` | `1` | 页码 |
| `size` | `20` | 每页数量，映射为 `per_page` |
| `topic` | — | 主题 slug |
| `query` | — | 搜索关键词 |

## 本地测试

```bash
# 自然主题
make test-native-unsplash COOKIE='你的Cookie'

# 关键词搜索
echo '{"action":"fetch","data":{"channelId":"search","route":"/unsplash/search/:query","params":{"query":"sea life","page":"1","size":"5"},"vars":{"cookie":"你的Cookie"}}}' | go run .

# 第二页
make test-native-unsplash CHANNEL=nature PARAMS='{"topic":"nature","page":"2","size":"20"}' COOKIE='你的Cookie'
```

## 数据字段

每条图片包含：标题（`description` / `alt_description`）、作者、尺寸与点赞数、高清图（`Cover` / `Image` 使用 `urls.regular`）、Unsplash 详情页 URL、已审核通过的主题标签。
