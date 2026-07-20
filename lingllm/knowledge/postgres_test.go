package knowledge

import "testing"

func TestPostgresHandler_TableName(t *testing.T) {
	h := &PostgresHandler{}
	table, err := h.tableName("My-KB/01")
	if err != nil {
		t.Fatal(err)
	}
	if table != pgTablePrefix+"my_kb_01" {
		t.Fatalf("unexpected table: %s", table)
	}
}

func TestPostgresHandler_TableName_EmptyNamespace(t *testing.T) {
	h := &PostgresHandler{}
	_, err := h.tableName("")
	if err != ErrCollectionNotFound {
		t.Fatalf("err=%v", err)
	}
}

func TestFillMissingVectors(t *testing.T) {
	recs := []Record{{Content: "hello"}, {Content: "world", Vector: []float32{1, 0, 0, 0}}}
	dim, err := fillMissingVectors(t.Context(), fakeEmbedder{dim: 4}, recs)
	if err != nil || dim != 4 {
		t.Fatalf("dim=%d err=%v", dim, err)
	}
	if len(recs[0].Vector) != 4 {
		t.Fatal("expected vector filled")
	}
}
