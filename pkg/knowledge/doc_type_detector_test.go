package knowledge

import (
	"context"
	"testing"
)

func TestRuleBasedDocumentTypeDetector_StructuredMarkdown(t *testing.T) {
	d := &RuleBasedDocumentTypeDetector{}
	dt, err := d.DetectDocumentType(context.Background(), "# Title\n\n## Sec\nhello")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if dt != DocumentTypeStructured {
		t.Fatalf("want structured, got %v", dt)
	}
}

func TestRuleBasedDocumentTypeDetector_TableMarkdown(t *testing.T) {
	d := &RuleBasedDocumentTypeDetector{}
	text := `
| Name | Age |
| ---- | --- |
| A    |  10 |
| B    |  20 |
`
	dt, err := d.DetectDocumentType(context.Background(), text)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if dt != DocumentTypeTableKV {
		t.Fatalf("want table/kv, got %v", dt)
	}
}

func TestRuleBasedDocumentTypeDetector_KVForm(t *testing.T) {
	d := &RuleBasedDocumentTypeDetector{}
	text := `
姓名：张三
年龄：18
日期：2026-05-02
项目：LingVoice
地址：深圳
电话：123
邮箱：a@b.com
学校：X
专业：Y
`
	dt, err := d.DetectDocumentType(context.Background(), text)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if dt != DocumentTypeTableKV {
		t.Fatalf("want table/kv, got %v", dt)
	}
}

func TestRuleBasedDocumentTypeDetector_UnstructuredNoPunct(t *testing.T) {
	d := &RuleBasedDocumentTypeDetector{}
	text := "这是一个很长很长的文本但是几乎没有标点也没有段落也没有换行它可能来自ocr扫描或者小说直出" +
		"继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续" +
		"继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续继续"
	dt, err := d.DetectDocumentType(context.Background(), text)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if dt != DocumentTypeUnstructured {
		t.Fatalf("want unstructured, got %v", dt)
	}
}

