# 1x.com 摄影插件

1x.com 是全球最大的精选摄影库，汇集来自世界各地专业摄影师的 20 万+ 高质量照片作品。本插件提供实时摄影流订阅，支持按分类浏览获奖作品、最新发布和摄影杂志。

## 功能特性

- 实时获取 1x.com 精选摄影作品
- 支持 23 个摄影分类和杂志频道
- 完整的照片元数据（标题、摄影师、高清图链接）
- 高性能 Go 原生实现
- 支持分页浏览历史图片（`features.pagination`）

## 频道配置

所有频道通过 `features.pagination` 声明翻页能力，默认定时订阅并落库：

```json
"features": {
  "pagination": {
    "style": "offset",
    "param": "page",
    "default": "1",
    "sizeParam": "size",
    "defaultSize": 20
  }
}
```

### 摄影分类频道（获奖作品）

| 频道 ID | 频道名称 | 路由参数 |
|--------|--------|--------|
| `latest-awarded` | 最新获奖 | `latest/awarded` |
| `popular-awarded` | 热门获奖 | `popular/awarded` |
| `abstract-awarded` | 抽象摄影获奖 | `abstract/awarded` |
| `action-awarded` | 动作摄影获奖 | `action/awarded` |
| `animals-awarded` | 动物摄影获奖 | `animals/awarded` |
| `architecture-awarded` | 建筑摄影获奖 | `architecture/awarded` |
| `conceptual-awarded` | 概念摄影获奖 | `conceptual/awarded` |
| `creative-edit-awarded` | 创意编辑获奖 | `creative-edit/awarded` |
| `documentary-awarded` | 纪录摄影获奖 | `documentary/awarded` |
| `everyday-awarded` | 日常摄影获奖 | `everyday/awarded` |
| `fine-art-nude-awarded` | 艺术人体获奖 | `fine-art-nude/awarded` |
| `humour-awarded` | 幽默摄影获奖 | `humour/awarded` |
| `landscape-awarded` | 风景摄影获奖 | `landscape/awarded` |
| `macro-awarded` | 微距摄影获奖 | `macro/awarded` |
| `mood-awarded` | 情绪摄影获奖 | `mood/awarded` |
| `night-awarded` | 夜景摄影获奖 | `night/awarded` |
| `performance-awarded` | 表演摄影获奖 | `performance/awarded` |
| `portrait-awarded` | 人像摄影获奖 | `portrait/awarded` |
| `still-life-awarded` | 静物摄影获奖 | `still-life/awarded` |
| `street-awarded` | 街拍摄影获奖 | `street/awarded` |
| `underwater-awarded` | 水下摄影获奖 | `underwater/awarded` |
| `wildlife-awarded` | 野生动物获奖 | `wildlife/awarded` |

### 杂志频道

| 频道 ID | 频道名称 | 路由参数 |
|--------|--------|--------|
| `magazine-latest` | 杂志最新 | 路由：`/1x/magazine/:category`，参数：`latest` |

## 自定义频道

支持通过参数 `category` 自定义频道，格式为 `category/type`：

**Category（分类）**：上述任意摄影分类或其他 1x.com 支持的分类

**Type（类型）**：
- `awarded` - 获奖作品
- `published` - 已发布作品

### 频道配置示例

**基础配置**：
```json
{
  "id": "custom-landscape",
  "label": "风景摄影（已发布）",
  "route": "/1x/:category",
  "params": {
    "category": "landscape/published",
    "page": "1",
    "size": "20"
  },
  "status": "enabled",
  "features": {
    "pagination": {
      "style": "offset",
      "param": "page",
      "default": "1",
      "sizeParam": "size",
      "defaultSize": 20
    }
  }
}
```

**参数说明**：
- `features.pagination` - 支持翻页；定时刷新拉第一页并落库，加载更多追加历史
- `page` - 初始页码（从 1 开始）
- `size` - 每页数量（默认 20，最大 20）

## 使用说明

### 基本使用流程

1. 在 Orbit 应用中添加此插件
2. 选择预置频道或自定义分类参数
3. 实时接收摄影作品更新
4. 点击订阅项目直接跳转到 1x.com 原页面

### 动态查询示例

**示例 1：浏览风景摄影的第二页**

```json
{
  "category": "landscape/awarded",
  "page": "2",
  "size": "20"
}
```

**示例 2：查看动物摄影已发布作品**

```json
{
  "category": "animals/published",
  "page": "1",
  "size": "20"
}
```

**示例 3：一次性查看更多图片（注意：最多仍为 20 条）**

```json
{
  "category": "latest/awarded",
  "page": "1",
  "size": "40"  // 会被自动限制为 20
}
```

### 工作原理

1. 用户选择频道或修改查询条件
2. 应用发送请求给插件，包含 `page`、`size` 和 `category` 参数
3. 插件计算 API 的 `from` 偏移量：`from = (page - 1) * size`
4. 向 1x.com API 发送请求：`/backend/lm2.php?style=normal&mode=<mode>&from=<from>`
5. 解析返回的图片数据并渲染到应用中

## 查询条件与分页

### 查询参数

所有频道都支持以下查询参数进行动态刷新：

| 参数 | 类型 | 默认值 | 说明 |
|-----|------|--------|------|
| `category` | string | 取决于频道 | 摄影分类，格式为 `category/type` |
| `page` | string | `1` | 页码，从 1 开始 |
| `size` | string | `20` | 每页数量，最大 20（API 限制） |

### 分页查询

插件支持标准的分页参数进行动态查询：

- **page** - 页码（从 1 开始，默认值：`1`）
- **size** - 每页数量（默认值：`20`）

**分页示例：**

```json
{
  "category": "latest/awarded",
  "page": "2",
  "size": "20"
}
```

**API 限制说明：**

1x.com 的 API 有硬限制，每次请求最多返回 **20 条记录**。即使指定 `size` 参数为更大的值（如 40、50），插件也会自动限制为最多 20 条，以确保与 API 的兼容性。

### 分页计算逻辑

插件内部将分页参数转换为 API 的 `from` 偏移量：

```
from = (page - 1) * size
```

例如：
- `page=1, size=20` → `from=0`（获取第 1-20 条）
- `page=2, size=20` → `from=20`（获取第 21-40 条）
- `page=3, size=20` → `from=40`（获取第 41-60 条）

### 翻页浏览

当频道配置了 `features.pagination` 时，用户可以：

1. **修改分页** - 通过改变 `page` 参数翻页查看历史图片
2. **自定义页面大小** - 通过 `size` 参数调整每页显示数量（最大 20）
3. **切换分类** - 通过 `category` 参数切换不同的摄影分类

定时刷新拉取第一页并落库；加载更多时追加更旧记录。

**使用场景**：

- 查看最新获奖作品的第 2 页
- 浏览动物摄影分类的历史作品
- 一次性加载更多图片（调整 size 参数）

## 数据字段

每条摄影作品包含以下信息：

- **标题** - 作品名称
- **摄影师** - 创作者名字
- **图片** - 高清作品图链接
- **链接** - 作品详情页 URL
- **发布时间** - 数据获取时间
