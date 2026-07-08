# Liblib AI创意素材库

AI创意素材库的Orbit插件，支持摄影、插画、平面设计、动漫游戏等多个分类。

## 功能特性

- 🎨 多个创意分类频道（摄影写真、风格插画、平面设计、动漫游戏）
- 📖 分页支持
- ⚡ 快速加载
- 🔄 自动刷新

## 支持的频道

| 频道ID | 标签名称 | Tag ID |
|--------|--------|--------|
| inspiration | 摄影写真 | 550005 |
| illustration | 风格插画 | 560045 |
| design | 平面设计 | 560024 |
| game | 动漫游戏 | 560032 |

## API说明

插件使用 Liblib API (`https://api2.liblib.art/api/www/img/group/search`) 来获取创意作品。

- 默认分类：摄影写真 (550005)
- 分页方式：偏移分页 (page)
- 每页数量：30

## 开发

### 构建

```bash
make build PLUGIN=liblib
```

### 测试

```bash
make try PLUGIN=liblib
make try PLUGIN=liblib CHANNEL=illustration
```

### 打包

```bash
make package PLUGIN=liblib
make orbit PLUGIN=liblib
```

## 故障排查

- **无法获取数据**：检查网络和API是否可访问
- **分类无结果**：使用默认分类 550005（摄影写真）
