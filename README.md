# Orbit Sources

Orbit Sources 是 Orbit 的官方 WASM 插件源码仓库。插件使用 Go 编写，编译为 `wasip1/wasm`，由 Orbit Runtime 加载执行。

## 快速开始

```bash
make list
make build PLUGIN=juejin
make package PLUGIN=juejin
```

常用命令：

| 命令 | 说明 |
|------|------|
| `make list` | 列出已发现插件 |
| `make build PLUGIN=<id>` | 编译 `dist/<id>/plugin.wasm` |
| `make package PLUGIN=<id>` | 编译并复制 `manifest.json`、`README.md`、`assets/` |
| `make orbit PLUGIN=<id>` | 生成 `dist/<id>/extension.orbit` |
| `make try PLUGIN=<id>` | 原生 `go run` 快速测试 |
| `make try-wasm PLUGIN=<id>` | 使用 `wasmtime` 测试 WASM 产物 |
| `make dev PLUGIN=<id>` | 对接本地 Orbit Runtime 联调 |

## 目录结构

```text
orbit-sources/
  sdk/                 # 插件 SDK
  plugins/             # 插件源码，按分类组织
    programming/juejin/
      main.go
      manifest.json
      Makefile
  schemas/             # manifest、features、playback、ABI 协议
  scripts/             # 开发测试脚本
  cmd/orbit-pack/      # extension.orbit 打包工具
  dist/                # 构建产物，不提交 git
```

## 当前插件

| 分类 | 插件 |
|------|------|
| `programming` | `hellogithub`, `juejin` |
| `social_media` | `douban`, `huxiu`, `sspai`, `substack`, `tmdb`, `voronoi`, `yansg`, `youtube`, `zaobao` |
| `reading` | `yilin` |
| `picture` | `1x`, `pixabay`, `unsplash` |
| `manga` | `baozi`, `gman` |
| `video` | `rycjapi` |
| `audio` | `lrts` |

## 文档

- `docs/development.md`：从零开发插件
- `docs/testing.md`：原生、WASM、Runtime 三层测试
- `docs/packaging.md`：构建、打包和发布产物
- `schemas/abi-v1.md`：插件与 Runtime 的 JSON 协议
- `schemas/manifest.wasm.schema.json`：manifest JSON Schema
