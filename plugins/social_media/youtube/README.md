# YouTube Plugin

YouTube视频订阅插件，使用YouTube Data API v3获取频道和播放列表视频。

## 配置要求

### 获取YouTube API密钥

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建新项目或选择现有项目
3. 启用 YouTube Data API v3：
   - 导航到 "APIs & Services" > "Library"
   - 搜索 "YouTube Data API v3"
   - 点击 "Enable"
4. 创建API密钥：
   - 导航到 "APIs & Services" > "Credentials"
   - 点击 "Create Credentials" > "API Key"
   - 复制生成的API密钥
5. （可选）限制API密钥：
   - 点击密钥名称编辑
   - "API restrictions" > 选择 "YouTube Data API v3"
   - 保存

### 配置 API Key

manifest 中**只声明**用户需填写的字段，**不要**把真实密钥提交到仓库：

```json
{
  "config": {
    "variables": {
      "apiKey": {
        "label": "YouTube API Key",
        "description": "Google Cloud API Key，需启用 YouTube Data API v3",
        "required": true,
        "secret": true
      }
    }
  }
}
```

在 Orbit 客户端的「插件设置」中填写 API Key；本地测试可通过 `vars` 传入：

```bash
echo '{"action":"fetch","data":{...,"vars":{"apiKey":"YOUR_YOUTUBE_API_KEY"}}}' | go run .
```

## API配额说明

- **免费配额**：每天10,000 units
- **成本**：
  - `playlistItems.list`: 1 unit（获取播放列表视频）
  - `videos.list`: 1 unit（获取视频统计）
  - `channels.list`: 1 unit（获取频道信息）
  - 每次刷新约消耗 **2-3 units**

- **估算**：10,000 units可支持每天约 **3,000-5,000次** 刷新

## 测试

### Native测试（无需WASM）

```bash
# 测试频道
echo '{"action":"fetch","data":{"channelId":"test","route":"/youtube/channel/:channelId","params":{"channelId":"UCDwDMPOZfxVV0x_dz0eQ8KQ"},"vars":{"apiKey":"YOUR_API_KEY"}}}' | go run .

# 测试用户名
echo '{"action":"fetch","data":{"channelId":"test","route":"/youtube/user/:username","params":{"username":"laogao"},"vars":{"apiKey":"YOUR_API_KEY"}}}' | go run .

# 测试播放列表
echo '{"action":"fetch","data":{"channelId":"test","route":"/youtube/playlist/:playlistId","params":{"playlistId":"PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z"},"vars":{"apiKey":"YOUR_API_KEY"}}}' | go run .
```

### 构建WASM

```bash
make build    # 编译到 dist/youtube/plugin.wasm
make package  # 打包manifest和assets
```

## 支持的路由

1. **频道订阅** (`/youtube/channel/:channelId`)
   - 通过频道ID获取最新上传视频
   - 示例：`UCDwDMPOZfxVV0x_dz0eQ8KQ`

2. **用户订阅** (`/youtube/user/:username`)
   - 通过用户名（@handle）获取视频
   - 示例：`laogao` 或 `@laogao`

3. **播放列表订阅** (`/youtube/playlist/:playlistId`)
   - 通过播放列表ID获取视频
   - 示例：`PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z`

## 技术说明

- 使用 `playlistItems.list` 替代 `search.list`，成本降低100倍（1 unit vs 100 units）
- 批量获取视频统计信息（views等），每50个视频仅消耗1 unit
- 从HTML页面提取用户名对应的频道ID（无API成本）
- 每次返回最多 50 个视频
- 分页使用 `lastId` 游标（上一页最后一条视频的 ID），无状态，适合 WASM 运行

## 分页说明

YouTube Data API 使用 `pageToken` 翻页，但插件对外暴露 **基于视频 ID 的 `lastId` 游标**：

- 首次加载：`lastId` 为空，返回第一页
- 加载更多：客户端传入上一页最后一条视频的 `item.id`
- 插件从列表头遍历定位该视频，返回其后的下一批结果
- 响应中 `hasMore: true` 时，`next.lastId` 为当前页最后一条视频 ID

## 故障排查

### "YouTube API key required"
- 确认已在客户端插件设置或测试命令的 `vars.apiKey` 中配置 API Key

### "http status 403"
- API密钥无效或未启用YouTube Data API v3
- 检查Google Cloud Console中API是否已启用

### "http status 429"
- 超出每日配额限制（10,000 units）
- 等待24小时后重置，或申请配额扩展

### "quota exceeded"
- 申请更高配额：Google Cloud Console > "APIs & Services" > "Quotas"
- 填写配额申请表，通常可扩展到100,000+ units/天
