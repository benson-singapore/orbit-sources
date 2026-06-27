# 测试指南

Orbit 插件推荐三层验证：原生测试、WASM 测试、Runtime 联调。

## 1. 原生测试

最快，适合日常开发抓取和解析逻辑：

```bash
make try PLUGIN=juejin ROUTE=/juejin/trending PARAMS='{}'
```

也可以进入插件目录手动构造请求：

```bash
cd plugins/programming/juejin
echo '{"action":"fetch","data":{"channelId":"trending","route":"/juejin/trending","params":{}}}' | go run .
```

期望输出：

```json
{"ok":true,"data":{"title":"...","items":[...]}}
```

## 2. WASM 测试

用于确认 `wasip1/wasm` 产物和 host 函数行为：

```bash
brew install wasmtime
make try-wasm PLUGIN=juejin ROUTE=/juejin/trending PARAMS='{}'
```

如果 `dist/<id>/plugin.wasm` 不存在，脚本会先执行打包。

## 3. Runtime 联调

先启动 Orbit Runtime：

```bash
cd ..
make dev-go
```

然后在 `orbit-sources` 中执行：

```bash
make dev PLUGIN=juejin CHANNEL=trending ROUTE=/juejin/trending PARAMS='{}'
```

Runtime 联调会执行：

- 构建并 package 插件
- 调用 Runtime 插件安装或重新扫描接口
- 刷新 feed
- 拉取前 3 条结果用于检查

## 4. 常见场景

测试带分类参数的频道：

```bash
make try PLUGIN=juejin CHANNEL=category-frontend ROUTE=/juejin/category/:category PARAMS='{"category":"frontend"}'
```

测试需要用户变量的插件：

```bash
cd plugins/social_media/youtube
echo '{"action":"fetch","data":{"channelId":"search","route":"/youtube/search","params":{"query":"orbit"},"vars":{"apiKey":"YOUR_KEY"}}}' | go run .
```

## 5. 发布前验证

- `make list` 能发现插件
- `make try PLUGIN=<id> ROUTE=<route> PARAMS='{}'` 返回 `ok: true`
- `make build PLUGIN=<id>` 能生成 `dist/<id>/plugin.wasm`
- `make orbit PLUGIN=<id>` 能生成 `dist/<id>/extension.orbit`
- manifest 不包含密钥、账号、Cookie 或成人内容
