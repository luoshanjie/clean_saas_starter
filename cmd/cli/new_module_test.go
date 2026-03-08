package main

import "testing"

func TestParseModuleSpec(t *testing.T) {
	spec, err := parseModuleSpec("ai_workflow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.EntityName != "AiWorkflow" {
		t.Fatalf("unexpected entity name: %s", spec.EntityName)
	}
	if spec.PluralEntityName != "AiWorkflows" {
		t.Fatalf("unexpected plural entity name: %s", spec.PluralEntityName)
	}
	if spec.PluralName != "ai_workflows" {
		t.Fatalf("unexpected plural name: %s", spec.PluralName)
	}
}

func TestParseModuleSpec_Invalid(t *testing.T) {
	if _, err := parseModuleSpec("123bad"); err == nil {
		t.Fatalf("expected invalid module name error")
	}
}

func TestBuildModuleScaffold(t *testing.T) {
	spec := moduleSpec{
		EntityName:       "Post",
		PluralEntityName: "Posts",
		SnakeName:        "post",
		PluralName:       "posts",
	}
	files := buildModuleScaffold(spec, "20260308000000", false)
	if len(files) != 7 {
		t.Fatalf("unexpected scaffold file count: %d", len(files))
	}
	if files[0].Path != "internal/domain/model/post.go" {
		t.Fatalf("unexpected first scaffold path: %s", files[0].Path)
	}
	if files[6].Path != "migrations/20260308000000_add_posts.sql" {
		t.Fatalf("unexpected migration path: %s", files[6].Path)
	}
}

func TestBuildModuleScaffold_WithTest(t *testing.T) {
	spec := moduleSpec{
		EntityName:       "Post",
		PluralEntityName: "Posts",
		SnakeName:        "post",
		PluralName:       "posts",
	}
	files := buildModuleScaffold(spec, "20260308000000", true)
	if len(files) != 8 {
		t.Fatalf("unexpected scaffold file count with test: %d", len(files))
	}
	if files[7].Path != "internal/app/usecase/post_test.go" {
		t.Fatalf("unexpected test scaffold path: %s", files[7].Path)
	}
}
