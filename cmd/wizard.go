package cmd

import (
	"os"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/skryfon-org/gostrata/pkg/generator"
	"github.com/spf13/cobra"
)

// interactive reports whether stdin and stdout are both terminals, i.e. whether arrow-key menus can be shown.
func interactive() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

func options(values []string, labels map[string]string) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(values))
	for _, v := range values {
		label := v
		if l, ok := labels[v]; ok {
			label = l
		}
		opts = append(opts, huh.NewOption(label, v))
	}
	return opts
}

// runStackWizard asks for the router, database and ORM. A choice is skipped when its flag was passed
// explicitly or when only one option exists, so scripts and CI keep working with flags alone.
func runStackWizard(cmd *cobra.Command) error {
	var fields []huh.Field

	if !cmd.Flags().Changed("router") && len(generator.SupportedRouters) > 1 {
		fields = append(fields, huh.NewSelect[string]().
			Title("HTTP framework").
			Options(options(generator.SupportedRouters, map[string]string{
				generator.RouterGin: "Gin",
				generator.RouterChi: "Chi",
			})...).
			Value(&flagRouter))
	}
	if !cmd.Flags().Changed("database") && len(generator.SupportedDatabases) > 1 {
		fields = append(fields, huh.NewSelect[string]().
			Title("Database").
			Options(options(generator.SupportedDatabases, map[string]string{
				generator.DBPostgres: "PostgreSQL",
				generator.DBMySQL:    "MySQL",
				generator.DBSQLite:   "SQLite",
			})...).
			Value(&flagDatabase))
	}
	if !cmd.Flags().Changed("orm") && len(generator.SupportedORMs) > 1 {
		fields = append(fields, huh.NewSelect[string]().
			Title("ORM / data access").
			Options(options(generator.SupportedORMs, map[string]string{
				generator.ORMGorm: "GORM",
			})...).
			Value(&flagORM))
	}
	if len(fields) == 0 {
		return nil
	}
	return huh.NewForm(huh.NewGroup(fields...)).Run()
}
