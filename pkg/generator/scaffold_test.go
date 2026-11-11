package generator

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func newProject(t *testing.T, tidy bool) string {
	t.Helper()
	return newStackProject(t, tidy, "", "")
}

func newStackProject(t *testing.T, tidy bool, db, router string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "demo")
	err := GenerateProject(ProjectOptions{ProjectName: "demo", ModuleName: "example.com/demo", DBName: "demo_db", TargetDir: dir, Database: db, Router: router, NoGit: true, NoTidy: !tidy})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func mustFields(t *testing.T, defs ...string) []Field {
	t.Helper()
	f, err := ParseFields(defs)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParseFields(t *testing.T) {
	f, err := ParseFields([]string{"name:string", "unit_price:float", "Stock:int", "note"})
	if err != nil || len(f) != 4 {
		t.Fatalf("got %v %v", f, err)
	}
	if f[1].Pascal != "UnitPrice" || f[1].Snake != "unit_price" || f[1].GoType != "float64" || f[3].Type != "string" {
		t.Errorf("unexpected fields: %+v", f)
	}
	for _, bad := range []string{"name:blob", "id:int", ":int", "a:int,a:int"} {
		if _, err := ParseFields([]string{bad}); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
	if _, err := ParseFields([]string{"a:int", "a:string"}); err == nil {
		t.Error("expected duplicate error")
	}
}

func TestModuleWithFieldsAutoRegisters(t *testing.T) {
	dir := newProject(t, false)
	opts := Options{Fields: mustFields(t, "name:string", "price:float", "active:bool", "launched:time")}
	if err := GenerateModule(dir, "Product", opts); err != nil {
		t.Fatal(err)
	}
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Join(dir, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	container := read("internal/infrastructure/container/container.go")
	routes := read("internal/delivery/http/routes.go")
	for _, want := range []string{"ProductHandler *handlers.ProductHandler", "productRepo := persistence.NewProductRepository(db)", "ProductHandler: productHandler"} {
		if !strings.Contains(container, want) {
			t.Errorf("container.go missing %q", want)
		}
	}
	if !strings.Contains(routes, `v1.Group("/products")`) || !strings.Contains(routes, "c.ProductHandler.Update") {
		t.Error("routes.go not patched")
	}
	if !strings.Contains(read("migrations/000002_create_products_table.up.sql"), "price DOUBLE PRECISION NOT NULL") {
		t.Error("migration missing price column")
	}
	if !strings.Contains(read("internal/domain/entity/product.go"), "Price") || !strings.Contains(read("internal/types/product.go"), `binding:"required"`) {
		t.Error("entity/types missing fields")
	}

	// Idempotent: a second run without --force changes nothing and does not double-register.
	before := container
	if err := GenerateModule(dir, "Product", opts); err != nil {
		t.Fatal(err)
	}
	if read("internal/infrastructure/container/container.go") != before {
		t.Error("container.go changed on second run")
	}
	assertGoParses(t, dir)
}

func TestNoRegisterAndDryRun(t *testing.T) {
	dir := newProject(t, false)
	if err := GenerateModule(dir, "Order", Options{DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "internal/domain/entity/order.go")); err == nil {
		t.Error("dry run wrote a file")
	}
	if err := GenerateModule(dir, "Order", Options{NoRegister: true}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "internal/delivery/http/routes.go"))
	if strings.Contains(string(b), "/orders") {
		t.Error("--no-register still patched routes.go")
	}
}

func TestForceOverwrites(t *testing.T) {
	dir := newProject(t, false)
	p := filepath.Join(dir, "internal/domain/entity/item.go")
	if err := GenerateSingleLayer(dir, "entity", "Item", Options{}); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(p, []byte("custom"), 0644)
	_ = GenerateSingleLayer(dir, "entity", "Item", Options{})
	if b, _ := os.ReadFile(p); string(b) != "custom" {
		t.Error("existing file was overwritten without --force")
	}
	_ = GenerateSingleLayer(dir, "entity", "Item", Options{Force: true})
	if b, _ := os.ReadFile(p); string(b) == "custom" {
		t.Error("--force did not overwrite")
	}
}

func TestMissingMarkersFallsBack(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/a\n\ngo 1.22\n"), 0644)
	rt := filepath.Join(dir, "internal/delivery/http")
	_ = os.MkdirAll(rt, 0755)
	_ = os.WriteFile(filepath.Join(rt, "routes.go"), []byte("package http\n"), 0644)
	if err := GenerateModule(dir, "Thing", Options{}); err != nil {
		t.Fatal(err) // must not fail; just prints manual steps
	}
}

func TestDryRunProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "x")
	if err := GenerateProject(ProjectOptions{ProjectName: "x", TargetDir: dir, DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Error("dry run created the project directory")
	}
}

// assertGoParses checks every generated .go file is syntactically valid (no network needed).
func assertGoParses(t *testing.T, root string) {
	t.Helper()
	fset := token.NewFileSet()
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".go") {
			if _, perr := parser.ParseFile(fset, p, nil, parser.AllErrors); perr != nil {
				t.Errorf("%s: %v", p, perr)
			}
		}
		return nil
	})
}

