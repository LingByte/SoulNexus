package parser

// SupportedDocumentFormat is one file type the upload pipeline can parse (see DetectFileType / DefaultRouter).
type SupportedDocumentFormat struct {
	Extension   string `json:"extension"`
	Description string `json:"description"`
}

// SupportedDocumentFormats lists extensions accepted for knowledge upload (legacy .doc is rejected at parse time).
func SupportedDocumentFormats() []SupportedDocumentFormat {
	return []SupportedDocumentFormat{
		{".txt", "纯文本"},
		{".md", "Markdown"},
		{".markdown", "Markdown"},
		{".csv", "CSV"},
		{".html", "HTML"},
		{".htm", "HTML"},
		{".json", "JSON"},
		{".yaml", "YAML"},
		{".yml", "YAML"},
		{".eml", "邮件"},
		{".rtf", "RTF"},
		{".pdf", "PDF"},
		{".docx", "Word"},
		{".pptx", "PowerPoint"},
		{".xlsx", "Excel"},
		{".png", "图片（OCR）"},
		{".jpg", "图片（OCR）"},
		{".jpeg", "图片（OCR）"},
	}
}

// SupportedDocumentNotes are caveats for operators / UI.
func SupportedDocumentNotes() []string {
	return []string{
		"不支持旧版 Word .doc，请转换为 .docx 或 PDF 后上传",
		"图片（.png / .jpg / .jpeg）需后端以 ocr 构建标签并安装 Tesseract 后方可解析",
	}
}
