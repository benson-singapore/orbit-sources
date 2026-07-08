# Orbit Sources — 项目级 Codex Skills

本项目包含针对 Orbit 插件开发的本地 skill。

## 📂 结构

```
.codex/
├── config.json                  # 项目配置
├── README.md                    # 本文件
└── skills/
    └── orbit-plugin-builder/    # Orbit 插件构建 skill
        ├── SKILL.md             # 主文档
        ├── references/          # 快速参考
        ├── evals/               # 测试场景
        └── scripts/             # 预留
```

## 🎯 本地 Skills

### orbit-plugin-builder

完整的 Orbit 插件构建工作流。

**触发条件：** 在 Codex 中提及 "Orbit 插件"、"创建"、"分页" 等关键词

**功能：**
- 从模板创建新插件
- 实现分页功能（3 种风格）
- 配置 manifest.json
- 编译、打包、测试
- 故障排查和调试

**位置：** `.codex/skills/orbit-plugin-builder/`

## 📖 使用方式

在 Codex 中打开本项目时，skill 会自动可用：

```
用户：怎样创建新的 Orbit 插件？
Codex（使用 orbit-plugin-builder skill）：
  1. 复制模板...
  2. 编辑文件...
  3. 构建命令...
```

## 🔧 配置

项目配置文件：`.codex/config.json`

```json
{
  "skillsPath": ".codex/skills",
  "version": "1.0.0",
  "project": "orbit-sources",
  "description": "Orbit news plugin sources with local skill support"
}
```

## 📝 维护

### 更新 Skill

当 schemas 或实现变更时：

```bash
# 重新生成 skill（如果有新的规则分析）
cp -r /path/to/updated/orbit-plugin-builder .codex/skills/

# 验证
head -5 .codex/skills/orbit-plugin-builder/SKILL.md
```

### 添加新 Skills

1. 创建新目录：`.codex/skills/新-skill-名/`
2. 遵循结构：SKILL.md + references/ + evals/
3. 提交到项目

## 🚀 快速开始

在项目根目录打开 Codex，问关于 Orbit 插件的任何问题：

```bash
cd /Users/benson/Documents/project-rust/orbit-sources
# 在 Codex 中打开此项目
# Skill 会自动可用
```

---

**本地 Skills 已准备好！**

