# LingLLM

A Go framework for building LLM applications and real-time voice agents. It combines a
provider-agnostic LLM core, a full RAG stack, and an extensive real-time speech and
telephony layer (ASR, TTS, VAD, voice cloning, voiceprint, WebRTC/voice) behind unified,
strongly-typed interfaces.

Version: 1.4.5 · Go 1.26 · ~77k lines of code with ~37k lines of tests.

## Features

**LLM core**
- Multi-provider chat behind one interface — OpenAI, Anthropic, Ollama
- Tool / function calling with automatic multi-round execution
- Streaming responses with event-based processing
- Composable chain pipeline for multi-step workflows
- Conversation memory and prompt management
- Built-in metrics for latency, token usage, and error rates

**Retrieval (RAG)**
- Multi-provider embeddings — OpenAI, Ollama, Nvidia, DashScope, Local
- Bleve-powered full-text search with facets, highlighting, suggestions
- Multi-strategy retrieval — vector, keyword, hybrid — with reranking
- Document chunking with multiple strategies
- Knowledge base over Qdrant and Milvus
- Document parsing (PDF, Office, OCR via gosseract)

**Real-time voice**
- ASR (speech-to-text) — 13+ engines: AWS, Baidu, Deepgram, FunASR, Gladia, Google,
  Tencent Cloud, Volcengine, Whisper, and more
- TTS (text-to-speech) — 16+ engines: Azure, AWS, Google, OpenAI, ElevenLabs, MiniMax,
  FishAudio, Coqui, Volcengine, Xunfei, Aliyun, Tencent, Qiniu, Baidu, and local
- VAD (voice activity detection) — Volcengine, Xunfei
- Voice cloning and voiceprint (speaker) recognition
- Real-time conversational agents — Aliyun Omni, Volcengine dialogue
- Audio media pipeline — codecs via Rust lecodec (`-tags lingcodec` under `media/encoder/lecodec`) or legacy Go/hraban; resampling, denoise (RNNoise), routing

**Telephony**
- voice signaling stack and voice media handling
- WebRTC / RTP / SRTP / DTLS transport built on pion

## Installation

```bash
go get github.com/LingByte/lingllm
```

## Project Structure

```
lingllm/
├── protocol/        # Core LLM types, provider clients, and the media/voice stack
│   ├── types.go     # ChatRequest, ChatResponse, Message, Tool, ChatStream
│   ├── factory.go   # Provider factory
│   ├── stream.go    # Streaming utilities and transformers
│   ├── openai/      # OpenAI client
│   ├── anthropic/   # Anthropic client
│   ├── ollama/      # Ollama client
│   ├── voice/       # Voice session protocol
│   ├── sip/         # voice signaling stack
│   └── sipmedia/    # voice media handling
├── chain/           # Chain-based processing pipeline
├── tools/           # Tool definitions, executors, and tool chains
├── prompt/          # Prompt templates and management
├── memory/          # Conversation memory (single and layered)
├── metrics/         # Call metrics and monitoring
│
├── embedder/        # Text embedding providers (OpenAI, Ollama, Nvidia, DashScope, Local)
├── search/          # Bleve full-text search engine
├── retrieve/        # Multi-strategy retrieval (vector, keyword, hybrid)
├── rerank/          # Document reranking
├── chunk/           # Document chunking strategies
├── knowledge/       # Knowledge base over Qdrant / Milvus
├── parser/          # Document parsing (PDF, Office, OCR)
├── cache/           # Caching layer
│
├── recognizer/      # ASR engines (speech-to-text)
├── synthesizer/     # TTS engines (text-to-speech)
├── vad/             # In-process VAD (energy + local/webrtc|silero|levad); no remote Python service
├── voiceclone/      # Voice cloning
├── voiceprint/      # Speaker (voiceprint) recognition
├── realtime/        # Real-time conversational agents
├── media/           # Audio media pipeline (codec, resample, denoise, routing)
│
├── utils/           # Shared text/audio utilities
├── shared/          # Shared helpers
├── examples/        # Runnable demos for each module
└── version/         # Build version info
```

## LLM Core

### Basic Chat

```go
package main

import (
	"context"
	"fmt"

	"github.com/LingByte/lingllm/protocol"
)

func main() {
	req := protocol.NewChatRequest(
		"gpt-4",
		protocol.UserMessage("What is the capital of France?"),
	)

	// Call your provider implementation:
	// resp, err := model.Chat(context.Background(), *req)
	// fmt.Println(resp.FirstContent())
	_ = req
	_ = context.Background
	_ = fmt.Println
}
```

Build requests fluently:

```go
req := protocol.NewChatRequest("gpt-4",
	protocol.SystemMessage("You are a helpful assistant"),
	protocol.UserMessage("Hello"),
).
	WithMaxTokens(1000).
	WithTemperature(0.7).
	WithTopP(0.9).
	WithStop("END")
```

### Tool Calling

