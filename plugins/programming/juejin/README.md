# 掘金插件使用指南

## 概述
本插件从掘金开发者社区抓取技术文章，支持热榜和多个技术分类的内容源。

## 内置频道

### 支持的分类
插件目前支持以下 6 个频道：

| 频道ID | 频道名称 | 说明 |
|--------|---------|------|
| `trending` | 热榜 | 掘金全站热门文章（默认） |
| `category-frontend` | 前端 | 前端技术文章 |
| `category-backend` | 后端 | 后端技术文章 |
| `category-ai` | 人工智能 | AI 相关技术文章 |
| `category-freebie` | 开发工具 | 开发工具与效率 |
| `category-general` | 综合 | 首页综合推荐 |

## 自定义频道

如需添加其他分类或自定义频道，请编辑 manifest.json 配置中的 `channels` 数组。

### 示例：添加新的分类频道

```json
{
  "id": "category-mobile",
  "label": "移动开发",
  "route": "/juejin/category/:category",
  "params": { "category": "mobile" }
}
```

### 添加步骤

1. 编辑 manifest.json 文件
2. 在 `config.channels` 数组中添加新对象
3. 设置唯一的 `id`（用于内部标识）
4. 设置用户可见的 `label`（频道显示名称）
5. 选择合适的 `route`：
   - `/juejin/trending` - 用于热榜类频道（无需 params）
   - `/juejin/category/:category` - 用于分类频道（需要 params.category）
6. 在 `params` 中设置对应的参数值
7. 保存并重新加载插件

## 参数说明

### 路由参数

- **`:category`** - 分类标识（仅用于 `/juejin/category/:category` 路由）
  - 取值范围：`frontend`、`backend`、`ai`、`freebie`、`general`
  - 可扩展：支持自定义分类 ID（需对应掘金 API 中的 category_id）

### 其他配置

- **`refreshInterval`** - 刷新间隔（秒），默认 1800 秒（30分钟）
- **`defaultChannel`** - 默认频道 ID，应该是 channels 数组中存在的 id（当前：trending）
- **`timeoutMs`** - 超时时间（毫秒），默认 120000ms（2分钟）
- **`maxMemoryMB`** - WASM 内存限制（MB），默认 64MB

## 配置示例

### 完整频道配置

```json
"channels": [
  {
    "id": "trending",
    "label": "热榜",
    "route": "/juejin/trending"
  },
  {
    "id": "category-frontend",
    "label": "前端",
    "route": "/juejin/category/:category",
    "params": { "category": "frontend" }
  },
  {
    "id": "category-backend",
    "label": "后端",
    "route": "/juejin/category/:category",
    "params": { "category": "backend" }
  },
  {
    "id": "category-ai",
    "label": "人工智能",
    "route": "/juejin/category/:category",
    "params": { "category": "ai" }
  },
  {
    "id": "category-freebie",
    "label": "开发工具",
    "route": "/juejin/category/:category",
    "params": { "category": "freebie" }
  },
  {
    "id": "category-general",
    "label": "综合",
    "route": "/juejin/category/:category",
    "params": { "category": "general" }
  }
]
```

## 数据返回格式

每条文章包含以下字段：

- **`title`** - 文章标题
- **`url`** - 文章链接
- **`summary`** - 文章摘要
- **`cover`** - 封面图片 URL
- **`content`** - 完整文章 HTML 内容
- **`author`** - 作者名称
- **`tags`** - 文章标签数组
- **`publishedAt`** - 发布时间（RFC3339 格式）

## 常见问题

### Q: 如何添加新的技术分类？
A: 掘金支持特定的分类 ID。如需添加新分类，请：
1. 确认掘金官网中该分类存在
2. 查找其 category_id 并在 manifest.json 中配置对应的 params
3. 同步更新 main.go 中的 categoryIDMap 映射表

### Q: 为什么有些文章内容为空？
A: 可能原因：1) 文章页面结构已变更；2) IP 被限流；3) 网络连接问题。请检查网络连接并稍后重试。

### Q: 如何修改刷新频率？
A: 编辑 `config.refreshInterval` 参数（单位：秒）。例如改为 900 表示每 15 分钟刷新一次。

### Q: 综合频道和热榜有什么区别？
A: 热榜展示全站最受欢迎的文章；综合频道基于用户偏好的个性化推荐。

## 技术细节

- 插件通过网页爬虫方式自动提取文章详情（非直接 API 调用）
- 自动提取文章封面、完整 HTML 内容和标签
- 支持 8 个并发的页面爬取任务以提高效率
- 每次请求最多返回 20 条文章
