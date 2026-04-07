```bash
brew install autoconf automake libtool
git clone https://github.com/xiph/rnnoise.git
cd rnnoise
./autogen.sh && ./configure && make
sudo make install   # 常见结果：/usr/local/include/rnnoise.h、/usr/local/lib/librnnoise.*
```

说明：若脚本里用 `sha256sum` 校验，macOS 可能没有该命令，一般会跳过校验；需要可自行用 `shasum -a 256 文件` 对照 `model_version` 里的哈希。

若你使用 **`./configure --prefix=$HOME/rnnoise`** 等非标准路径，请设置：

```bash
export CGO_CFLAGS="-I$HOME/rnnoise/include"
export CGO_LDFLAGS="-L$HOME/rnnoise/lib -lrnnoise"
```

**MacPorts**

```bash
sudo port install rnnoise
```

然后：

```bash
export CGO_ENABLED=1
go build -tags rnnoise ./pkg/rnnoise/...
```

### Debian / Ubuntu

```bash
sudo apt-get install -y librnnoise-dev
go build -tags rnnoise ./pkg/rnnoise/...
```

若库在非标准路径，可设置例如：

```bash
export CGO_CFLAGS="-I/custom/include"
export CGO_LDFLAGS="-L/custom/lib -lrnnoise"
```

### 未使用 `-tags rnnoise`

桩实现会编译通过；`New()` 返回 `ErrUnavailable`；`FrameSamples` 仍返回 480，便于占位。生产环境启用降噪时请带 `-tags rnnoise` 并安装库。

## API 摘要

| 函数/方法 | 说明 |
|-----------|------|
| `New()` | 创建 denoiser（默认模型） |
| `(*Denoiser) Close()` | 释放原生状态 |
| `FrameSamples()` | `rnnoise_get_frame_size()`，一般为 480 |
| `FrameBytes()` | `FrameSamples() * 2`（int16 字节数） |
| `ProcessPCM16LE([]byte)` | 单帧输入/输出，长度须等于 `FrameBytes()` |
| `Process([]byte)` | 连续多帧 + 尾部原样透传 |

## 可运行示例

仓库内 **`examples/rnnoise-wav`**：读入 **48 kHz / mono / PCM16** 的 WAV，整段 `Process` 后写出。未带 `-tags rnnoise` 时会**原样复制**并打印提示。

```bash
# 生成测试输入（需 ffmpeg）
ffmpeg -f lavfi -i "sine=frequency=440:duration=2" -ar 48000 -ac 1 -c:a pcm_s16le in.wav

# 真实降噪（需已安装 librnnoise）；-v 在 stderr 打印与输入的样本差异统计（确认是否在处理）
CGO_ENABLED=1 go run -tags rnnoise ./examples/rnnoise-wav/ -v -- in.wav out.wav

# 无库时：仍可运行，输出等于输入
go run ./examples/rnnoise-wav/ -- in.wav out.wav
```

听感上「差别不大」常见于：素材本身干净、或纯正弦/非语音；RNNoise 主要针对**语音上的噪声**。可用 `-v` 看 `changed` 是否远大于 0；若为 0 则输出与输入完全相同（需检查是否误用桩构建）。

**想听感更明显**：尽量录**真人说话**，并在**有底噪的环境**（风扇、街道、多人房间）录；或录好后用下面「人声 + 噪声」叠一层再降噪。

### macOS：用 ffmpeg 从麦克风录 48 kHz mono WAV

需已安装 ffmpeg（如 `brew install ffmpeg`）。先列设备，确认麦克风对应的 **audio 索引**（多为 `0`）：

```bash
ffmpeg -f avfoundation -list_devices true -i "" 2>&1
```

只录声音、不录画面（`none` 表示不要视频输入；若报错可改为 `":0"` 或把 `none` 换成列表里显示的索引）：

```bash
ffmpeg -f avfoundation -i "none:0" -t 15 -ar 48000 -ac 1 -c:a pcm_s16le voice_in.wav
```

然后：

```bash
CGO_ENABLED=1 go run -tags rnnoise ./examples/rnnoise-wav/ -v -- voice_in.wav voice_out.wav
```

**叠噪再降噪（对比更明显）**：在人声上叠一层粉噪（`duration=first` 以人声长度为准；可调 `weights` 里第二项加大噪声）。

```bash
ffmpeg -y -i voice_in.wav -f lavfi -i "anoisesrc=color=pink:amplitude=0.35:sample_rate=48000" \
  -filter_complex "amix=inputs=2:duration=first:weights=1 0.4" -ar 48000 -ac 1 -c:a pcm_s16le voice_noisy.wav
```

再对 `voice_noisy.wav` 跑 `rnnoise-wav` 得到 `voice_out.wav`，与 `voice_noisy.wav` 对比听背景嘶声。

## 许可

- 本 Go 封装：项目仓库许可证（AGPL-3.0，见文件头）。
- **RNNoise** 以原项目许可证为准（BSD 风格，见 upstream）。
