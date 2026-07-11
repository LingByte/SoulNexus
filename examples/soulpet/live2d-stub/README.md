# Soul Pet 示例 — Live2D 占位

将 Cubism 导出的 `model3.json`、`.moc3`、纹理放入 `assets/live2d/`。

当前 `pet.js` 为 **Live2D 兼容占位 runtime**（Canvas 预览 + `__PET_LIVE2D__` API）。
接入完整 Cubism Web SDK 时，只需替换 `pet.js` 内渲染逻辑，**manifest 与 SDK API 不变**。

```javascript
window.__SOUL_PET__.setEmotion('happy')
window.__SOUL_PET__.playAnimation('idle')
window.__SOUL_PET__.chat('你好')
```
