package generator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/skryfon-org/goclean/pkg/strutil"
	"github.com/skryfon-org/goclean/pkg/ui"
	"github.com/skryfon-org/goclean/templates"
)

type StubData struct {
	ModuleName         string
	EntityPascal       string // e.g. "Product"
	EntityCamel        string // e.g. "product"
	EntitySnake        string // e.g. "product"
	EntityKebab        string // e.g. "product"
	EntityPluralSnake  string // e.g. "products"
	EntityPluralPascal string // e.g. "Products"
	EntityPluralCamel  string // e.g. "products"
	EntityPluralKebab  string // e.g. "products" (used for URL paths)
	Database           string // postgres | mysql | sqlite (from .goclean.json)
	Router             string // gin | chi (from .goclean.json)
	Fields             []Field
}

// Options controls how files are generated.
type Options struct {
	Fields     []Field
	Force      bool // overwrite existing files
	DryRun     bool // print what would happen without touching the disk
	NoRegister bool // make:module: don't patch container.go / routes.go
}

// FindGoModuleName locates the nearest go.mod and extracts the module path.
func FindGoModuleName(startDir string) (string, string, error) {
	curr := startDir
	for {
		goModPath := filepath.Join(curr, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			file, err := os.Open(goModPath)
			if err != nil {
				return "", "", err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "module ") {
					modName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
					return modName, curr, nil
				}
			}
			return "", "", fmt.Errorf("module line not found in %s", goModPath)
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return "", "", fmt.Errorf("go.mod not found in %s or any parent directory", startDir)
}

// BuildStubData prepares naming variations for code templates.
func BuildStubData(moduleName, rawName string) StubData {
	pascal := strutil.ToPascalCase(rawName)
	camel := strutil.ToCamelCase(rawName)
	snake := strutil.ToSnakeCase(rawName)
	kebab := strutil.ToKebabCase(rawName)
	pluralSnake := strutil.Pluralize(snake)
	pluralPascal := strutil.Pluralize(pascal)

	return StubData{
		ModuleName:         moduleName,
		EntityPascal:       pascal,
		EntityCamel:        camel,
		EntitySnake:        snake,
		EntityKebab:        kebab,
		EntityPluralSnake:  pluralSnake,
		EntityPluralPascal: pluralPascal,
		EntityPluralCamel:  strutil.ToCamelCase(pluralSnake),
		EntityPluralKebab:  strutil.ToKebabCase(pluralSnake),
	}
}

// runner carries options and records what happened for the final summary.
type runner struct {
	opts        Options
	created     []string
	overwritten []string
	skipped     []string
}

// write creates or overwrites a file, honouring Force and DryRun.
func (r *runner) write(path string, content []byte) error {
	_, statErr := os.Stat(path)
	exists := statErr == nil
	if exists && !r.opts.Force {
		r.skipped = append(r.skipped, path)
		ui.Warn("Skipped existing file: %s", path)
		return nil
	}
	verb, would, list := "Created", "create", &r.created
	if exists {
		verb, would, list = "Overwrote", "overwrite", &r.overwritten
	}
	*list = append(*list, path)
	if r.opts.DryRun {
		ui.Info("Would %s: %s", would, path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	content, err := formatGo(path, content)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	ui.Success("%s: %s", verb, path)
	return nil
}

// render renders a stub from templates/stubs and writes it to targetPath.
func (r *runner) render(stubName, targetPath string, data StubData) error {
	content, err := templates.FS.ReadFile("stubs/" + stubName)
	if err != nil {
		return fmt.Errorf("failed to read stub %s: %w", stubName, err)
	}
	tmpl, err := template.New(stubName).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse stub %s: %w", stubName, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute stub %s: %w", stubName, err)
	}
	return r.write(targetPath, buf.Bytes())
}

func (r *runner) summary() {
	fmt.Println()
	ui.Info("%d created, %d overwritten, %d skipped", len(r.created), len(r.overwritten), len(r.skipped))
	if len(r.skipped) > 0 {
		ui.Warn("Skipped files already exist; re-run with --force to overwrite them.")
	}
	if r.opts.DryRun {
		ui.Info("Dry run: no files were changed.")
	}
}

type stubFile struct{ stub, dest string }

// layerFiles maps each layer name to its (stub, destination) pairs.
func layerFiles(projectRoot string, d StubData) map[string][]stubFile {
	j := filepath.Join
	return map[string][]stubFile{
		"entity": {{"entity.tmpl", j(projectRoot, "internal", "domain", "entity", d.EntitySnake+".go")}},
		"types":  {{"types.tmpl", j(projectRoot, "internal", "types", d.EntitySnake+".go")}},
		"repo": {
			{"repository.tmpl", j(projectRoot, "internal", "domain", "repository", d.EntitySnake+"_repository.go")},
			{"persistence.tmpl", j(projectRoot, "internal", "infrastructure", "persistence", d.EntitySnake+"_repository.go")},
		},
		"usecase": {
			{"usecase_interface.tmpl", j(projectRoot, "internal", "usecase", "interface", d.EntitySnake+"_usecase.go")},
			{"usecase_implementation.tmpl", j(projectRoot, "internal", "usecase", "implementation", d.EntitySnake+"_usecase.go")},
		},
		"handler": {{handlerStub(d), j(projectRoot, "internal", "delivery", "http", "handlers", d.EntitySnake+"_handler.go")}},
	}
}

func handlerStub(d StubData) string {
	if d.Router == RouterChi {
		return "handler_chi.tmpl"
	}
	return "handler.tmpl"
}

// moduleOrder is the order in which layers are generated by GenerateModule.
var moduleOrder = []string{"entity", "types", "repo", "usecase", "handler"}

// loadProjectConfig reads .goclean.json; projects created before it existed are postgres + gin + gorm.
func loadProjectConfig(projectRoot string) ProjectConfig {
	cfg := ProjectConfig{Database: DBPostgres, Router: RouterGin, ORM: ORMGorm}
	if b, err := os.ReadFile(filepath.Join(projectRoot, projectConfigFile)); err == nil {
		var parsed ProjectConfig
		if json.Unmarshal(b, &parsed) == nil {
			if contains(SupportedDatabases, parsed.Database) {
				cfg.Database = parsed.Database
			}
			if contains(SupportedRouters, parsed.Router) {
				cfg.Router = parsed.Router
			}
			if contains(SupportedORMs, parsed.ORM) {
				cfg.ORM = parsed.ORM
			}
		}
	}
	return cfg
}

func prepare(rootPath, rawName string, opts Options) (StubData, string, error) {
	if !strutil.IsValidIdentifier(rawName) {
		return StubData{}, "", fmt.Errorf("invalid name %q: must be a valid Go identifier (letters, digits, '_' or '-', not starting with a digit)", rawName)
	}
	if isGoKeyword(strutil.ToCamelCase(rawName)) {
		return StubData{}, "", fmt.Errorf("invalid name %q: %q is a Go keyword", rawName, strutil.ToCamelCase(rawName))
	}
	moduleName, projectRoot, err := FindGoModuleName(rootPath)
	if err != nil {
		return StubData{}, "", err
	}
	data := BuildStubData(moduleName, rawName)
	cfg := loadProjectConfig(projectRoot)
	data.Database, data.Router = cfg.Database, cfg.Router
	data.Fields = make([]Field, len(opts.Fields))
	for i, f := range opts.Fields {
		f.SQLType = sqlTypeFor(cfg.Database, f.Type)
		data.Fields[i] = f
	}
	return data, projectRoot, nil
}

func (r *runner) renderLayer(projectRoot, layer string, data StubData) error {
	for _, f := range layerFiles(projectRoot, data)[layer] {
		if err := r.render(f.stub, f.dest, data); err != nil {
			return err
		}
	}
	return nil
}

// nextMigrationNumber returns the next sequential migration number (max existing numeric prefix + 1).
func nextMigrationNumber(dir string) int {
	max := 0
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		prefix, _, _ := strings.Cut(e.Name(), "_")
		if n, err := strconv.Atoi(prefix); err == nil && n > max {
			max = n
		}
	}
	return max + 1
}

