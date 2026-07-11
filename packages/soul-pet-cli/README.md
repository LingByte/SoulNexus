# Soul Pet CLI

本地开发 Soul Pet 包（`.soulpet` 规范）。

## 安装

```bash
cd packages/soul-pet-cli
npm link
# 或 node bin/soul-pet.mjs validate
```

## 环境变量

| 变量 | 说明 |
|------|------|
| `SOUL_PET_TOKEN` | Bearer token（Web 登录后从 localStorage 复制） |
| `SOUL_PET_SERVER` | API 根，默认 `http://127.0.0.1:7072/api` |

配置会写入 `.soulpet/link.json`。

## 命令

```bash
soul-pet init my-ghost --template live2d-stub   # 或 sprite-ghost（examples/soulpet/）
cd my-ghost
soul-pet validate
soul-pet push --create --name "Ghost"           # 创建「我的桌宠」→ jsSourceId
soul-pet push                                   # 更新已有项目
soul-pet pull
soul-pet publish --name "Ghost"                 # 发布到公开市场（marketId，无 jsSourceId）
soul-pet dev --port 5179                        # 本地预览
```

## 模板

- `sprite-ghost` — 帧动画（见 `examples/soulpet/sprite-ghost/`）
- `live2d-stub` — Live2D 占位 runtime
