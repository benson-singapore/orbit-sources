# 联合早报插件使用指南

## 概述
本插件从新加坡联合早报网站抓取实时热门新闻，支持多个地区和分类的新闻源。

## 内置频道

### 支持的分类
插件目前支持以下 4 种新闻分类：

| section值 | 分类名称 | 说明 |
|-----------|---------|------|
| `china` | 中国 | 中港台热门新闻（默认） |
| `singapore` | 新加坡 | 新加坡本地新闻 |
| `world` | 国际 | 国际热门新闻 |
| `zfinance` | 财经 | 财经金融资讯 |

## 自定义频道

如需添加其他分类或自定义频道，请编辑 manifest.json 配置中的 `channels` 数组。

### 示例：添加财经频道

```json
{
  "id": "finance",
  "label": "财经",
  "route": "/zaobao/realtime/:section",
  "params": { "section": "zfinance" }
}
```

### 添加步骤

1. 编辑 manifest.json 文件
2. 在 `config.channels` 数组中添加新对象
3. 设置唯一的 `id`（用于内部标识）
4. 设置用户可见的 `label`（频道显示名称）
5. 保持 `route` 为 `/zaobao/realtime/:section`
6. 在 `params` 中设置 `section` 值（见上表）
7. 保存并重新加载插件

## 参数说明

### route 参数

- **`:section`** - 必需，指定新闻分类。取值范围：`china` | `singapore` | `world` | `zfinance`

### 其他配置

- **`refreshInterval`** - 刷新间隔（秒），默认 1800 秒（30分钟）
- **`defaultChannel`** - 默认频道 ID，应该是 channels 数组中存在的 id
- **`timeoutMs`** - 超时时间（毫秒），默认 120000ms（2分钟）

## 配置示例

### 完整配置（含财经频道）

```json
"channels": [
  {
    "id": "china",
    "label": "中国",
    "route": "/zaobao/realtime/:section",
    "params": { "section": "china" }
  },
  {
    "id": "singapore",
    "label": "新加坡",
    "route": "/zaobao/realtime/:section",
    "params": { "section": "singapore" }
  },
  {
    "id": "world",
    "label": "国际",
    "route": "/zaobao/realtime/:section",
    "params": { "section": "world" }
  },
  {
    "id": "finance",
    "label": "财经",
    "route": "/zaobao/realtime/:section",
    "params": { "section": "zfinance" }
  }
]
```

## 数据返回格式

每条新闻包含以下字段：

- **`title`** - 新闻标题
- **`url`** - 新闻链接
- **`summary`** - 新闻摘要
- **`cover`** - 封面图片 URL
- **`content`** - 完整文章 HTML 内容
- **`published_at`** - 发布时间（RFC3339 格式）

## 常见问题

### Q: 如何添加新的新闻分类？
A: 早报网站目前只支持上表中的 4 种分类。如需其他分类，请检查早报网站是否提供相应的 URL 路由。

### Q: 为什么有些频道返回空结果？
A: 可能原因：1) 网站结构变更；2) IP 被限流；3) 网络连接问题。请检查网络连接并稍后重试。

### Q: 如何修改刷新频率？
A: 编辑 `config.refreshInterval` 参数（单位：秒）。例如改为 900 表示每 15 分钟刷新一次。

## 技术细节

- 插件通过网页爬虫方式抓取数据（非 API）
- 自动提取文章封面、摘要和完整 HTML 内容
- 支持新加坡版（.com.sg）和香港版（.com）网站
- 每次请求最多返回 10 条新闻
