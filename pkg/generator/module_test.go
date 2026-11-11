package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildStubData(t *testing.T) {
	data := BuildStubData("github.com/example/app", "product_category")

	if data.ModuleName != "github.com/example/app" {
		t.Errorf("expected module name github.com/example/app, got %s", data.ModuleName)
	}
	if data.EntityPascal != "ProductCategory" {
		t.Errorf("expected ProductCategory, got %s", data.EntityPascal)
	}
	if data.EntityCamel != "productCategory" {
		t.Errorf("expected productCategory, got %s", data.EntityCamel)
	}
	if data.EntitySnake != "product_category" {
		t.Errorf("expected product_category, got %s", data.EntitySnake)
	}
	if data.EntityPluralSnake != "product_categories" {
		t.Errorf("expected product_categories, got %s", data.EntityPluralSnake)
	}
}

func TestGenerateModuleWithTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a dummy go.mod
	goModContent := "module github.com/example/testapp\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	if err := GenerateModule(tmpDir, "Product", Options{}); err != nil {
		t.Fatalf("GenerateModule failed: %v", err)
	}

	expectedFiles := []string{
		"internal/domain/entity/product.go",
		"internal/types/product.go",
		"internal/domain/repository/product_repository.go",
		"internal/infrastructure/persistence/product_repository.go",
		"internal/usecase/interface/product_usecase.go",
		"internal/usecase/implementation/product_usecase.go",
		"internal/delivery/http/handlers/product_handler.go",
	}

	for _, relPath := range expectedFiles {
		fullPath := filepath.Join(tmpDir, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", relPath)
		}
	}
}

func TestGenerateSingleLayerTypes(t *testing.T) {
	tmpDir := t.TempDir()

	goModContent := "module github.com/example/testapp\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	if err := GenerateSingleLayer(tmpDir, "types", "Order", Options{}); err != nil {
		t.Fatalf("GenerateSingleLayer types failed: %v", err)
	}

	orderTypesPath := filepath.Join(tmpDir, "internal", "types", "order.go")
	if _, err := os.Stat(orderTypesPath); os.IsNotExist(err) {
		t.Errorf("expected %s to exist", orderTypesPath)
	}
}

func TestInvalidNameRejected(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/a\n\ngo 1.22\n"), 0644)
	for _, name := range []string{"", "---", "1abc", "a$b"} {
		if err := GenerateSingleLayer(tmpDir, "entity", name, Options{}); err == nil {
			t.Errorf("expected error for name %q", name)
		}
	}
}

func TestUnknownLayerAndMissingGoMod(t *testing.T) {
	tmpDir := t.TempDir()
	if err := GenerateSingleLayer(tmpDir, "entity", "Foo", Options{}); err == nil {
		t.Error("expected error without go.mod")
	}
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/a\n\ngo 1.22\n"), 0644)
	if err := GenerateSingleLayer(tmpDir, "bogus", "Foo", Options{}); err == nil {
		t.Error("expected error for unknown layer")
	}
}

func TestMigrationAndSoftDeleteEntity(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/a\n\ngo 1.22\n"), 0644)
	if err := GenerateModule(tmpDir, "Person", Options{}); err != nil {
		t.Fatal(err)
	}
	ups, _ := filepath.Glob(filepath.Join(tmpDir, "migrations", "*_create_people_table.up.sql"))
	downs, _ := filepath.Glob(filepath.Join(tmpDir, "migrations", "*_create_people_table.down.sql"))
	if len(ups) != 1 || len(downs) != 1 {
		t.Fatalf("expected up/down migrations, got %v %v", ups, downs)
	}
	b, _ := os.ReadFile(filepath.Join(tmpDir, "internal/domain/entity/person.go"))
	if !strings.Contains(string(b), "gorm.DeletedAt") {
		t.Error("entity should use gorm.DeletedAt for soft delete")
	}
}

func TestMigrationsAreSequential(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/a\n\ngo 1.22\n"), 0644)
	mig := filepath.Join(tmpDir, "migrations")
	_ = os.MkdirAll(mig, 0755)
	_ = os.WriteFile(filepath.Join(mig, "000001_create_users_table.up.sql"), nil, 0644)
	_ = os.WriteFile(filepath.Join(mig, "000001_create_users_table.down.sql"), nil, 0644)
	if err := GenerateSingleLayer(tmpDir, "migration", "Order", Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(mig, "000002_create_orders_table.up.sql")); err != nil {
		t.Errorf("expected 000002 migration: %v", err)
	}
}

// Projects generated before the rename keep working: old config file and old markers.
func TestLegacyGocleanProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".goclean.json"), []byte(`{"database":"sqlite","router":"chi","orm":"gorm"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if cfg := loadProjectConfig(dir); cfg.Database != DBSQLite || cfg.Router != RouterChi {
		t.Fatalf("legacy config not read: %+v", cfg)
	}

	file := filepath.Join(dir, "routes.go")
	if err := os.WriteFile(file, []byte("package x\n\nfunc f() {\n\t// goclean:routes\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	r := &runner{}
	applied, _, err := r.applyPatches(file, "g()", []patch{{markerRoutes, "g()"}})
	if err != nil || !applied {
		t.Fatalf("legacy marker not patched: applied=%v err=%v", applied, err)
	}
	if got, _ := os.ReadFile(file); !strings.Contains(string(got), "g()") {
		t.Fatalf("snippet missing:\n%s", got)
	}
}