// TestSmokeBuild scaffolds a project, adds a module with fields and compiles everything.
// It needs network access (go mod tidy), so it only runs when GOSTRATA_SMOKE=1 (set in CI).
func TestSmokeBuild(t *testing.T) {
	if os.Getenv("GOSTRATA_SMOKE") != "1" {
		t.Skip("set GOSTRATA_SMOKE=1 to run the scaffold-and-build smoke test")
	}
	for _, db := range SupportedDatabases {
		for _, router := range SupportedRouters {
			t.Run(db+"-"+router, func(t *testing.T) { smokeBuild(t, db, router) })
		}
	}
}

func smokeBuild(t *testing.T, db, router string) {
	dir := newStackProject(t, true, db, router)
	if err := GenerateModule(dir, "Product", Options{Fields: mustFields(t, "name:string", "price:float", "stock:int", "sku:uuid", "launched:time", "active:bool")}); err != nil {
		t.Fatal(err)
	}
	if err := GenerateModule(dir, "OrderItem", Options{}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"build", "./..."}, {"vet", "./..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}

func TestStackMatrixGeneratesValidGo(t *testing.T) {
	for _, db := range SupportedDatabases {
		for _, router := range SupportedRouters {
			t.Run(db+"-"+router, func(t *testing.T) {
				dir := newStackProject(t, false, db, router)
				if err := GenerateModule(dir, "Product", Options{Fields: mustFields(t, "name", "price:float", "ref:uuid", "at:time")}); err != nil {
					t.Fatal(err)
				}
				assertGoParses(t, dir)

				read := func(p string) string {
					b, err := os.ReadFile(filepath.Join(dir, p))
					if err != nil {
						t.Fatalf("%s: %v", p, err)
					}
					return string(b)
				}
				if got := read(".gostrata.json"); !strings.Contains(got, db) || !strings.Contains(got, router) {
					t.Errorf(".gostrata.json = %s", got)
				}
				_, composeErr := os.Stat(filepath.Join(dir, "docker-compose.yml"))
				if (db == DBSQLite) != os.IsNotExist(composeErr) {
					t.Errorf("docker-compose.yml presence wrong for %s (err=%v)", db, composeErr)
				}
				for _, f := range []string{"Dockerfile", ".dockerignore"} {
					if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
						t.Errorf("%s not generated for %s: %v", f, db, err)
					}
				}
				df := read("Dockerfile")
				if (db == DBSQLite) != strings.Contains(df, "/data") {
					t.Errorf("Dockerfile /data volume wrong for %s", db)
				}
				routes := read("internal/delivery/http/routes.go")
				if router == RouterChi && !strings.Contains(routes, `v1.Route("/products"`) {
					t.Error("chi routes not registered")
				}
				if router == RouterChi && strings.Contains(routes, `("",`) {
					t.Error(`chi panics on empty route patterns; use "/"`)
				}
				if router == RouterGin && !strings.Contains(routes, `v1.Group("/products")`) {
					t.Error("gin routes not registered")
				}
				if !strings.Contains(read("go.mod"), map[string]string{RouterChi: "go-chi/chi", RouterGin: "gin-gonic/gin"}[router]) {
					t.Error("go.mod missing router dependency")
				}
				mig := read("migrations/000002_create_products_table.up.sql")
				wantUUID := map[string]string{DBPostgres: "ref UUID", DBMySQL: "ref CHAR(36)", DBSQLite: "ref TEXT"}[db]
				if !strings.Contains(mig, wantUUID) {
					t.Errorf("migration for %s missing %q:\n%s", db, wantUUID, mig)
				}
				if strings.Contains(read("internal/domain/entity/product.go"), "gen_random_uuid") != (db == DBPostgres) {
					t.Error("gen_random_uuid default should only appear for postgres")
				}
				// A router/db that wasn't chosen must not leak into the project.
				if router == RouterChi && strings.Contains(read("internal/delivery/http/response/response.go"), "gin-gonic") {
					t.Error("gin leaked into chi project")
				}
			})
		}
	}
}

func TestInvalidStackRejected(t *testing.T) {
	for _, o := range []ProjectOptions{{Database: "oracle"}, {Router: "echo"}} {
		o.ProjectName, o.TargetDir = "x", filepath.Join(t.TempDir(), "x")
		if err := GenerateProject(o); err == nil {
			t.Errorf("expected error for %+v", o)
		}
	}
}
