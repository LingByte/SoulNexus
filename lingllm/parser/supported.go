package parser

// SupportedDocumentFormat is one file type the upload pipeline can parse (see DetectFileType / DefaultRouter).
type SupportedDocumentFormat struct {
	Extension   string `json:"extension"`
	Description string `json:"description"`
}

// SupportedDocumentFormats lists extensions accepted for knowledge upload.
func SupportedDocumentFormats() []SupportedDocumentFormat {
	return []SupportedDocumentFormat{
		{".txt", "纯文本"},
		{".md", "Markdown"},
		{".markdown", "Markdown"},
		{".mdx", "MDX (Markdown with JSX)"},
		{".csv", "CSV"},
		{".html", "HTML"},
		{".htm", "HTML"},
		{".json", "JSON"},
		{".yaml", "YAML"},
		{".yml", "YAML"},
		{".eml", "邮件"},
		{".rtf", "RTF"},
		{".pdf", "PDF"},
		{".doc", "Word 97-2003 (.doc)"},
		{".docx", "Word"},
		{".pptx", "PowerPoint"},
		{".xlsx", "Excel"},
		{".epub", "EPUB 电子书"},
		{".mhtml", "MHTML 网页归档"},
		{".mht", "MHTML 网页归档"},
		{".png", "图片（OCR）"},
		{".jpg", "图片（OCR）"},
		{".jpeg", "图片（OCR）"},
		{".webp", "图片（OCR）"},
		{".gif", "图片（OCR）"},
		{".bmp", "图片（OCR）"},
		{".tif", "图片（OCR）"},
		{".tiff", "图片（OCR）"},
		{".wav", "音频（ASR）"},
		{".mp3", "音频（ASR）"},
		{".ogg", "音频（ASR）"},
		{".flac", "音频（ASR）"},
		{".m4a", "音频（ASR）"},
		{".aac", "音频（ASR）"},
	}
}

// SupportedDocumentNotes are caveats for operators / UI.
func SupportedDocumentNotes() []string {
	return []string{
		"旧版 .doc 使用 OLE 启发式提取，复杂排版可能不完整",
		"扫描版 PDF 无法提取文字，请先 OCR 或转换为文本 PDF",
		"图片 OCR 需后端以 ocr 构建标签并安装 Tesseract",
		"音频 ASR 需后端以 asr 构建标签、安装 libvosk，并设置 VOSK_MODEL 环境变量",
		"非 WAV/MP3 音频解码可能需要系统安装 ffmpeg",
	}
}
