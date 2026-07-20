package models

import (
	"strings"
	"testing"
)

func TestHTMLToPlainText(t *testing.T) {
	got := HTMLToPlainText("<p>Hi &amp; {{name}}</p>")
	if !strings.Contains(got, "Hi & {{name}}") {
		t.Fatalf("%q", got)
	}
}

func TestDeriveTemplateVariables(t *testing.T) {
	got := DeriveTemplateVariables("Hello {{foo}}", "{{bar}}")
	if !strings.Contains(got, `"foo"`) || !strings.Contains(got, `"bar"`) {
		t.Fatalf("%q", got)
	}
}

func TestCreateMailTemplate_nil(t *testing.T) {
	if err := CreateMailTemplate(nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestMailTemplateResolveVariables_derives(t *testing.T) {
	got := resolveMailTemplateVariables("{{x}}", "", "")
	if got == "" {
		t.Fatal("expected derived variables")
	}
}
