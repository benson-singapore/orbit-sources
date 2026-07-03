# 分页参数 `seenIds` 规范

本文说明 Orbit 插件分页中的 `seenIds` 是什么、何时需要、如何在 manifest / Runtime / 插件代码中正确传参。

适用场景：**推荐流、发现页、热门榜**等「内容池会重排、仅靠 `page` 或 `offset` 会大量重复」的数据源。参考实现：`plugins/picture/jimeng`、`plugins/manga/baozi`。

---

## 1. `seenIds` 是什么

`seenIds` 是一个 **请求参数字符串**，值为**逗号分隔、已展示给用户的条目 ID 列表**（累计、去重、保序）。

| 属性 | 说明 |
|------|------|
| 参数名 | `seenIds` |
| 类型 | `string` |
| 格式 | `id1,id2,id3`（无空格；空字符串表示「尚未展示任何条目」） |
| ID 来源 | 插件返回的 `FeedItem.id`（源站作品/帖子/漫画 id，非 Orbit 加前缀后的 fullId） |
| 作用 | 告诉插件与上游 API：「这些内容用户已经看过了，请不要再次返回」 |

它不是数据库字段，也**不应**在 `persist: true` 时依赖本地库去重代替——`seenIds` 解决的是**单次浏览会话内、跨分页请求**的去重。

---

## 2. 为什么需要 `seenIds`

部分站点（如即梦发现页）的后端逻辑是：

- 每页固定 `count=40`，`offset = (page - 1) × 40`
- 内容是**推荐池**，会随时间重排，**不是**稳定时间线
- 仅靠 `page` 翻页时，相邻页重叠率可达 50% 以上

上游通常提供「已曝光 ID 列表」字段（如即梦的 `expose_item_id_list`）。插件把 `seenIds` 映射过去，才能拿到真正的新内容。

---

## 3. manifest 如何声明

在频道 `params` 中声明初始值，在 `pagination.carryParams` 中声明「加载更多时要从 `next` 带回的参数」。

```json
{
  "id": "trending",
  "route": "/jimeng/explore",
  "params": {
    "page": "1",
    "seenIds": ""
  },
  "features": {
    "pagination": {
      "style": "offset",
      "param": "page",
      "default": "1",
      "defaultSize": 40,
      "carryParams": ["seenIds"]
    },
    "feed": {
      "persist": false,
      "refresh": true,
      "limit": 40
    }
  }
}
```

| 字段 | 含义 |
|------|------|
| `params.seenIds` | 首屏默认值，必须为 `""` |
| `pagination.param` | 主分页参数（如 `page`） |
| `pagination.carryParams` | 除 `param` 外、需随 `next` 一并合并的附加参数名列表 |
| `feed.persist` | 推荐流通常设为 `false`（实时展示、不落库） |

Schema 定义见 `schemas/features.schema.json` 中 `pagination.carryParams`。

---

## 4. 传参规则（Runtime / 客户端）

### 4.1 首屏与下拉刷新

使用 manifest 中的 **默认分页值**，`seenIds` 置空：

```json
{
  "channelId": "trending",
  "route": "/jimeng/explore",
  "params": {
    "page": "1",
    "seenIds": ""
  }
}
```

**不要**在 `page=1` 时携带上一次浏览遗留的 `seenIds`（插件侧也会对 `page=1` 忽略旧 `seenIds`，但客户端仍应主动清空）。

### 4.2 加载更多

上一次 `fetch` 若返回 `hasMore: true`，响应中会带有 `next`：

```json
{
  "ok": true,
  "data": {
    "items": [ "..." ],
    "hasMore": true,
    "next": {
      "page": "2",
      "seenIds": "7627090507965009202,7651742291727600906,..."
    }
  }
}
```

加载更多时，将 **`next` 中与当前频道相关的字段合并进 `params`**：

1. 始终更新 `pagination.param`（上例为 `page` → `"2"`）
2. 对 `pagination.carryParams` 中的每个 key（上例为 `seenIds`），使用 `next` 中的值
3. 保留频道固定参数（如 `categoryId`、`workType`）；若 `next` 已包含则以其为准

合并后的请求示例：

