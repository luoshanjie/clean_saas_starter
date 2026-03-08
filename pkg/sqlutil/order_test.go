package sqlutil

import "testing"

func TestOrderBy_EmptyField(t *testing.T) {
	clause, err := OrderBy(map[string]string{"created_at": "created_at"}, "", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if clause != "" {
		t.Fatalf("expected empty clause, got: %q", clause)
	}
}

func TestOrderBy_AllowedField(t *testing.T) {
	clause, err := OrderBy(map[string]string{"created_at": "created_at"}, "created_at", "desc")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if clause != " ORDER BY created_at DESC" {
		t.Fatalf("unexpected clause: %q", clause)
	}
}

func TestOrderBy_InvalidField(t *testing.T) {
	_, err := OrderBy(map[string]string{"created_at": "created_at"}, "bad", "asc")
	if err != ErrInvalidOrderBy {
		t.Fatalf("expected ErrInvalidOrderBy, got: %v", err)
	}
}

func TestOrderBy_InvalidOrder(t *testing.T) {
	_, err := OrderBy(map[string]string{"created_at": "created_at"}, "created_at", "drop table")
	if err != ErrInvalidOrder {
		t.Fatalf("expected ErrInvalidOrder, got: %v", err)
	}
}
