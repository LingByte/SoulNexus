package knowledge

import (
	"context"
	"testing"
)

func TestAliyunHandler_Provider(t *testing.T) {
	ah := &AliyunHandler{}
	if got := ah.Provider(); got != ProviderAliyun {
		t.Fatalf("want %s, got %s", ProviderAliyun, got)
	}
}

func TestAliyunHandler_NilHandler(t *testing.T) {
	var ah *AliyunHandler
	ctx := context.Background()

	if _, err := ah.Query(ctx, "test", nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := ah.Upsert(ctx, []Record{{ID: "1", Content: "test"}}, nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if _, err := ah.Get(ctx, []string{"1"}, nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if _, err := ah.List(ctx, nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := ah.Delete(ctx, []string{"1"}, nil); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := ah.Ping(ctx); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := ah.CreateNamespace(ctx, "test"); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if err := ah.DeleteNamespace(ctx, "test"); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}

	if _, err := ah.ListNamespaces(ctx); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}
}

func TestAliyunHandler_EmptyQuery(t *testing.T) {
	ah := &AliyunHandler{}
	ctx := context.Background()

	if _, err := ah.Query(ctx, "", nil); err != ErrEmptyQuery {
		t.Fatalf("want ErrEmptyQuery, got %v", err)
	}

	if _, err := ah.Query(ctx, "   ", nil); err != ErrEmptyQuery {
		t.Fatalf("want ErrEmptyQuery, got %v", err)
	}
}

func TestAliyunHandler_EmptyNamespace(t *testing.T) {
	ah := &AliyunHandler{}
	ctx := context.Background()

	if err := ah.CreateNamespace(ctx, ""); err != ErrNamespaceNotFound {
		t.Fatalf("want ErrNamespaceNotFound, got %v", err)
	}

	if err := ah.DeleteNamespace(ctx, ""); err != ErrNamespaceNotFound {
		t.Fatalf("want ErrNamespaceNotFound, got %v", err)
	}
}

func TestAliyunHandler_EmptyRecordID(t *testing.T) {
	ah := &AliyunHandler{}
	ctx := context.Background()

	if err := ah.Upsert(ctx, []Record{{ID: "", Content: "test"}}, nil); err == nil {
		t.Fatalf("expected error for empty record id")
	}
}

func TestAliyunHandler_DeleteNotSupported(t *testing.T) {
	ah := &AliyunHandler{}
	ctx := context.Background()

	if err := ah.Delete(ctx, []string{"1"}, nil); err == nil {
		t.Fatalf("expected error for delete operation")
	}
}
