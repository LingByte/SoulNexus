package common

import "testing"

func TestNormalizePageParams_defaults(t *testing.T) {
	pp := NormalizePageParams(0, 0, 0)
	if pp.Page != 1 || pp.Size != DefaultPageSize || pp.Offset != 0 {
		t.Fatalf("got %+v", pp)
	}
}

func TestNormalizePageParams_clampsSize(t *testing.T) {
	pp := NormalizePageParams(2, 1000, 500)
	if pp.Page != 2 || pp.Size != 500 || pp.Offset != 500 {
		t.Fatalf("got %+v", pp)
	}
}

func TestTotalPages(t *testing.T) {
	if TotalPages(0, 20) != 0 {
		t.Fatal("expected 0 pages for empty total")
	}
	if TotalPages(21, 20) != 2 {
		t.Fatal("expected 2 pages")
	}
}
