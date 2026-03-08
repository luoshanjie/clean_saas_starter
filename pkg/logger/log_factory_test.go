package logger

import (
	"reflect"
	"testing"
)

func TestLogFactory_Cache(t *testing.T) {
	f := NewLogFactory(Config{Dir: t.TempDir(), Level: "info", Format: "json", EnableConsole: false})
	l1 := f.Get("app")
	l2 := f.Get("app")

	p1 := reflect.ValueOf(l1).Pointer()
	p2 := reflect.ValueOf(l2).Pointer()
	if p1 != p2 {
		t.Fatalf("expected same logger instance")
	}
}
