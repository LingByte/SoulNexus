package prompt

const (
	// ContentTypeText represents text content type
	ContentTypeText = "text"
	// ContentTypeImage represents image content type
	ContentTypeImage = "image"
	// ContentTypeAudio represents audio content type
	ContentTypeAudio = "audio"
	// ContentTypeEmbeddedResource represents embedded resource content type
	ContentTypeEmbeddedResource = "embedded_resource"
)

// ProgressToken 是所有MCP进度标记的基本进度标记结构。
type ProgressToken interface{}

// Cursor is the base cursor struct for all MCP cursors.
type Cursor string

// Role 表示消息的发送者或接收者。
type Role string

// Request MCP（模型控制协议）请求的基础结构：
// ProgressToken 是一个空接口，表示进度令牌的通用结构。
// Request 是所有 MCP 请求的基类，包含请求方法和参数，其中参数可选地包含一个 _meta 字段，内部包含一个可选的 ProgressToken。
type Request struct {
	Method string `json:"method"`
	Params struct {
		Meta *struct {
			ProgressToken ProgressToken `json:"progressToken,omitempty"`
		} `json:"_meta,omitempty"`
	} `json:"params,omitempty"`
}

// Result 基础返回结果，包含一个可选的元信息字段
type Result struct {
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// PaginatedResult 分页返回结果，嵌套了 Result，并添加了可选的 NextCursor 字段，用于分页查询时标识下一页的位置。
type PaginatedResult struct {
	Result
	NextCursor Cursor `json:"nextCursor,omitempty"`
}

// Annotated describes an annotated resource.
type Annotated struct {
	// Annotations (optional)
	Annotations *struct {
		Audience []Role  `json:"audience,omitempty"`
		Priority float64 `json:"priority,omitempty"`
	} `json:"annotations,omitempty"`
}

// TextContent represents text content
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Annotated
}

// NewTextContent helpe functions for content creation
func NewTextContent(text string) TextContent {
	return TextContent{
		Type: ContentTypeText,
		Text: text,
	}
}

func (TextContent) isContent() {}

// ImageContent represents image content
type ImageContent struct {
	Type     string `json:"type"`
	Data     string `json:"data"` // base64 encoded image data
	MimeType string `json:"mimeType"`
	Annotated
}

func (ImageContent) isContent() {}

// NewImageContent creates a new image content
func NewImageContent(data string, mimeType string) ImageContent {
	return ImageContent{
		Type:     ContentTypeImage,
		Data:     data,
		MimeType: mimeType,
	}
}

// AudioContent represents audio content
type AudioContent struct {
	Type     string `json:"type"`
	Data     string `json:"data"` // base64 encoded audio data
	MimeType string `json:"mimeType"`
	Annotated
}

func (AudioContent) isContent() {}

// NewAudioContent creates a new audio content
func NewAudioContent(data string, mimeType string) AudioContent {
	return AudioContent{
		Type:     ContentTypeAudio,
		Data:     data,
		MimeType: mimeType,
	}
}

// Notification 表示一个通知，包含方法名 Method 和参数 Params。
type Notification struct {
	Method string             `json:"method"`
	Params NotificationParams `json:"params,omitempty"`
}

// NotificationParams 通知参数的基础结构，包含可选的元信息 Meta 和未指定的额外字段
type NotificationParams struct {
	Meta             map[string]interface{} `json:"_meta,omitempty"`
	AdditionalFields map[string]interface{} `json:"-"` // Additional fields that are not part of the MCP protocol.
}
