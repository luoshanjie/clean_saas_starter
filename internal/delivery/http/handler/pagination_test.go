package handler

import "testing"

func TestPaginateSlice_DefaultPage(t *testing.T) {
	items := []int{1, 2, 3}
	paged, page, pageSize, total := paginateSlice(items, 0, 0)
	if page != 1 || pageSize != 20 || total != 3 {
		t.Fatalf("unexpected meta: page=%d size=%d total=%d", page, pageSize, total)
	}
	if len(paged) != 3 {
		t.Fatalf("expected all items in first page, got %d", len(paged))
	}
}

func TestPaginateSlice_SpecificPage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	paged, page, pageSize, total := paginateSlice(items, 2, 2)
	if page != 2 || pageSize != 2 || total != 5 {
		t.Fatalf("unexpected meta: page=%d size=%d total=%d", page, pageSize, total)
	}
	if len(paged) != 2 || paged[0] != 3 || paged[1] != 4 {
		t.Fatalf("unexpected page items: %+v", paged)
	}
}

func TestPaginateSlice_OutOfRange(t *testing.T) {
	items := []int{1, 2, 3}
	paged, page, pageSize, total := paginateSlice(items, 3, 2)
	if page != 3 || pageSize != 2 || total != 3 {
		t.Fatalf("unexpected meta: page=%d size=%d total=%d", page, pageSize, total)
	}
	if len(paged) != 0 {
		t.Fatalf("expected empty page, got %+v", paged)
	}
}
