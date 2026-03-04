package synthesizer

const (
	TTS_QCLOUD            = "tts.qcloud"
	TTS_TENCENT           = "tts.tencent"
	TTS_XUNFEI            = "tts.xunfei"
	TTS_QINIU             = "tts.qiniu"
	TTS_BAIDU             = "tts.baidu"
	TTS_GOOGLE            = "tts.google"
	TTS_AWS               = "tts.aws"
	TTS_AZURE             = "tts.azure"
	TTS_OPENAI            = "tts.openai"
	TTS_ELEVENLABS        = "tts.elevenlabs"
	TTS_LOCAL             = "tts.local"
	TTS_LOCAL_GOSPEECH    = "tts.local_gospeech"
	TTS_FISHSPEECH        = "tts.fishspeech"
	TTS_FISHAUDIO         = "tts.fishaudio"
	TTS_COQUI             = "tts.coqui"
	TTS_VOLCENGINE        = "tts.volcengine"
	TTS_VOLCENGINE_CLONE  = "tts.volcengine_clone"
	TTS_VOLCENGINE_LLM    = "tts.volcengine_llm"
	TTS_VOLCENGINE_STREAM = "tts.volcengine_stream"
	TTS_MINIMAX           = "tts.minimax"
)

// TTSProvider TTS服务提供商类型
type TTSProvider string

const (
	// ProviderQiniu 七牛云TTS
	ProviderQiniu TTSProvider = "qiniu"
	// ProviderXunfei 讯飞TTS
	ProviderXunfei TTSProvider = "xunfei"
	// ProviderAliyun 阿里云TTS
	ProviderAliyun TTSProvider = "aliyun"
	// ProviderTencent 腾讯云TTS
	ProviderTencent TTSProvider = "qcloud"
	// ProviderBaidu 百度TTS
	ProviderBaidu TTSProvider = "baidu"
	// ProviderAzure 微软Azure TTS
	ProviderAzure TTSProvider = "azure"
	// ProviderGoogle Google Cloud TTS
	ProviderGoogle TTSProvider = "google"
	// ProviderAWS Amazon Polly TTS
	ProviderAWS TTSProvider = "aws"
	// ProviderOpenAI OpenAI TTS
	ProviderOpenAI TTSProvider = "openai"
	// ProviderElevenLabs ElevenLabs TTS
	ProviderElevenLabs TTSProvider = "elevenlabs"
	// ProviderLocal 本地TTS
	ProviderLocal TTSProvider = "local"
	// ProviderLocalGoSpeech 本地go-speech TTS
	ProviderLocalGoSpeech TTSProvider = "local_gospeech"
	// ProviderFishSpeech FishSpeech TTS
	ProviderFishSpeech TTSProvider = "fishspeech"
	// ProviderFishAudio Fish Audio TTS
	ProviderFishAudio TTSProvider = "fishaudio"
	// ProviderCoqui Coqui TTS
	ProviderCoqui TTSProvider = "coqui"
	// ProviderVolcengine 火山引擎标准TTS
	ProviderVolcengine TTSProvider = "volcengine"
	// ProviderVolcengineClone 火山引擎音色克隆TTS
	ProviderVolcengineClone TTSProvider = "volcengine_clone"
	// ProviderVolcengineLLM 火山引擎LLM TTS
	ProviderVolcengineLLM TTSProvider = "volcengine_llm"
	// ProviderVolcengineStream 火山引擎流式TTS
	ProviderVolcengineStream TTSProvider = "volcengine_stream"
	// ProviderMinimax Minimax TTS
	ProviderMinimax TTSProvider = "minimax"
)

func (tp TTSProvider) ToString() string {
	return string(tp)
}
