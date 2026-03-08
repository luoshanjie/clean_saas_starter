package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

var moduleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type moduleSpec struct {
	Name             string
	EntityName       string
	PluralEntityName string
	SnakeName        string
	PluralName       string
	CommentName      string
}

type scaffoldFile struct {
	Path     string
	Template string
}

func runNewModuleCommand(args []string) error {
	fs := flag.NewFlagSet("new-module", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var name string
	var migrationPrefix string
	var withTest bool
	fs.StringVar(&name, "name", "", "module name, e.g. post")
	fs.StringVar(&migrationPrefix, "migration-prefix", time.Now().UTC().Format("20060102150405"), "migration filename prefix")
	fs.BoolVar(&withTest, "with-test", false, "generate a minimal usecase test skeleton")
	if err := fs.Parse(args); err != nil {
		return err
	}

	spec, err := parseModuleSpec(name)
	if err != nil {
		return err
	}

	files := buildModuleScaffold(spec, migrationPrefix, withTest)
	for _, file := range files {
		if err := writeScaffoldFile(file, spec); err != nil {
			return err
		}
		fmt.Printf("created %s\n", file.Path)
	}
	fmt.Printf("module scaffold ready: %s\n", spec.SnakeName)
	return nil
}

func parseModuleSpec(raw string) (moduleSpec, error) {
	name := strings.TrimSpace(strings.ToLower(raw))
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.Trim(name, "_")
	if !moduleNamePattern.MatchString(name) {
		return moduleSpec{}, fmt.Errorf("invalid module name %q: use snake_case starting with a letter", raw)
	}
	return moduleSpec{
		Name:             raw,
		EntityName:       toCamel(name),
		PluralEntityName: toCamel(pluralize(name)),
		SnakeName:        name,
		PluralName:       pluralize(name),
		CommentName:      strings.ReplaceAll(name, "_", " "),
	}, nil
}

func buildModuleScaffold(spec moduleSpec, migrationPrefix string, withTest bool) []scaffoldFile {
	files := []scaffoldFile{
		{
			Path:     filepath.Join("internal", "domain", "model", spec.SnakeName+".go"),
			Template: modelTemplate,
		},
		{
			Path:     filepath.Join("internal", "domain", "port", spec.SnakeName+".go"),
			Template: portTemplate,
		},
		{
			Path:     filepath.Join("internal", "app", "usecase", spec.SnakeName+".go"),
			Template: usecaseTemplate,
		},
		{
			Path:     filepath.Join("internal", "repo", "pg", spec.SnakeName+"_repo_pg.go"),
			Template: repoTemplate,
		},
		{
			Path:     filepath.Join("internal", "delivery", "http", "handler", spec.SnakeName+"_handler.go"),
			Template: handlerTemplate,
		},
		{
			Path:     filepath.Join("db", "query", spec.SnakeName+".sql"),
			Template: sqlTemplate,
		},
		{
			Path:     filepath.Join("migrations", migrationPrefix+"_add_"+spec.PluralName+".sql"),
			Template: migrationTemplate,
		},
	}
	if withTest {
		files = append(files, scaffoldFile{
			Path:     filepath.Join("internal", "app", "usecase", spec.SnakeName+"_test.go"),
			Template: usecaseTestTemplate,
		})
	}
	return files
}

func writeScaffoldFile(file scaffoldFile, spec moduleSpec) error {
	if _, err := os.Stat(file.Path); err == nil {
		return fmt.Errorf("refusing to overwrite existing file: %s", file.Path)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file.Path), 0o755); err != nil {
		return err
	}
	tpl, err := template.New(filepath.Base(file.Path)).Parse(file.Template)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(file.Path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return tpl.Execute(f, spec)
}

func toCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func pluralize(s string) string {
	switch {
	case strings.HasSuffix(s, "ch"), strings.HasSuffix(s, "sh"), strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "z"):
		return s + "es"
	case strings.HasSuffix(s, "y") && len(s) > 1 && !strings.ContainsAny(string(s[len(s)-2]), "aeiou"):
		return s[:len(s)-1] + "ies"
	default:
		return s + "s"
	}
}

const modelTemplate = `package model

import "time"

// {{.EntityName}} is the domain model for the {{.CommentName}} module.
type {{.EntityName}} struct {
	ID        string
	TenantID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
`

const portTemplate = `package port

import (
	"context"

	"service/internal/domain/model"
)

// {{.EntityName}}Repo defines repository contracts for the {{.CommentName}} module.
type {{.EntityName}}Repo interface {
	GetByID(ctx context.Context, id string) (*model.{{.EntityName}}, error)
	ListPage(ctx context.Context, tenantID string, limit, offset int) ([]*model.{{.EntityName}}, int, error)
	Create(ctx context.Context, item *model.{{.EntityName}}) error
}
`