// renderMigration writes sequentially numbered up/down migration files.
func (r *runner) renderMigration(projectRoot string, data StubData) error {
	dir := filepath.Join(projectRoot, "migrations")
	base := filepath.Join(dir, fmt.Sprintf("%06d_create_%s_table", nextMigrationNumber(dir), data.EntityPluralSnake))
	if err := r.render("migration.tmpl", base+".up.sql", data); err != nil {
		return err
	}
	return r.write(base+".down.sql", []byte(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", data.EntityPluralSnake)))
}

// register patches container.go and routes.go. It returns true when both were handled
// (patched or already registered); false means the caller should print manual steps.
func (r *runner) register(projectRoot string, d StubData) bool {
	container := filepath.Join(projectRoot, "internal", "infrastructure", "container", "container.go")
	routes := filepath.Join(projectRoot, "internal", "delivery", "http", "routes.go")

	cPatches := []patch{
		{markerHandlerFields, fmt.Sprintf("%sHandler *handlers.%sHandler", d.EntityPascal, d.EntityPascal)},
		{markerRepositories, fmt.Sprintf("%sRepo := persistence.New%sRepository(db)", d.EntityCamel, d.EntityPascal)},
		{markerUsecases, fmt.Sprintf("%sUsecase := usecaseimpl.New%sUsecase(%sRepo)", d.EntityCamel, d.EntityPascal, d.EntityCamel)},
		{markerHandlers, fmt.Sprintf("%sHandler := handlers.New%sHandler(%sUsecase)", d.EntityCamel, d.EntityPascal, d.EntityCamel)},
		{markerHandlerInit, fmt.Sprintf("%sHandler: %sHandler,", d.EntityPascal, d.EntityCamel)},
	}
	v, p, k := d.EntityPluralCamel, d.EntityPascal, d.EntityPluralKebab
	routeKey := fmt.Sprintf(`v1.Group("/%s")`, k)
	routeSnippet := fmt.Sprintf(`%[1]s := v1.Group("/%[3]s")
{
	%[1]s.POST("", c.%[2]sHandler.Create)
	%[1]s.GET("", c.%[2]sHandler.List)
	%[1]s.GET("/:id", c.%[2]sHandler.GetByID)
	%[1]s.PUT("/:id", c.%[2]sHandler.Update)
	%[1]s.DELETE("/:id", c.%[2]sHandler.Delete)
}
`, v, p, k)
	if d.Router == RouterChi {
		routeKey = fmt.Sprintf(`v1.Route("/%s"`, k)
		routeSnippet = fmt.Sprintf(`v1.Route("/%[2]s", func(r chi.Router) {
	r.Post("/", c.%[1]sHandler.Create)
	r.Get("/", c.%[1]sHandler.List)
	r.Get("/{id}", c.%[1]sHandler.GetByID)
	r.Put("/{id}", c.%[1]sHandler.Update)
	r.Delete("/{id}", c.%[1]sHandler.Delete)
})
`, p, k)
	}
	rPatches := []patch{{markerRoutes, routeSnippet}}

	ok := true
	for _, step := range []struct {
		path, key, what string
		patches         []patch
	}{
		{container, fmt.Sprintf("handlers.New%sHandler(", d.EntityPascal), "container.go", cPatches},
		{routes, routeKey, "routes.go", rPatches},
	} {
		applied, skipped, err := r.applyPatches(step.path, step.key, step.patches)
		switch {
		case err != nil:
			ui.Warn("Could not patch %s: %v", step.what, err)
			ok = false
		case skipped:
			ui.Warn("%s already registers %s; left unchanged", step.what, d.EntityPascal)
		case applied:
			ui.Success("Registered %s in %s", d.EntityPascal, step.what)
		default:
			ui.Warn("Could not patch %s (missing goclean markers)", step.what)
			ok = false
		}
	}
	return ok
}

func printManualSteps(projectRoot string, d StubData) {
	fmt.Println()
	ui.Info("Next steps to register the module:")
	fmt.Printf("  1. In %s/internal/infrastructure/container/container.go:\n", projectRoot)
	fmt.Printf("     %sRepo := persistence.New%sRepository(db)\n", d.EntityCamel, d.EntityPascal)
	fmt.Printf("     %sUsecase := usecaseimpl.New%sUsecase(%sRepo)\n", d.EntityCamel, d.EntityPascal, d.EntityCamel)
	fmt.Printf("     %sHandler := handlers.New%sHandler(%sUsecase)\n\n", d.EntityCamel, d.EntityPascal, d.EntityCamel)
	fmt.Printf("  2. In %s/internal/delivery/http/routes.go:\n", projectRoot)
	g := d.EntityPluralCamel
	fmt.Printf("     %s := v1.Group(\"/%s\")\n", g, d.EntityPluralKebab)
	fmt.Printf("     %s.POST(\"\", c.%sHandler.Create)\n", g, d.EntityPascal)
	fmt.Printf("     %s.GET(\"\", c.%sHandler.List)\n", g, d.EntityPascal)
	fmt.Printf("     %s.GET(\"/:id\", c.%sHandler.GetByID)\n", g, d.EntityPascal)
	fmt.Printf("     %s.PUT(\"/:id\", c.%sHandler.Update)\n", g, d.EntityPascal)
	fmt.Printf("     %s.DELETE(\"/:id\", c.%sHandler.Delete)\n\n", g, d.EntityPascal)
}

// GenerateModule creates all layers for a Clean Architecture feature and registers it.
func GenerateModule(rootPath, rawName string, opts Options) error {
	data, projectRoot, err := prepare(rootPath, rawName, opts)
	if err != nil {
		return err
	}
	r := &runner{opts: opts}

	ui.Header(fmt.Sprintf("Generating Clean Architecture Module: %s", data.EntityPascal))

	for _, layer := range moduleOrder {
		if err := r.renderLayer(projectRoot, layer, data); err != nil {
			return err
		}
	}
	if err := r.renderMigration(projectRoot, data); err != nil {
		return err
	}

	if opts.NoRegister || !r.register(projectRoot, data) {
		printManualSteps(projectRoot, data)
	}
	r.summary()
	return nil
}

// GenerateSingleLayer generates an isolated layer (entity, types, repo, usecase, handler, or migration).
func GenerateSingleLayer(rootPath, layerType, rawName string, opts Options) error {
	data, projectRoot, err := prepare(rootPath, rawName, opts)
	if err != nil {
		return err
	}
	r := &runner{opts: opts}

	layer := strings.ToLower(layerType)
	switch layer {
	case "type", "dto":
		layer = "types"
	case "repository":
		layer = "repo"
	}
	if layer == "migration" {
		err = r.renderMigration(projectRoot, data)
	} else if _, ok := layerFiles(projectRoot, data)[layer]; ok {
		err = r.renderLayer(projectRoot, layer, data)
	} else {
		return fmt.Errorf("unknown layer type: %s", layerType)
	}
	if err != nil {
		return err
	}
	r.summary()
	return nil
}
