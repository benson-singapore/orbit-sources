# Orbit Plugin Builder Skill — 文件导航

## 快速开始

1. **新建插件？** → 读 SKILL.md 的 "Build Workflow" 部分 + "File Templates"
2. **加分页？** → 读 SKILL.md 的 "Pagination" 部分 + references/abi-quick-ref.md
3. **调试选择器？** → 读 SKILL.md 的 "Troubleshooting" 部分
4. **理解 manifest？** → 读 references/manifest-quick-ref.md
5. **理解 ABI？** → 读 references/abi-quick-ref.md

## 文件结构

```
orbit-plugin-builder/
│
├── SKILL.md (606 行)
│   ├── 核心概念 (Plugin Anatomy, SDK Interface, Manifest)
│   ├── 构建流程 (Scaffold → Build → Package → Orbit)
│   ├── 通用模式 (Parse HTML, HTTP, Pagination, Normalize Items)
│   ├── 验证清单 (Verification Checklist)
│   ├── 故障排查 (Troubleshooting)
│   └── 文件模板 (main.go, Makefile, go.mod, manifest.json)
│
├── references/
│   ├── manifest-quick-ref.md
│   │   ├── 最小有效 manifest
│   │   ├── Config 字段表
│   │   └── 常见模式 (news, social, search, manga)
│   │
│   └── abi-quick-ref.md
│       ├── 请求/响应格式
│       ├── FeedItem 字段表
│       ├── 社交字段 (stats, media, quote)
│       ├── 分页示例 (offset/cursor/lastId)
│       └── 代码示例 (Fetch 实现)
│
├── evals/
│   └── evals.json (5 个测试场景)
│       1. 新建插件
│       2. 加分页 (3 种风格)
│       3. 调试选择器
│       4. 打包发布
│       5. 用户变量
│
├── scripts/ (预留)
│
├── README.md
│   ├── Skill 简介
│   ├── 何时使用
│   ├── 规则汇总
│   ├── 结构说明
│   └── 规则来源
│
└── INDEX.md (本文件)
    └── 快速导航
```

## 按场景查找

### 场景 1：从零创建新插件

**文件：** SKILL.md + references/manifest-quick-ref.md

**步骤：**
1. SKILL.md § "Scaffold New Plugin"
2. 使用 SKILL.md § "File Templates" 中的模板
3. 参考 references/manifest-quick-ref.md § "Minimal Valid Manifest"
4. 按 SKILL.md § "Build & Package" 执行

---

### 场景 2：添加分页功能

**文件：** SKILL.md + references/abi-quick-ref.md

**步骤：**
1. SKILL.md § "pagination — load more"
2. 对比 references/abi-quick-ref.md § "Pagination" 中的三种风格
3. 按 SKILL.md § "Common Patterns" 中的代码模式实现
4. 使用 `make try PARAMS='{"page":"2"}'` 测试

---

### 场景 3：调试空列表

**文件：** SKILL.md

**步骤：**
1. SKILL.md § "Troubleshooting" → "Feed items missing or limited"
2. SKILL.md § "Debug Selectors"
3. 遵循 SKILL.md § "Testing & Debugging" 中的调试模式

---

### 场景 4：理解 manifest 结构

**文件：** references/manifest-quick-ref.md

**关键部分：**
- "Minimal Valid Manifest" → 最小示例
- "Config Fields" → 字段参考
- "Channel Features Matrix" → 功能矩阵
- "Common Patterns" → 新闻/社交/搜索/漫画的现成配置

---

### 场景 5：理解 ABI v1 契约

**文件：** references/abi-quick-ref.md

**关键部分：**
- "Request-Response Flow" → 基本 JSON 结构
- "FeedItem Fields" → 所有字段及规范化
- "Social Fields" → 社交特定字段
- "Code Example" → Fetch 实现参考

---

### 场景 6：社交插件开发

**文件：** SKILL.md + references/abi-quick-ref.md

**步骤：**
1. SKILL.md § "mediaType Options" → 选择 `social`
2. references/abi-quick-ref.md § "Social Fields" → 了解 stats/media/quote
3. references/manifest-quick-ref.md § "Common Patterns" → 参考社交模式
4. SKILL.md § "File Templates" → 使用新闻模板作基础，修改为社交字段

---

### 场景 7：漫画/剧集插件（三级导航）

**文件：** SKILL.md + references/manifest-quick-ref.md

**关键：**
- SKILL.md § "chapters — sub-list (three-level nav)"
- references/manifest-quick-ref.md § "Common Patterns" → manga 模式
- 特别注意 chapters.detail.parentParam 的角色

---

### 场景 8：搜索功能

**文件：** SKILL.md + references/manifest-quick-ref.md

**关键：**
- SKILL.md § "search — user query"
- manifest 中: `"feed": {"persist": false, "refresh": false}`
- `"search": {"param": "query", "required": true}`

---

## 规则优先级

当遇到不明确的情况时，按以下优先级查阅：

1. **Official Schemas** (最权威)
   - `schemas/manifest.wasm.schema.json`
   - `schemas/features.schema.json`
   - `schemas/abi-v1.md`

2. **This Skill** (已对齐 schemas)
   - SKILL.md 主体
   - references/ 快速参考

3. **Reference Implementations** (实例)
   - `plugins/news/zaobao/`
   - `plugins/news/huanqiu/`

4. **SDK Source** (实现细节)
   - `sdk/types.go`
   - `cmd/orbit-pack/main.go`

---

## 关键术语

| 术语 | 定义 |
|------|------|
| **Plugin** | Go WASM 程序，实现 `Plugin.Fetch(FetchRequest) FeedResult` |
| **Channel** | Plugin 内的一个列表（如新闻的"首页"、"科技"） |
| **Route** | 调用 Fetch 的路径模板（如 `/plugin/list/:section`） |
| **manifest.json** | 插件元数据、频道、路由、feature 声明 |
| **FetchRequest** | 运行时发送给 Plugin 的请求（channelId, route, params, vars） |
| **FeedResult** | Plugin 返回的结果（title, items[], hasMore, next） |
| **FeedItem** | 列表中的一条项目（id, title, url, cover, publishedAt, ...） |
| **hasMore** | 是否还有后续页面 |
| **next** | 下一页的参数 map（对应 pagination style） |
| **ABI v1** | 插件与运行时的通信约定（JSON stdin/stdout） |

---

## 常见命令速查

```bash
# 列出所有插件
make list

# 构建 + 打包 + 生成 extension.orbit
make build PLUGIN=myplugin
make package PLUGIN=myplugin
make orbit PLUGIN=myplugin

# 测试
make try PLUGIN=myplugin                    # 原生 Go
make try-wasm PLUGIN=myplugin               # WASM
make dev PLUGIN=myplugin                    # Runtime 集成

# 带参数的测试
make try PLUGIN=myplugin \
  CHANNEL=home \
  ROUTE=/myplugin/list/:section \
  PARAMS='{"section":"home","page":"1"}'

# 清理
make clean PLUGIN=myplugin
```

---

## 反馈与更新

Skill 基于 Orbit 仓库的当前状态 (2025-01-08)。

当以下内容变更时，请更新 Skill：
- `schemas/*.schema.json` 或 `schemas/abi-v1.md`
- 参考实现的模式变更
- 新增 mediaType 或 features
- 构建管道变更

