package models

import (
	"testing"

	apperror "github.com/LingByte/SoulNexus/pkg/errors"
)

func TestListPage_NilDB(t *testing.T) {
	_, _, err := ListPage(nil, 1, 20, "")
	if err == nil {
		t.Error("expected error for nil db")
	}
}

func TestCreate_EmptyKey(t *testing.T) {
	_, err := Create(nil, CreateInput{Key: ""})
	if err != apperror.ErrConfigKeyRequired {
		t.Errorf("expected ErrConfigKeyRequired, got %v", err)
	}
}

func TestCreate_InvalidFormat(t *testing.T) {
	_, err := Create(nil, CreateInput{Key: "TEST", Format: ""})
	if err != apperror.ErrConfigFormatInvalid {
		t.Errorf("expected ErrConfigFormatInvalid, got %v", err)
	}
}
