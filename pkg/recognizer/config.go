package recognizer

// AuthConfig represents authentication configuration
type AuthConfig struct {
	ResourceId string `json:"resource_id" yaml:"resource_id"`
	AccessKey  string `json:"access_key" yaml:"access_key"`
	AppKey     string `json:"app_key" yaml:"app_key"`
}

// Config represents the configuration for SAUC ASR client
type Config struct {
	// Connection settings
	URL string `json:"url" yaml:"url"`

	// Auth settings
	Auth AuthConfig `json:"auth" yaml:"auth"`

	// User metadata
	User UserConfig `json:"user" yaml:"user"`

	// Audio format settings
	Audio AudioConfig `json:"audio" yaml:"audio"`

	// Request settings
	Request RequestConfig `json:"request" yaml:"request"`

	// Buffer settings
	Buffer BufferConfig `json:"buffer" yaml:"buffer"`
}

// UserConfig represents user metadata configuration
type UserConfig struct {
	UID        string `json:"uid" yaml:"uid" default:"demo_uid"`
	DID        string `json:"did" yaml:"did"`
	Platform   string `json:"platform" yaml:"platform"`
	SDKVersion string `json:"sdk_version" yaml:"sdk_version"`
	APPVersion string `json:"app_version" yaml:"app_version"`
}

// AudioConfig represents audio format configuration
type AudioConfig struct {
	Format  string `json:"format" yaml:"format" default:"pcm"`
	Codec   string `json:"codec" yaml:"codec" default:"raw"`
	Rate    int    `json:"rate" yaml:"rate" default:"16000"`
	Bits    int    `json:"bits" yaml:"bits" default:"16"`
	Channel int    `json:"channel" yaml:"channel" default:"1"`
}

// RequestConfig represents request configuration
type RequestConfig struct {
	ModelName       string       `json:"model_name" yaml:"model_name" default:"bigmodel"`
	EnableITN       bool         `json:"enable_itn" yaml:"enable_itn" default:"true"`
	EnablePUNC      bool         `json:"enable_punc" yaml:"enable_punc" default:"true"`
	EnableDDC       bool         `json:"enable_ddc" yaml:"enable_ddc" default:"true"`
	ShowUtterances  bool         `json:"show_utterances" yaml:"show_utterances" default:"true"`
	EnableNonstream bool         `json:"enable_nonstream" yaml:"enable_nonstream" default:"false"`
	Corpus          CorpusConfig `json:"corpus" yaml:"corpus"`
}

// CorpusConfig represents corpus configuration
type CorpusConfig struct {
	BoostingTableName string `json:"boosting_table_name" yaml:"boosting_table_name"`
	CorrectTableName  string `json:"correct_table_name" yaml:"correct_table_name"`
	Context           string `json:"context" yaml:"context"`
}

// BufferConfig represents buffer configuration
type BufferConfig struct {
	SegmentDurationMs int `json:"segment_duration_ms" yaml:"segment_duration_ms" default:"200"`
	MaxBufferSize     int `json:"max_buffer_size" yaml:"max_buffer_size"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Auth: AuthConfig{
			ResourceId: "volc.bigasr.sauc.duration",
			AccessKey:  "",
			AppKey:     "",
		},
		User: UserConfig{
			UID: "demo_uid",
		},
		Audio: AudioConfig{
			Format:  "pcm",
			Codec:   "raw",
			Rate:    16000,
			Bits:    16,
			Channel: 1,
		},
		Request: RequestConfig{
			ModelName:       "bigmodel",
			EnableITN:       true,
			EnablePUNC:      true,
			EnableDDC:       true,
			ShowUtterances:  true,
			EnableNonstream: false,
			Corpus: CorpusConfig{
				Context: "",
			},
		},
		Buffer: BufferConfig{
			SegmentDurationMs: 200,
		},
	}
}

// WithURL sets the URL for the ASR service
func (c *Config) WithURL(url string) *Config {
	c.URL = url
	return c
}

// WithAuth sets the auth configuration
func (c *Config) WithAuth(auth AuthConfig) *Config {
	c.Auth = auth
	return c
}

// WithUser sets the user configuration
func (c *Config) WithUser(user UserConfig) *Config {
	c.User = user
	return c
}

// WithAudio sets the audio configuration
func (c *Config) WithAudio(audio AudioConfig) *Config {
	c.Audio = audio
	return c
}

// WithRequest sets the request configuration
func (c *Config) WithRequest(request RequestConfig) *Config {
	c.Request = request
	return c
}

// WithBuffer sets the buffer configuration
func (c *Config) WithBuffer(buffer BufferConfig) *Config {
	c.Buffer = buffer
	return c
}

// CalculateBufferSize calculates the buffer size based on audio format and segment duration
func (c *Config) CalculateBufferSize() int {
	if c.Buffer.MaxBufferSize > 0 {
		return c.Buffer.MaxBufferSize
	}
	// Calculate based on audio format: bytes per ms = (bits/8) * channels * rate / 1000
	bytesPerMs := (c.Audio.Bits / 8) * c.Audio.Channel * c.Audio.Rate / 1000
	return bytesPerMs * c.Buffer.SegmentDurationMs
}
