# Cubism 桌宠 JS 模板（一形象一文件）

每个 Live2D 形象对应 `templates/` 下**单独**一个 JS 文件（与 `lanlan.js` 相同用法），通过 JSTemplate 发布为各自的 `embed.js`。

| 模板文件 | 形象 | CDN 模型路径 | 说明 |
|----------|------|----------------|------|
| `haru-greeter.js` | Haru（招呼） | `live2d/haru/haru_greeter_t03.model3.json` | pixi-live2d-display 示例 |
| `haru.js` | Haru（官方样本） | `live2d/haru/Haru.model3.json` | `example/cubism-samples/Haru` |
| `hiyori.js` | Hiyori | `live2d/hiyori/Hiyori.model3.json` | 官方样本 |
| `mao.js` | Mao | `live2d/mao/Mao.model3.json` | 官方样本 |
| `mark.js` | Mark | `live2d/mark/Mark.model3.json` | 官方样本 |
| `natori.js` | Natori | `live2d/natori/Natori.model3.json` | 官方样本 |
| `ren.js` | Ren | `live2d/ren/Ren.model3.json` | 官方样本 |
| `rice.js` | Rice | `live2d/rice/Rice.model3.json` | 官方样本 |
| `wanko.js` | Wanko | `live2d/wanko/Wanko.model3.json` | 官方样本 |
| `shizuku.js` | Shizuku | `live2d/shizuku/shizuku.model.json` | Cubism 2 |

## 生成模板

逻辑源码：`templates/cubism-pet.template.js`（勿直接当 embed 用）。改完后：

```bash
go run ./cmd/tools/gencubismpet
```

预设列表在 `cmd/tools/gencubismpet/main.go`。

## 上传模型到 CDN

本地样本目录（gitignore）：`example/cubism-samples/{Haru,Hiyori,...}`，来源 [Live2D/CubismWebSamples](https://github.com/Live2D/CubismWebSamples) 的 `Samples/Resources`。

```bash
# 官方 Cubism 4 样本（目录名首字母大写，前缀小写）
for name in Hiyori Mao Mark Natori Ren Rice Wanko; do
  go run ./cmd/tools/storeupload -prefix "live2d/$(echo $name | tr '[:upper:]' '[:lower:]')" \
    "./example/cubism-samples/$name/"
done

# 官方 Haru（与 greeter 同前缀 live2d/haru/）
go run ./cmd/tools/storeupload -prefix live2d/haru ./example/cubism-samples/Haru/
```

SDK 与已上传的 greeter / Shizuku 见 `live2d/sdk/`（含 `cubism2.min.js`、`cubism4.min.js`）、`live2d/haru/`、`live2d/shizuku/`。

**注意**：Cubism 2（Shizuku）与 Cubism 4 使用不同插件；同一页面切换预览时需整页刷新，否则会因全局 `PIXI` 冲突报错。

## 发布

每个 `templates/<形象>.js` 单独创建 JSTemplate 并注入桌面宠；**不要**再使用已删除的多角色合并文件 `cubism-pet.js`。
