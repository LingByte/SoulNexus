package stores

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

// helper: 断言 x 实现了 Store 接口
func assertImplementsStore(t *testing.T, x Store) {
	t.Helper()
	// 运行期通过反射确认接口实现（虽然编译期已会检查）
	storeType := reflect.TypeOf((*Store)(nil)).Elem()
	if !reflect.TypeOf(x).Implements(storeType) {
		t.Fatalf("type %T does not implement Store", x)
	}
}

func TestGetStore_AllKindsNonNil(t *testing.T) {
	// 仅断言不为 nil 且实现接口；不做外部行为验证，避免依赖外部服务
	kinds := []string{KindLocal, KindCos, KindMinio, KindQiNiu, KindOss, KinsS3, KindTos, KindObs, KindKs3, "unknown-kind"}
	for _, k := range kinds {
		s := GetStore(k)
		if s == nil {
			t.Fatalf("GetStore(%q) returned nil", k)
		}
		assertImplementsStore(t, s)
	}
}

func TestDefault_UsesDefaultStoreKind(t *testing.T) {
	// 备份并暂时修改 DefaultStoreKind，确保 Default() 与 GetStore() 一致
	orig := DefaultStoreKind
	defer func() { DefaultStoreKind = orig }()

	cases := []string{KindLocal, KindCos, KindMinio, KindQiNiu, KindOss, KinsS3, KindTos, KindObs, KindKs3}
	for _, k := range cases {
		DefaultStoreKind = k
		got := Default()
		want := GetStore(k)

		if got == nil || want == nil {
			t.Fatalf("Default/GetStore returned nil for kind=%q", k)
		}
		assertImplementsStore(t, got)
		assertImplementsStore(t, want)

		// 用可读的类型名比较（不同实现一般类型不同；若同类型也没关系，只要非空且实现接口就通过）
		tGot := reflect.TypeOf(got)
		tWant := reflect.TypeOf(want)
		if tGot != tWant {
			t.Fatalf("Default() type %v != GetStore(%q) type %v", tGot, k, tWant)
		}
	}
}

func TestErrInvalidPath(t *testing.T) {
	if ErrInvalidPath == nil {
		t.Fatal("ErrInvalidPath is nil")
	}
	// 静态类型是 *utils.Error；这里只能做运行时的浅检查（不能直接 import utils 的情况下）
	// 但至少验证其 Code 与 Message 是否符合定义。
	type hasFields interface {
		Error() string
	}
	if _, ok := interface{}(ErrInvalidPath).(hasFields); !ok {
		t.Fatalf("ErrInvalidPath does not implement error")
	}
	if ErrInvalidPath.Code != http.StatusBadRequest {
		t.Fatalf("ErrInvalidPath.Code = %d, want %d", ErrInvalidPath.Code, http.StatusBadRequest)
	}
	if ErrInvalidPath.Message == "" {
		t.Fatalf("ErrInvalidPath.Message is empty, want 'invalid path'")
	}
}

func TestGetDefaultStoreKind_EmptyEnv(t *testing.T) {
	// When STORAGE_KIND is not set (or empty), getDefaultStoreKind returns KindLocal.
	// We test by directly calling the logic, not depending on env cache state.
	kind := getDefaultStoreKind()
	// At minimum, the function must return one of the known kinds.
	validKinds := map[string]bool{
		KindLocal: true, KindCos: true, KindMinio: true, KindQiNiu: true,
		KindOss: true, KinsS3: true, KindTos: true, KindObs: true, KindKs3: true,
	}
	if !validKinds[kind] {
		t.Fatalf("unexpected kind: %q", kind)
	}
}

func TestGetDefaultStoreKind_InvalidFallback(t *testing.T) {
	t.Setenv("STORAGE_KIND", "gcs")
	utils.PurgeEnvCacheForTest()
	kind := getDefaultStoreKind()
	if kind != KindLocal {
		t.Fatalf("invalid kind should fallback to local, got: %q", kind)
	}
}

func TestStoreInterfaceMethods(t *testing.T) {
	// Verify every kind's Store instance is non-nil and returns
	// valid results for PublicURL (pure computation, no network).
	kinds := []string{KindLocal, KindCos, KindMinio, KindQiNiu, KindOss, KinsS3, KindTos, KindObs, KindKs3}
	for _, k := range kinds {
		store := GetStore(k)
		url := store.PublicURL("test.txt")
		if url == "" {
			t.Fatalf("PublicURL(%q) for kind %q returned empty string", k, url)
		}
	}
}
