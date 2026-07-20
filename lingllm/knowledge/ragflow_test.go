package knowledge

import (
	"context"
	"net/http"
	"testing"
)

func TestRAGFlowHandler_Provider(t *testing.T) {
	rh := &RAGFlowHandler{}
	if got := rh.Provider(); got != ProviderRAGFlow {
		t.Fatalf("want %s, got %s", ProviderRAGFlow, got)
	}
}

func TestRAGFlowHandler_NilHandler(t *testing.T) {
	var rh *RAGFlowHandler
	ctx := context.Background()

	if _, err := rh.Query(ctx, "test", nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := rh.Upsert(ctx, []Record{{ID: "1", Content: "test"}}, nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if _, err := rh.Get(ctx, []string{"1"}, nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if _, err := rh.List(ctx, nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := rh.Delete(ctx, []string{"1"}, nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := rh.Ping(ctx); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := rh.CreateNamespace(ctx, "test"); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := rh.DeleteNamespace(ctx, "test"); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if _, err := rh.ListNamespaces(ctx); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}
}

func TestRAGFlowHandler_EmptyQuery(t *testing.T) {
	rh := &RAGFlowHandler{}
	ctx := context.Background()

	if _, err := rh.Query(ctx, "", nil); err != ErrEmptyQuery {
		t.Fatalf("want ErrEmptyQuery, got %v", err)
	}

	if _, err := rh.Query(ctx, "   ", nil); err != ErrEmptyQuery {
		t.Fatalf("want ErrEmptyQuery, got %v", err)
	}
}

func TestRAGFlowHandler_EmptyNamespace(t *testing.T) {
	rh := &RAGFlowHandler{}
	ctx := context.Background()

	if err := rh.CreateNamespace(ctx, ""); err != ErrNamespaceNotFound {
		t.Fatalf("want ErrNamespaceNotFound, got %v", err)
	}

	if err := rh.DeleteNamespace(ctx, ""); err != ErrNamespaceNotFound {
		t.Fatalf("want ErrNamespaceNotFound, got %v", err)
	}
}

func TestRAGFlowHandler_EmptyRecordID(t *testing.T) {
	rh := &RAGFlowHandler{
		HTTPClient: &http.Client{},
	}
	ctx := context.Background()

	if err := rh.Upsert(ctx, []Record{{ID: "", Content: "test"}}, nil); err == nil {
		t.Fatalf("expected error for empty record id")
	}
}
