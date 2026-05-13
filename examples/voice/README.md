# 语音媒体面运行示例

在 **SoulNexus 仓库根目录** 下执行文档中的 `go run ./cmd/...` 命令（本目录保留文档与 `xiaozhi-client` 示例）。

`cmd/voice` 把**媒体面**（SIP / xiaozhi WS / WebRTC）和**对话面**（LLM + 业务逻辑）拆成两个独立进程，二者通过一条 WebSocket 通讯：

```
                                    ┌──────────────────────┐
  浏览器 / ESP32 / 话机 ──── 媒体 ───>│ cmd/voice (语音端)   │
                                    │  · SIP UAS / UAC      │
                                    │  · xiaozhi WS adapter │
                                    │  · WebRTC pion stack  │
                                    │  · ASR / TTS / 录音   │
                                    │  · 持久化 sqlite       │
                                    └──────────┬───────────┘
                                               │ -dialog-ws
                                               │ ws://… 双工 JSON
                                    ┌──────────▼───────────┐
                                    │ dialog-example (对话端)│
                                    │  · LLM 流式调用       │
                                    │  · 句子切分 → tts.speak│
                                    └──────────────────────┘
```

xiaozhi 和 WebRTC 共享 `cmd/voice` 的同一个 HTTP 监听器（`-http` 标志），WS / signaling / 浏览器 demo 都走这一个端口。

---

## 启动顺序

### 1. 对话端 — `cmd/dialog-example`

```bash
# .env 里需要：LLM_PROVIDER / LLM_APIKEY / LLM_APP_ID / LLM_MODEL
go run ./cmd/dialog-example \
    -addr 127.0.0.1:9090 \
    -path /ws/call
```

打印：

```
dialog-example LLM config: provider=openai model=gpt-4o-mini app_id="https://api.openai.com/v1"
dialog-example listening: ws://127.0.0.1:9090/ws/call
```

### 2. 语音端 — `cmd/voice`（在 SoulNexus 仓库根目录执行）

```bash
# .env 里需要：ASR_APPID / ASR_SECRET_ID / ASR_SECRET_KEY / TTS_*
go run ./cmd/voice \
    -sip 0.0.0.0:5060 -local-ip 192.168.3.69 \
    -rtp-start 31000 -rtp-end 31100 \
    -http 0.0.0.0:7080 \
    -dialog-ws ws://127.0.0.1:9090/ws/call \
    -record -record-bucket voiceserver-recordings
```

打印：

```
voiceserver ready: sip=udp:0.0.0.0:5060 ...
[xiaozhi] mounted: ws://<http>/xiaozhi/v1/ demo=/xiaozhi/demo -> dialog=ws://127.0.0.1:9090/ws/call (esp32+web)
[webrtc]  mounted: offer=/webrtc/v1/offer hangup=/webrtc/v1/hangup demo=/webrtc/v1/demo (ice=… public=…) -> dialog=…
[http]    listening on 0.0.0.0:7080 (xiaozhi=true webrtc=true)
```

> 注意：对话端 `-addr 127.0.0.1:9090` 和语音端 `-http 0.0.0.0:7080` **必须不同端口**，否则会撞车。

---

## 三种通话方式

打开浏览器访问 **http://192.168.3.69:7080/**，会看到一个索引页，列出两个浏览器 demo。

### 1. xiaozhi — 浏览器模拟 ESP32 设备

直接访问：

```
http://192.168.3.69:7080/xiaozhi/demo
```

页面用法：

- 点 **开始通话** → 浏览器请求麦克风权限 → 与 `cmd/voice` 建立 xiaozhi WS（`/xiaozhi/v1/`）+ 发 `hello{format:pcm, sr:16000}`
- 状态徽章变 `ready`
- 点 **开始说话**（对应 `listen:start`）→ 麦克风电平条实时跳动 → PCM 帧以 64 ms 节奏推到服务器
- 点 **结束说话**（对应 `listen:stop`）→ 等待服务器的 `stt` 文本 + `tts:start` + 二进制 PCM
- 对话气泡里左侧显示 ASR 听到的（"you"）、右侧显示 AI 回复（"ai"）
- AI 的语音通过 Web Audio API 播放，无需任何插件
- 点 **挂断** 关闭连接

真实 ESP32 走完全相同的协议（区别仅是 `format:opus` + cgo 解码）。

### 2. WebRTC — 浏览器走完整 WebRTC 1v1

直接访问：

```
http://192.168.3.69:7080/webrtc/v1/demo
```

- 浏览器与 `cmd/voice` 走完整的 ICE / DTLS-SRTP / Opus 协商
- 不需要切 mic state — getUserMedia 一开始就送 RTP，VAD 决定何时是用户在说话
- 实时统计面板每 2 秒刷新（RTT、收发包、丢包、抖动）
- AI 的语音通过 `<audio>` 元素自动播放

### 3. SIP — 用任意 SIP 客户端

```bash
baresip -e '/dial sip:demo@192.168.3.69:5060'
```

或 Linphone / MicroSIP / 软电话机等图形客户端，呼叫 `sip:demo@192.168.3.69:5060`。

### （备选）xiaozhi 命令行模拟设备

如果你想在脚本里跑而不是浏览器：

```bash
go run ./examples/voice/xiaozhi-client \
    -url ws://127.0.0.1:7080/xiaozhi/v1/ \
    -text "你好，请简单介绍一下你自己" \
    -out ai-reply.wav
```

CLI 用 Tencent TTS 把 `-text` 合成成"用户的声音"，推给 `cmd/voice`，把 AI 回复音频写到 `ai-reply.wav`。完整 AI 对话往返，全自动，不需要麦克风。

---

## 验证持久化

通话结束后查 SQLite（默认在进程当前工作目录下的 `voiceserver.db`），三种通话写入完全相同的 schema：

```bash
sqlite3 voiceserver.db "
  select call_id, transport, direction, codec, end_status, duration_sec
    from sip_calls order by id desc limit 5;
"

sqlite3 voiceserver.db "
  select call_id, kind, level, datetime(at) from call_events
    where call_id = (select call_id from sip_calls order by id desc limit 1)
    order by at;
"

sqlite3 voiceserver.db "
  select call_id, transport, format, layout, sample_rate, bytes, duration_ms
    from call_recording order by id desc limit 5;
"
```

每通话产出：

- 1 行 `sip_calls`（含 transport=sip/xiaozhi/webrtc）
- 若干 `call_events`（call.started / media.codec / asr.final / tts.end / dialog.hangup / call.terminated …）
- 1 行 `call_recording`（双声道 WAV：L=用户，R=AI）
- WebRTC 还会有若干 `call_media_stats`（每 5s 一行 + 一行 final=true）
