package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

type projectSpec struct {
	Name        string
	DisplayName string
	Slug        string
	ModulePath  string
	OutputDir   string
	CmdName     string
	BinaryName  string
	DBName      string
}

type sourceProjectMeta struct {
	ModulePath  string
	CmdName     string
	BinaryName  string
	ProjectName string
	DBName      string
}

func runNewProjectCommand(args []string) error {
	fs := flag.NewFlagSet("new-project", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var name string
	var output string
	var modulePath string
	var initGit bool
	fs.StringVar(&name, "name", "", "project name, e.g. my-saas")
	fs.StringVar(&output, "output", "", "output directory, defaults to ../<project_name>")
	fs.StringVar(&modulePath, "module-path", "", "go module path, defaults to project name")
	fs.BoolVar(&initGit, "init-git", false, "initialize a git repository in the generated project")
	if err := fs.Parse(args); err != nil {
		return err
	}

	spec, err := parseProjectSpec(name, output, modulePath)
	if err != nil {
		return err
	}

	srcRoot, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := scaffoldProject(srcRoot, spec); err != nil {
		return err
	}
	if initGit {
		if err := initProjectGit(spec.OutputDir); err != nil {
			return err
		}
	}
	printProjectNextSteps(os.Stdout, spec, initGit)
	return nil
}

func parseProjectSpec(name, output, modulePath string) (projectSpec, error) {
	rawName := strings.TrimSpace(name)
	rawOutput := strings.TrimSpace(output)
	rawModulePath := strings.TrimSpace(modulePath)
	if rawName == "" {
		return projectSpec{}, errors.New("project name is required")
	}
	slug := slugifyProjectName(rawName)
	if !projectNamePattern.MatchString(slug) {
		return projectSpec{}, fmt.Errorf("invalid project name %q", rawName)
	}
	if rawModulePath == "" {
		rawModulePath = slug
	}
	if rawOutput == "" {
		rawOutput = filepath.Join("..", slug)
	}
	return projectSpec{
		Name:        rawName,
		DisplayName: displayProjectName(rawName),
		Slug:        slug,
		ModulePath:  rawModulePath,
		OutputDir:   filepath.Clean(rawOutput),
		CmdName:     slug,
		BinaryName:  slug,
		DBName:      strings.ReplaceAll(slug, "-", "_"),
	}, nil
}

func scaffoldProject(srcRoot string, spec projectSpec) error {
	absSrc, err := filepath.Abs(srcRoot)
	if err != nil {
		return err
	}
	absOut, err := filepath.Abs(spec.OutputDir)
	if err != nil {
		return err
	}
	if absOut == absSrc || strings.HasPrefix(absOut, absSrc+string(os.PathSeparator)) {
		return errors.New("output directory must be outside the current repository")
	}
	if err := ensureEmptyDir(absOut); err != nil {
		return err
	}
	sourceMeta, err := detectSourceProjectMeta(absSrc)
	if err != nil {
		return err
	}

	return filepath.WalkDir(absSrc, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(absSrc, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipProjectPath(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetRel := mapProjectPath(rel, sourceMeta, spec)
		targetPath := filepath.Join(absOut, targetRel)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out := applyProjectReplacements(raw, sourceMeta, spec)
		return os.WriteFile(targetPath, out, 0o644)
	})
}

func ensureEmptyDir(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0o755)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("output path is not a directory: %s", path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("output directory is not empty: %s", path)
	}
	return nil
}

func shouldSkipProjectPath(rel string, isDir bool) bool {
	base := filepath.Base(rel)
	if base == ".git" || base == "build" || base == "logs" || base == ".gocache" {
		return true
	}
	normalized := filepath.ToSlash(rel)
	if normalized == "migrations/sqlite" || strings.HasPrefix(normalized, "migrations/sqlite/") {
		return true
	}
	if normalized == "internal/repo/sqlite" || strings.HasPrefix(normalized, "internal/repo/sqlite/") {
		return true
	}
	if normalized == "docs/sqlite-support-plan.md" {
		return true
	}
	if normalized == "docs/superpowers" || strings.HasPrefix(normalized, "docs/superpowers/") {
		return true
	}
	if normalized == "cmd/cli/new_project_test.go" {
		return true
	}
	if strings.HasSuffix(base, ".DS_Store") {
		return true
	}
	if isDir {
		return false
	}
	switch rel {
	case ".env", "app.yaml":
		return true
	default:
		return false
	}
}

func mapProjectPath(rel string, source sourceProjectMeta, spec projectSpec) string {
	normalized := filepath.ToSlash(rel)
	if normalized == "cmd/"+source.CmdName+"/main.go" {
		return filepath.Join("cmd", spec.CmdName, "main.go")
	}
	return rel
}

func applyProjectReplacements(raw []byte, source sourceProjectMeta, spec projectSpec) []byte {
	replacer := strings.NewReplacer(
		"# Clean SaaS Starter", "# "+spec.DisplayName,
		"An open-source multi-tenant SaaS starter for building new systems on top of a reusable kernel instead of starting from a vertical business project.", spec.DisplayName+" backend service built on a reusable SaaS kernel.",
		"This repository is a generic SaaS scaffold, not a concrete business application.", spec.DisplayName+" backend service generated from Clean SaaS Starter.",
		"一个开源的多租户 SaaS 脚手架，用于在可复用的内核之上构建新系统，而不是从某个具体业务项目开始演化。", spec.DisplayName+" 后端服务，基于可复用 SaaS 内核构建。",
		"本仓库是一个通用 SaaS 脚手架，不是某个垂直业务应用。", spec.DisplayName+" 后端服务，由 Clean SaaS Starter 生成。",
		"module "+source.ModulePath, "module "+spec.ModulePath,
		"\""+source.ModulePath+"/", "\""+spec.ModulePath+"/",
		"cmd/"+source.CmdName, "cmd/"+spec.CmdName,
		"APP_NAME := "+source.BinaryName, "APP_NAME := "+spec.BinaryName,
		"PROJECT := "+source.ProjectName, "PROJECT := "+spec.BinaryName,
		"/out/"+source.BinaryName, "/out/"+spec.BinaryName,
		"/app/"+source.BinaryName, "/app/"+spec.BinaryName,
		"/opt/"+source.BinaryName+"/"+source.BinaryName, "/opt/"+spec.BinaryName+"/"+spec.BinaryName,
		"/opt/"+source.BinaryName, "/opt/"+spec.BinaryName,
		"/etc/"+source.BinaryName, "/etc/"+spec.BinaryName,
		"/var/log/"+source.BinaryName, "/var/log/"+spec.BinaryName,
		source.BinaryName+".env", spec.BinaryName+".env",
		"${IMAGE:-"+source.BinaryName+":latest}", "${IMAGE:-"+spec.BinaryName+":latest}",
		"container_"+source.BinaryName, "container_"+spec.BinaryName,
		"  "+source.BinaryName+":\n", "  "+spec.BinaryName+":\n",
		source.BinaryName+"_cli_logs", spec.BinaryName+"_cli_logs",
		source.DBName+"-dev", spec.DBName+"-dev",
		"CREATE DATABASE "+source.DBName+"_dev", "CREATE DATABASE "+spec.DBName+"_dev",
		"datname = '"+source.DBName+"_dev'", "datname = '"+spec.DBName+"_dev'",
		"CREATE DATABASE "+source.DBName+"_release", "CREATE DATABASE "+spec.DBName+"_release",
		"datname = '"+source.DBName+"_release'", "datname = '"+spec.DBName+"_release'",
		":5432/"+source.DBName+"?", ":5432/"+spec.DBName+"?",
		"Service API", spec.DisplayName+" API",
		"Service API (development docs).", spec.DisplayName+" API (development docs).",
	)
	return []byte(replacer.Replace(string(raw)))
}

func slugifyProjectName(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	s = strings.ReplaceAll(s, " ", "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func displayProjectName(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	parts := strings.Fields(s)
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

func containsText(data []byte, text string) bool {
	return bytes.Contains(data, []byte(text))
}

func initProjectGit(outputDir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func printProjectNextSteps(w io.Writer, spec projectSpec, gitInitialized bool) {
	fmt.Fprintf(w, "project scaffold ready: %s\n", spec.OutputDir)
	if gitInitialized {
		fmt.Fprintln(w, "git repository initialized")
	}
	fmt.Fprintln(w, "next steps:")
	fmt.Fprintf(w, "  cd %s\n", spec.OutputDir)
	fmt.Fprintln(w, "  cp .env.example .env")
	fmt.Fprintln(w, "  # edit .env and app.yaml.example for database, JWT secret, and storage settings")
	fmt.Fprintln(w, "  # execute SQL in migrations/pgsql/ for PostgreSQL-compatible databases")
	fmt.Fprintln(w, "  make build")
	fmt.Fprintln(w, "  make dev")
	fmt.Fprintln(w, "  # optional deployment bundle for x86_64 Linux servers")
	fmt.Fprintln(w, "  make package-linux-amd64")
}

func detectSourceProjectMeta(srcRoot string) (sourceProjectMeta, error) {
	moduleRaw, err := os.ReadFile(filepath.Join(srcRoot, "go.mod"))
	if err != nil {
		return sourceProjectMeta{}, err
	}
	modulePath := ""
	for _, line := range strings.Split(string(moduleRaw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			modulePath = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if modulePath == "" {
		return sourceProjectMeta{}, errors.New("failed to detect source module path")
	}

	makefileRaw, err := os.ReadFile(filepath.Join(srcRoot, "Makefile"))
	if err != nil {
		return sourceProjectMeta{}, err
	}
	binaryName := findMakeVar(string(makefileRaw), "APP_NAME")
	projectName := findMakeVar(string(makefileRaw), "PROJECT")
	if binaryName == "" || projectName == "" {
		return sourceProjectMeta{}, errors.New("failed to detect source make variables")
	}

	cmdEntries, err := os.ReadDir(filepath.Join(srcRoot, "cmd"))
	if err != nil {
		return sourceProjectMeta{}, err
	}
	cmdName := ""
	for _, entry := range cmdEntries {
		if !entry.IsDir() || entry.Name() == "cli" {
			continue
		}
		if _, err := os.Stat(filepath.Join(srcRoot, "cmd", entry.Name(), "main.go")); err == nil {
			if cmdName != "" {
				return sourceProjectMeta{}, errors.New("expected exactly one non-cli command entrypoint")
			}
			cmdName = entry.Name()
		}
	}
	if cmdName == "" {
		return sourceProjectMeta{}, errors.New("failed to detect source command entrypoint")
	}

	dbName := binaryName
	envRaw, err := os.ReadFile(filepath.Join(srcRoot, ".env.example"))
	if err == nil {
		for _, line := range strings.Split(string(envRaw), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "DB_DSN=") {
				continue
			}
			dsn := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "DB_DSN="))
			if idx := strings.Index(dsn, ":5432/"); idx >= 0 {
				rest := dsn[idx+len(":5432/"):]
				if q := strings.Index(rest, "?"); q >= 0 {
					dbName = rest[:q]
				}
			}
		}
	}

	return sourceProjectMeta{
		ModulePath:  modulePath,
		CmdName:     cmdName,
		BinaryName:  binaryName,
		ProjectName: projectName,
		DBName:      dbName,
	}, nil
}

func findMakeVar(content, key string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, key+" :=") {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, key+" :="))
	}
	return ""
}
