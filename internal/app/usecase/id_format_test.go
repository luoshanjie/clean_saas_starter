package usecase

import "testing"

func TestCanonicalUUIDText(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{
			in:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			want: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			ok:   true,
		},
		{
			in:   "AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA",
			want: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			ok:   true,
		},
		{
			in:   "aaaaaaaa-aaaaaaaa-aaaa-aaaa-aaaaaaaa",
			want: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			ok:   true,
		},
		{
			in:   "b1",
			want: "b1",
			ok:   true,
		},
	}
	for _, tt := range tests {
		got, err := canonicalUUIDText(tt.in)
		if tt.ok {
			if err != nil {
				t.Fatalf("canonicalUUIDText(%q) unexpected err: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("canonicalUUIDText(%q)=%q, want %q", tt.in, got, tt.want)
			}
			continue
		}
		if err == nil {
			t.Fatalf("canonicalUUIDText(%q) expected error, got %q", tt.in, got)
		}
	}
}