```go
executor := tools.NewSimpleToolExecutor()

weatherTool := tools.WeatherTool()
executor.RegisterTool(weatherTool, func(args json.RawMessage) (string, error) {
	return "Sunny, 72°F", nil
})

toolChain := tools.NewToolChain(model, executor)
toolChain.WithMaxRounds(5)

req := protocol.NewChatRequest("gpt-4",
	protocol.UserMessage("What's the weather in San Francisco?"))

resp, err := toolChain.ExecuteWithTools(context.Background(), *req)
if err != nil {
	panic(err)
}
fmt.Println(resp.FirstContent())
```

### Streaming

```go
stream, err := model.StreamChat(context.Background(), *req)
if err != nil {
	panic(err)
}
defer stream.Close()

for {
	chunk, err := stream.Recv()
	if err == io.EOF {
		break
	}
	if err != nil {
		panic(err)
	}
	fmt.Print(chunk.Delta)
}
```

### Chains

```go
c := chain.NewBuilder("my-chain").
	AddModel("model1", model1).
	AddProcessor("processor1", func(ctx context.Context, resp *protocol.ChatResponse) (*protocol.ChatResponse, error) {
		return resp, nil
	}).
	AddModel("model2", model2).
	Build()

resp, err := c.Invoke(context.Background(), *protocol.NewChatRequest("gpt-4", protocol.UserMessage("Hello")))
if err != nil {
	panic(err)
}
println(resp.FirstContent())
```

## Retrieval (RAG)

### Embeddings

```go
cfg := &embedder.Config{
	Provider: "openai",
	Model:    "text-embedding-3-small",
	APIKey:   os.Getenv("OPENAI_API_KEY"),
}

emb, err := embedder.Create(context.Background(), cfg)
if err != nil {
	panic(err)
}
defer emb.Close()

vec, _ := emb.EmbedSingle(context.Background(), "Hello world")
vecs, _ := emb.Embed(context.Background(), []string{"Hello world", "Goodbye world"})
fmt.Printf("dim=%d, batch=%d\n", len(vec), len(vecs))
```

### Full-Text Search

```go
cfg := search.Config{
	IndexPath:           "./search_index",
	DefaultAnalyzer:     "standard",
	DefaultSearchFields: []string{"title", "body"},
}
engine, err := search.New(cfg, search.BuildIndexMapping("standard"))
if err != nil {
	panic(err)
}
defer engine.Close()

engine.IndexBatch(context.Background(), []search.Doc{{
	ID:   "1",
	Type: "article",
	Fields: map[string]interface{}{
		"title": "Go Programming",
		"body":  "Go is a fast and efficient language",
	},
}})

result, _ := engine.Search(context.Background(), search.SearchRequest{Keyword: "Go", Size: 10})
fmt.Printf("Found %d results\n", result.Total)
```

### Hybrid Retrieval

```go
retriever, err := retrieve.New(retrieve.Config{
	Strategy:     retrieve.StrategyHybrid,
	Vector:       vectorStore,
	Search:       searchEngine,
	TopK:         10,
	VectorWeight: 0.65,
})
if err != nil {
	panic(err)
}

docs, _ := retriever.Retrieve(context.Background(), "machine learning", 10)
for i, doc := range docs {
	fmt.Printf("%d. %s (score: %.2f)\n", i+1, doc.Content, doc.Score)
}
```

### Knowledge Base

```go
emb, _ := embedder.Create(context.Background(), &embedder.Config{
	Provider: "openai",
	Model:    "text-embedding-3-small",
	APIKey:   os.Getenv("OPENAI_API_KEY"),
})

searcher, _ := search.New(search.Config{
	IndexPath:           "./search_index",
	DefaultSearchFields: []string{"title", "content"},
}, search.BuildIndexMapping("standard"))

handler, _ := knowledge.NewKnowledgeHandler(knowledge.HandlerFactoryParams{
	Provider: knowledge.ProviderQdrant,
	QdrantConfig: &knowledge.QdrantConfig{
		BaseURL: "http://localhost:6333",
		APIKey:  "your-api-key",
	},
})

kb, _ := knowledge.NewKnowledgeBase(knowledge.KnowledgeBaseConfig{
	Handler:  handler,
	Embedder: emb,
	Searcher: searcher,
})
defer kb.Close()

kb.AddDocument(context.Background(), "doc1", "Title", "Content...", nil)

results, _ := kb.Query(context.Background(), "search query", 10)
for _, r := range results {
	fmt.Printf("%s (score: %.2f)\n", r.Record.Title, r.Score)
}
```

## Real-time Voice

### Speech Recognition (ASR)

All ASR engines implement `recognizer.SpeechRecognitionEngine`, created through a factory
keyed by vendor. Recognition is callback-driven: you feed audio bytes in and receive
transcript results as they arrive.