const usecaseTemplate = `package usecase

import (
	"context"

	"service/internal/domain/model"
	"service/internal/domain/port"
)

type Create{{.EntityName}}Input struct {
	TenantID string
}

type Create{{.EntityName}}Usecase struct {
	Repo port.{{.EntityName}}Repo
}

func (u *Create{{.EntityName}}Usecase) Execute(ctx context.Context, in Create{{.EntityName}}Input) (*model.{{.EntityName}}, error) {
	// TODO: validate input and persist {{.SnakeName}}.
	_ = ctx
	_ = in
	return nil, nil
}

type List{{.PluralEntityName}}Input struct {
	TenantID string
	Page     int
	PageSize int
}

type List{{.PluralEntityName}}Output struct {
	Items []*model.{{.EntityName}}
	Total int
}

type List{{.PluralEntityName}}Usecase struct {
	Repo port.{{.EntityName}}Repo
}

func (u *List{{.PluralEntityName}}Usecase) Execute(ctx context.Context, in List{{.PluralEntityName}}Input) (*List{{.PluralEntityName}}Output, error) {
	// TODO: add filter normalization and call repo.
	_ = ctx
	_ = in
	return &List{{.PluralEntityName}}Output{}, nil
}
`

const repoTemplate = `package pg

import (
	"context"
	"errors"

	"service/internal/domain/model"
	"service/internal/domain/port"
)

// {{.EntityName}}RepoPG is the PostgreSQL adapter skeleton for the {{.CommentName}} module.
type {{.EntityName}}RepoPG struct{}

var _ port.{{.EntityName}}Repo = (*{{.EntityName}}RepoPG)(nil)

func (r *{{.EntityName}}RepoPG) GetByID(ctx context.Context, id string) (*model.{{.EntityName}}, error) {
	// TODO: implement sqlc query lookup.
	_ = ctx
	_ = id
	return nil, errors.New("not implemented")
}

func (r *{{.EntityName}}RepoPG) ListPage(ctx context.Context, tenantID string, limit, offset int) ([]*model.{{.EntityName}}, int, error) {
	// TODO: implement sqlc page query.
	_ = ctx
	_ = tenantID
	_ = limit
	_ = offset
	return nil, 0, errors.New("not implemented")
}

func (r *{{.EntityName}}RepoPG) Create(ctx context.Context, item *model.{{.EntityName}}) error {
	// TODO: implement sqlc insert query.
	_ = ctx
	_ = item
	return errors.New("not implemented")
}
`

const handlerTemplate = `package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"service/internal/app/usecase"
)

// {{.EntityName}}Handler is the HTTP delivery skeleton for the {{.CommentName}} module.
type {{.EntityName}}Handler struct {
	CreateUC *usecase.Create{{.EntityName}}Usecase
	ListUC   *usecase.List{{.PluralEntityName}}Usecase
}

func (h *{{.EntityName}}Handler) Create(c echo.Context) error {
	// TODO: bind request, call CreateUC, map response.
	_ = h
	return c.JSON(http.StatusNotImplemented, map[string]any{"message": "{{.SnakeName}} create not implemented"})
}

func (h *{{.EntityName}}Handler) List(c echo.Context) error {
	// TODO: bind request, call ListUC, map response.
	_ = h
	return c.JSON(http.StatusNotImplemented, map[string]any{"message": "{{.SnakeName}} list not implemented"})
}
`

const sqlTemplate = `-- {{.SnakeName}} module queries

-- name: Get{{.EntityName}}ByID :one
-- SELECT * FROM {{.PluralName}} WHERE id = $1 LIMIT 1;

-- name: List{{.PluralEntityName}} :many
-- SELECT * FROM {{.PluralName}}
-- WHERE tenant_id = $1
-- ORDER BY created_at DESC
-- LIMIT $2 OFFSET $3;

-- name: Count{{.PluralEntityName}} :one
-- SELECT count(*) FROM {{.PluralName}} WHERE tenant_id = $1;

-- name: Create{{.EntityName}} :exec
-- INSERT INTO {{.PluralName}} (
--   id,
--   tenant_id,
--   created_at,
--   updated_at
-- ) VALUES ($1, $2, $3, $4);
`

const migrationTemplate = `-- Add {{.PluralName}} table(s)
CREATE TABLE IF NOT EXISTS {{.PluralName}} (
  id uuid PRIMARY KEY,
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_{{.PluralName}}_tenant_id ON {{.PluralName}} (tenant_id);
`

const usecaseTestTemplate = `package usecase

import (
	"context"
	"testing"
)

func TestList{{.PluralEntityName}}Usecase_Execute(t *testing.T) {
	u := &List{{.PluralEntityName}}Usecase{}
	out, err := u.Execute(context.Background(), List{{.PluralEntityName}}Input{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatalf("expected non-nil output")
	}
}
`
