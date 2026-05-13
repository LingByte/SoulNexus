# pkg/voiceserver

独立语音媒体栈（原 `VoiceServer/pkg`），与 SoulNexus 业务包 `pkg/voice` 等**并列**，避免命名与职责混淆。

- **入口**：仓库根目录 `go run ./cmd/voice`（原 `voiceserver`）、`go run ./cmd/dialog-example`（对话面演示）。
- **数据库引导（可选）**：`pkg/voiceserver/dbconfig` + `cmd/voice-bootstrap`（原 `VoiceServer/cmd/bootstrap`）。
- **ASR / TTS**：与业务侧共用 `pkg/recognizer`、`pkg/synthesizer`（腾讯云流式 ASR 的句末边界逻辑在 `pkg/recognizer/qcloud.go` 内统一维护）。

import 路径形如：`github.com/LingByte/SoulNexus/pkg/voiceserver/<子包>`（不含 recognizer / synthesizer）。