```go
// SpeechRecognitionEngine interface:
//   Init(resultCb SpeechRecognitionResult, errorCb RecognitionError)
//   Vendor() string
//   ConnAndReceive(dialogId string) error
//   SendAudioBytes(data []byte) error
//   SendEnd() error
//   StopConn() error

factory := recognizer.NewTranscriberFactory()

cfg, _ := recognizer.NewTranscriberConfigFromMap("qcloud", map[string]interface{}{
	"appId":     "your-app-id",
	"secretId":  "your-secret-id",
	"secretKey": "your-secret-key",
})

engine, err := factory.CreateTranscriber(cfg)
if err != nil {
	panic(err)
}

engine.Init(
	func(text string, isLast bool, duration time.Duration, uuid string) {
		fmt.Printf("[%v] %s (final=%t)\n", duration, text, isLast)
	},
	func(err error, isFatal bool) {
		fmt.Printf("asr error (fatal=%t): %v\n", isFatal, err)
	},
)

if err := engine.ConnAndReceive("dialog-1"); err != nil {
	panic(err)
}

engine.SendAudioBytes(pcmFrame) // feed 16kHz PCM frames
engine.SendEnd()
engine.StopConn()
```

Supported vendors: `qcloud`, `google`, `aws`, `baidu`, `deepgram`, `gladia`, `whisper`,
`funasr`, `funasr_realtime`, `volcengine`, `volcengine_llm`, and more. Call
`factory.GetSupportedVendors()` for the full list at runtime.

### Speech Synthesis (TTS)

TTS engines implement `synthesizer.AudioSynthesisEngine` and stream audio through an
`AudioSynthesisHandler` callback.

```go
// AudioSynthesisEngine interface:
//   Provider() TTSProvider
//   Format() media.StreamFormat
//   Synthesize(ctx, handler AudioSynthesisHandler, text string) error
//   Close() error

engine, err := synthesizer.NewAudioSynthesisEngine("elevenlabs", map[string]any{
	"apiKey":  os.Getenv("ELEVENLABS_API_KEY"),
	"voiceId": "your-voice-id",
})
if err != nil {
	panic(err)
}
defer engine.Close()

err = engine.Synthesize(context.Background(), myHandler, "Hello from LingLLM")
```

`myHandler` implements:

```go
type AudioSynthesisHandler interface {
	OnMessage([]byte)                    // receive audio chunks
	OnTimestamp(ts SentenceTimestamp)    // receive word/sentence timing
}
```

Supported providers: `qiniu`, `xunfei`, `aliyun`, `qcloud`, `baidu`, `azure`, `google`,
`aws`, `openai`, `elevenlabs`, `minimax`, `fishspeech`, `fishaudio`, `coqui`,
`volcengine`, `volcengine_clone`, `local`, `local_gospeech`.

### Other Voice Modules

- **VAD** — `vad.NewDefaultFactory(logger)` creates voice-activity detectors (Volcengine, Xunfei).
- **Voice cloning** — `voiceclone.NewFactory()` for clone workflows (Volcengine, Xunfei).
- **Voiceprint** — `voiceprint.NewService(config, cache)` for speaker enrollment and identification.
- **Real-time agents** — `realtime.NewAgentFromCredential(cfg, opts)` for full-duplex
  conversational agents (Aliyun Omni, Volcengine dialogue).
- **Media pipeline** — the `media` package provides codecs, resampling, RNNoise denoise,
  low-pass filtering, routing, and an event bus for assembling audio processing stages.

## Examples

Runnable demos live under [`examples/`](examples/):

| Demo | Covers |
| --- | --- |
| `anthropic-demo`, `openai-demo`, `ollama-demo` | Provider chat clients |
| `tools-demo` | Tool / function calling |
| `chain-demo` | Chain pipelines |
| `prompt-demo` | Prompt templates |
| `memory-demo`, `memory-layers-demo` | Conversation memory |
| `embedder-demo` | Multi-provider embeddings |
| `search-demo` | Full-text search |
| `chunk-demo` | Document chunking |
| `knowledge-demo`, `qdrant-demo` | Knowledge base |
| `response-demo`, `batch-processing-demo` | Response handling / batching |
| `voice-demo` | Voice session |
| `voiceclone-volcengine-demo`, `voiceclone-xunfei-demo` | Voice cloning |
| `sip-uas-demo`, `sip-outbound-demo`, `sip-signaling-server`, `sip-rtp-server`, `sip-split` | voice telephony |

## Core Interfaces

| Interface | Package | Purpose |
| --- | --- | --- |
| `ChatModel` | `protocol` | Language model abstraction |
| `ChatStream` | `protocol` | Streaming responses |
| `Tool` / `ToolExecutor` | `tools` | Tool definitions and execution |
| `Chain` / `Node` | `chain` | Composable processing pipeline |
| `Embedder` | `embedder` | Text embedding |
| `SpeechRecognitionEngine` | `recognizer` | ASR engines |
| `AudioSynthesisEngine` | `synthesizer` | TTS engines |
| `Agent` | `realtime` | Real-time conversational agents |

## Testing

```bash
go test ./...           # run all tests
go test -cover ./...    # with coverage
```

The `embedder`, `search`, `retrieve`, and `knowledge` modules carry high coverage
(80%+, search at 96%+). Voice and voice modules include extensive integration tests.

## Contributing

Contributions are welcome:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## License

GNU Affero General Public License v3.0 (AGPL-3.0) — see the [LICENSE](LICENSE) file for details.
