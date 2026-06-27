# 构建与打包

## 1. 构建 WASM

```bash
make build PLUGIN=juejin
```

产物：

```text
dist/juejin/plugin.wasm
```

## 2. 打包运行目录

```bash
make package PLUGIN=juejin
```

产物：

```text
dist/juejin/
  plugin.wasm
  manifest.json
  README.md        # 如果插件提供
  assets/          # 如果插件提供
```

## 3. 生成 extension.orbit

```bash
make orbit PLUGIN=juejin
```

产物：

```text
dist/juejin/extension.orbit
```

`extension.orbit` 是 ZIP 包，包含：

- `manifest.json`
- `main.wasm.br`
- `README.md`，可选

## 4. 批量构建

```bash
make build-all
make package-all
make orbit-all
```

也可以指定多个插件：

```bash
make build PLUGIN=juejin,youtube
make orbit PLUGIN=juejin,youtube
```

## 5. 同步到应用仓库

仅在需要验证应用内置插件或打包桌面应用时使用：

```bash
make sync PLUGIN=juejin INSTALL_PLUGINS_DIR=../plugins
```

日常开发不需要 sync，推荐使用 `dist/` 加 Runtime dev 模式。

## 6. 清理产物

```bash
make clean PLUGIN=juejin
make clean-all
```
