# 歌曲宝音频插件

基于 [歌曲宝](https://www.gequbao.com/) 的音频插件，实现频道列表、歌曲详情、歌词与 MP3 直链播放。

## 前置配置

歌曲宝受 **Cloudflare** 保护。抓取策略为 **Session-assisted API**（见 `schemas/browser-preview.md`）：

```json
"executionMode": "wasm",
"browser": {
  "purpose": "session",
  "fallbackOn": ["captcha", "http_403"],
  "persist": ["cookie", "userAgent"]
}
```

默认无需手动配置：

1. 插件先走 WASM HTTP 请求（`executionMode: "wasm"`）
2. 若被 CF 拦截，Orbit 按 `browser.purpose: "session"` 弹出浏览器完成验证
3. 验证通过后 Runtime 把 Cookie / UA 写入插件变量（`persist`），再重试接口请求

**可选：** 在插件设置手动填入 `cookie`（含 `cf_clearance`），可跳过验证弹窗。Cookie 会过期，失效后清空即可重新触发验证。

## 路由说明

- 频道列表：`/gequbao/channel`
  - 参数：`url`（频道页 URL）、`page`（当前页码）
- 歌曲详情：`/gequbao/detail`
  - 参数：`url`（详情页 URL）或 `id`（歌曲 ID）

## 数据抓取逻辑

1. 列表页从 HTML 解析歌曲行（标题、歌手、时长、详情链接）。
2. 详情页解析 `window.appData`，提取 `play_id`、封面、标题、歌手等信息。
3. 通过 `POST /member/common-play-url` + `id=play_id` 换取真实 MP3 链接。
4. 解析 `#content-lrc` 输出歌词文本并写入 `content`。

## 本地测试

```bash
cd plugins/audio/gequbao

# 测试频道（无 Cookie，本地 go run 不会弹浏览器）
make test-native

# 带 Cookie 测试
make test-native VARS='{"cookie":"你的Cookie"}'

# 测试详情
echo '{"action":"fetch","data":{"channelId":"detail","route":"/gequbao/detail","params":{"url":"https://www.gequbao.com/music/6421"}}}' | go run .
```

## 构建

```bash
cd plugins/audio/gequbao
make build
make package
```
