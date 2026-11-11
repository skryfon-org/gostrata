package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/skryfon-org/goclean/pkg/strutil"
	"github.com/skryfon-org/goclean/pkg/ui"
	"github.com/skryfon-org/goclean/templates"
)

// Supported database engines and HTTP routers for generated projects.
const (
	DBPostgres = "postgres"
	DBMySQL    = "mysql"
	DBSQLite   = "sqlite"

	RouterGin = "gin"
	RouterChi = "chi"

	ORMGorm = "gorm"
)

var (
	// SupportedDatabases lists valid values for ProjectOptions.Database.
	SupportedDatabases = []string{DBPostgres, DBMySQL, DBSQLite}
	// SupportedRouters lists valid values for ProjectOptions.Router.
	SupportedRouters = []string{RouterGin, RouterChi}
	// SupportedORMs lists valid values for ProjectOptions.ORM.
	SupportedORMs = []string{ORMGorm}
)

// ProjectConfig is persisted as .goclean.json so later `make:*` commands match the project's stack.
type ProjectConfig struct {
	Database string `json:"database"`
	Router   string `json:"router"`
	ORM      string `json:"orm"`
}

const projectConfigFile = ".goclean.json"

type ProjectOptions struct {
	ProjectName string
	ModuleName  string
	DBName      string
	TargetDir   string
	Database    string // postgres (default), mysql, sqlite
	Router      string // gin (default), chi
	ORM         string // gorm (default)
	NoGit       bool   // skip `git init`
	NoTidy      bool   // skip `go mod tidy`
	Force       bool   // allow generating into a non-empty directory (overwrites files)
	DryRun      bool   // print the files that would be created and exit
}

// DBDefaultPort is the engine's standard port (used by config, .env and docker-compose templates).
func (o ProjectOptions) DBDefaultPort() string {
	if o.Database == DBMySQL {
		return "3306"
	}
	return "5432"
}

// DBDefaultUser is the default superuser for the engine.
func (o ProjectOptions) DBDefaultUser() string {
	if o.Database == DBMySQL {
		return "root"
	}
	return "postgres"
}

// ImageName is ProjectName normalised into a valid Docker image reference
// (lowercase; anything outside [a-z0-9._-] becomes '-').
func (o ProjectOptions) ImageName() string {
	var b strings.Builder
	for _, r := range strings.ToLower(o.ProjectName) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-._")
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// GenerateProject scaffolds a new Go Clean Architecture project from embedded templates.
func GenerateProject(opts ProjectOptions) (err error) {
	if opts.DBName == "" {
		opts.DBName = strutil.ToSnakeCase(opts.ProjectName) + "_db"
	}
	if opts.ModuleName == "" {
		opts.ModuleName = opts.ProjectName
	}
	if opts.Database == "" {
		opts.Database = DBPostgres
	}
	if opts.Router == "" {
		opts.Router = RouterGin
	}
	if opts.ORM == "" {
		opts.ORM = ORMGorm
	}
	if !contains(SupportedORMs, opts.ORM) {
		return fmt.Errorf("unsupported orm %q (supported: %s)", opts.ORM, strings.Join(SupportedORMs, ", "))
	}
	if !contains(SupportedDatabases, opts.Database) {
		return fmt.Errorf("unsupported database %q (supported: %s)", opts.Database, strings.Join(SupportedDatabases, ", "))
	}
	if !contains(SupportedRouters, opts.Router) {
		return fmt.Errorf("unsupported router %q (supported: %s)", opts.Router, strings.Join(SupportedRouters, ", "))
	}

	// Check if target directory exists and is non-empty
	if stat, err := os.Stat(opts.TargetDir); err == nil && stat.IsDir() {
		entries, err := os.ReadDir(opts.TargetDir)
		if err == nil && len(entries) > 0 && !opts.Force {
			return fmt.Errorf("directory %s already exists and is not empty (use --force to generate into it)", opts.TargetDir)
		}
	}

	// Track whether we created the target dir so a failed run can be rolled back.
	_, statErr := os.Stat(opts.TargetDir)
	createdTarget := os.IsNotExist(statErr)
	defer func() {
		if err != nil && createdTarget {
			_ = os.RemoveAll(opts.TargetDir)
		}
	}()

	ui.Step(1, 4, "Scaffolding Clean Architecture files")

	err = fs.WalkDir(templates.FS, "project", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel("project", path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// Destination path (strip .tmpl suffix if present)
		cleanRelPath := strings.TrimSuffix(relPath, ".tmpl")
		destPath := filepath.Join(opts.TargetDir, cleanRelPath)

		if d.IsDir() {
			if opts.DryRun {
				return nil
			}
			return os.MkdirAll(destPath, 0755)
		}

		content, err := templates.FS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		// Render template
		tmpl, err := template.New(relPath).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, opts); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", path, err)
		}

		// A template that renders to nothing (e.g. docker-compose for SQLite) is skipped.
		if len(bytes.TrimSpace(buf.Bytes())) == 0 {
			return nil
		}

		if opts.DryRun {
			ui.Info("Would create: %s", destPath)
			return nil
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		out, err := formatGo(destPath, buf.Bytes())
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, out, 0644)
	})

	if err != nil {
		return err
	}

	if opts.DryRun {
		ui.Info("Dry run: no files were changed.")
		return nil
	}

	cfgJSON, _ := json.MarshalIndent(ProjectConfig{Database: opts.Database, Router: opts.Router, ORM: opts.ORM}, "", "  ")
	if err := os.WriteFile(filepath.Join(opts.TargetDir, projectConfigFile), append(cfgJSON, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", projectConfigFile, err)
	}

	// Create .env from .env.example
	ui.Step(2, 4, "Configuring environment (.env)")
	envExamplePath := filepath.Join(opts.TargetDir, ".env.example")
	envPath := filepath.Join(opts.TargetDir, ".env")
	envBytes, err := os.ReadFile(envExamplePath)
	if err != nil {
		return fmt.Errorf("failed to read .env.example: %w", err)
	}
	if err := os.WriteFile(envPath, envBytes, 0600); err != nil {
		return fmt.Errorf("failed to write .env: %w", err)
	}

	// Initialize git repository
	ui.Step(3, 4, "Initializing Git repository")
	if opts.NoGit {
		ui.Info("git init skipped (--no-git)")
	} else {
		gitCmd := exec.Command("git", "init")
		gitCmd.Dir = opts.TargetDir
		if out, gitErr := gitCmd.CombinedOutput(); gitErr != nil {
			ui.Warn("git init skipped (%v): %s", gitErr, strings.TrimSpace(string(out)))
		}
	}

	// Run go mod tidy
	ui.Step(4, 4, "Downloading and tidying dependencies (go mod tidy)")
	if opts.NoTidy {
		ui.Info("go mod tidy skipped (--no-tidy); run it before building")
		return nil
	}
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = opts.TargetDir
	if tidyOutput, tidyErr := tidyCmd.CombinedOutput(); tidyErr != nil {
		ui.Warn("go mod tidy warning (you can run 'go mod tidy' later manually): %s", string(tidyOutput))
	}

	return nil
}
