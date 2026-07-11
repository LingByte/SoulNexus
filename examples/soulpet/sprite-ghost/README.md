# Soul Pet 示例 — 帧动画 Ghost

本地开发：

```bash
# 将本目录作为 .soulpet 包根目录
# 在 SoulNexus Web → 我的桌宠 → 导入 zip
# 或 CLI: soul-pet push（规划中）
```

必需文件：`soulpet.yaml`、`manifest.json`、`pet.js`、`assets/sprites/*`

运行时 API：加载后使用 `window.__SOUL_PET__.chat('你好')` 或 `__SOUL_PET__.playAnimation('tap')`
