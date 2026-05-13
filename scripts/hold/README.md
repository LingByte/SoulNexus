# scripts/hold/

运行时资源（不编译进二进制）。当前为对话面 WebSocket 断开重连时播放的**保持音**文案。

## hold_messages.json

网关在尝试重连对话面 WebSocket 时朗读的短语。可按语言 / 租户修改；`cmd/voice` 启动时读取该文件，若缺失或格式错误则回退到内置中文默认文案。

路径可通过 `-hold-messages` 指定（默认：`scripts/hold/hold_messages.json`）。

| Key | When it plays |
|---|---|
| `first_attempt` | Right after the WS dies, before the first redial |
| `retry` | Between subsequent retry attempts |
| `give_up` | Once after all retries fail; the call is then hung up |

短语走与 LLM 回复相同的 TTS 管线，每次合成有成本（首次使用后通常会缓存）。尽量保持简短。

## Adding a new prompt

1. 可选：把短语加入 `cmd/voice` 里 `prewarmTexts()`，以便启动时预热缓存、降低首包延迟。
2. 在 `hold_messages.json` 中增加对应 key。
3. 重启 `cmd/voice`。