```json
{
  "channelId": "trending",
  "route": "/jimeng/explore",
  "params": {
    "categoryId": "11222",
    "workType": "video,image,canvas",
    "page": "2",
    "seenIds": "7627090507965009202,7651742291727600906,..."
  }
}
```

### 4.3 伪代码（客户端）

```text
params = channel.params  // 来自 manifest 默认值

function onLoadMore(next):
  params[pagination.param] = next[pagination.param]
  for key in pagination.carryParams:
    params[key] = next[key]

function onRefresh():
  params[pagination.param] = pagination.default   // 通常 "1"
  for key in pagination.carryParams:
    params[key] = ""                              // seenIds 清空
```

### 4.4 常见错误

| 错误做法 | 后果 |
|----------|------|
| 加载更多只传 `page`，不传 `seenIds` | 大量重复条目 |
| 刷新时仍带旧 `seenIds` | 首屏内容异常偏少或为空 |
| 自行拼接 `seenIds` 但未包含当前页全部 id | 与上游曝光列表不一致，仍会重复 |
| 把 Orbit 前缀 fullId 写入 `seenIds` | 上游无法识别，去重失效 |

**正确做法**：只使用插件 `next.seenIds` 回传值，不要客户端自行维护半套逻辑。

---

## 5. 插件实现约定

### 5.1 读取与重置

```go
page := parseInt(req.Params["page"], 1)
seenIDs := parseIDList(req.Params["seenIds"])
if page == 1 {
    seenIDs = nil // 首屏忽略陈旧 seenIds
}
```

### 5.2 请求上游

- 第 1 页：按源站要求使用「首次加载」标记（即梦为 `feed_refer: "feed_refresh"`）
- 第 2 页起：使用「加载更多」标记，并传入曝光列表（即梦为 `expose_item_id_list: seenIDs`）

### 5.3 过滤与构造 `next`

```go
seenSet := set(seenIDs)
allSeen := append([]string{}, seenIDs...)

for _, raw := range upstreamItems {
    item := toFeedItem(raw)
    if seenSet[item.ID] { continue }
    seenSet[item.ID] = true
    allSeen = append(allSeen, item.ID)
    items = append(items, item)
}

if hasMore {
    next := copyParams(req.Params)
    next["page"] = strconv.Itoa(page + 1)
    next["seenIds"] = strings.Join(allSeen, ",")
}
```

要点：

- `seenIds` **累计**本频道会话内所有已展示 id，不是仅上一页
- `next` 建议 `copyParams` 保留 `categoryId` 等固定字段
- `hasMore` 可在上游为 true 但本页过滤后无新条目时置为 false

### 5.4 ID 格式

`seenIds` 中的每个 id 必须与 `FeedItem.id` 一致（源站原始 id）。Runtime 存储用的 `{pluginId}:{channelId}:{id}` 前缀**不要**写入 `seenIds`。

---

## 6. 测试

### 6.1 首屏

```bash
cd plugins/picture/jimeng
echo '{"action":"fetch","data":{"channelId":"trending","route":"/jimeng/explore","params":{"page":"1","seenIds":""}}}' | go run .
```

### 6.2 模拟加载更多

将上一次响应里的 `next` 原样并入 `params` 再请求：

```bash
echo '{"action":"fetch","data":{"channelId":"trending","route":"/jimeng/explore","params":{"page":"2","seenIds":"7627090507965009202,7651742291727600906"}}}' | go run .
```

### 6.3 验证去重

连续请求 page=1、2、3，检查：

- 各页 `items[].id` 交集应明显小于仅用 `page` 时不传 `seenIds` 的情况
- `next.seenIds` 长度随页数单调增加（累计 id 更多）

Makefile 快捷命令：

```bash
make try PLUGIN=jimeng ROUTE=/jimeng/explore PARAMS='{"page":"1","seenIds":""}'
```

---

## 7. 参考插件

| 插件 | 路径 | 说明 |
|------|------|------|
| 即梦 AI | `plugins/picture/jimeng` | `page` + `seenIds` + `carryParams`；映射 `expose_item_id_list` |
| 包子漫画 | `plugins/manga/baozi` | 列表页 HTML 去重，同样通过 `next.seenIds` 回传 |

协议字段说明亦见 `schemas/abi-v1.md` 中 **Feed result fields** 与 **pagination** 章节。
