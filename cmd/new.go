package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/skryfon-org/gostrata/pkg/generator"
	"github.com/skryfon-org/gostrata/pkg/strutil"
	"github.com/skryfon-org/gostrata/pkg/ui"
	"github.com/spf13/cobra"
)

var projectNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.-]*$`)

var (
	flagModule   string
	flagDB       string
	flagNoGit    bool
	flagDatabase string
	flagRouter   string
	flagORM      string
	flagNoTidy   bool
)

var newCmd = &cobra.Command{
	Use:   "new [project-name]",
	Short: "Create a fresh Go Clean Architecture project",
	Long:  "Bootstraps a complete, production-ready Go Clean Architecture backend with your choice of router (Gin or Chi) and database (PostgreSQL, MySQL or SQLite), plus GORM, Viper, Zerolog, migrations, and Docker.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Banner(AppVersion)

		reader := bufio.NewReader(os.Stdin)
		projectName := ""
		if len(args) > 0 {
			projectName = strings.TrimSpace(args[0])
		} else {
			fmt.Printf("  %s? Enter project name:%s ", ui.Bold, ui.Reset)
			input, err := reader.ReadString('\n')
			if err != nil && input == "" {
				return fmt.Errorf("failed to read project name: %w", err)
			}
			projectName = strings.TrimSpace(input)
		}

		if projectName == "" {
			return errors.New("project name cannot be empty")
		}
		if !projectNameRe.MatchString(projectName) {
			return fmt.Errorf("invalid project name %q: use letters, digits, '-', '_' or '.', starting with a letter or digit", projectName)
		}

		// Fail on an unusable target directory before asking any stack questions.
		if entries, err := os.ReadDir(filepath.Join(".", strutil.ToKebabCase(projectName))); err == nil && len(entries) > 0 && !flagForce {
			return fmt.Errorf("directory %s already exists and is not empty (use --force to generate into it)", strutil.ToKebabCase(projectName))
		}

		if interactive() && !flagDryRun {
			if err := runStackWizard(cmd); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					fmt.Println("Cancelled.")
					return nil
				}
				return fmt.Errorf("stack selection failed: %w", err)
			}
		}

		flagDatabase = strings.ToLower(flagDatabase)
		flagORM = strings.ToLower(flagORM)
		flagRouter = strings.ToLower(flagRouter)
		if !slices.Contains(generator.SupportedDatabases, flagDatabase) {
			return fmt.Errorf("unsupported --database %q (supported: %s)", flagDatabase, strings.Join(generator.SupportedDatabases, ", "))
		}
		if !slices.Contains(generator.SupportedRouters, flagRouter) {
			return fmt.Errorf("unsupported --router %q (supported: %s)", flagRouter, strings.Join(generator.SupportedRouters, ", "))
		}

		if !slices.Contains(generator.SupportedORMs, flagORM) {
			return fmt.Errorf("unsupported --orm %q (supported: %s)", flagORM, strings.Join(generator.SupportedORMs, ", "))
		}

		cleanProjectName := strutil.ToKebabCase(projectName)
		targetDir := filepath.Join(".", cleanProjectName)

		moduleName := flagModule
		if moduleName == "" {
			defaultModule := cleanProjectName
			fmt.Printf("  %s? Go module path%s [%s]: ", ui.Bold, ui.Reset, defaultModule)
			input, err := reader.ReadString('\n')
			if err != nil && input == "" && err != io.EOF {
				return fmt.Errorf("failed to read module path: %w", err)
			}
			input = strings.TrimSpace(input)
			if input != "" {
				moduleName = input
			} else {
				moduleName = defaultModule
			}
		}

		dbName := flagDB
		if dbName == "" {
			dbName = strutil.ToSnakeCase(cleanProjectName) + "_db"
		}

		ui.Header(fmt.Sprintf("Creating project '%s'...", cleanProjectName))
		ui.Info("Module:   %s", moduleName)
		ui.Info("Stack:    %s + %s + %s", flagRouter, flagDatabase, flagORM)
		ui.Info("Database: %s", dbName)
		ui.Info("Target:   %s", targetDir)
		fmt.Println()

		opts := generator.ProjectOptions{
			ProjectName: cleanProjectName,
			ModuleName:  moduleName,
			DBName:      dbName,
			TargetDir:   targetDir,
			Database:    flagDatabase,
			Router:      flagRouter,
			ORM:         flagORM,
			NoGit:       flagNoGit,
			NoTidy:      flagNoTidy,
			Force:       flagForce,
			DryRun:      flagDryRun,
		}

		if err := generator.GenerateProject(opts); err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		if flagDryRun {
			return nil
		}
		fmt.Println()
		ui.Success("Project '%s' created successfully!", cleanProjectName)
		fmt.Println()
		fmt.Printf("  %sNext steps:%s\n", ui.Bold, ui.Reset)
		fmt.Printf("    1. %scd %s%s\n", ui.Cyan, cleanProjectName, ui.Reset)
		step := 2
		if flagDatabase != generator.DBSQLite {
			fmt.Printf("    %d. %sdocker compose up -d%s        # Start local %s\n", step, ui.Cyan, ui.Reset, flagDatabase)
			step++
		}
		fmt.Printf("    %d. %smake migrate%s                 # Run initial database migrations\n", step, ui.Cyan, ui.Reset)
		fmt.Printf("    %d. %smake dev%s                     # Start server with live hot-reload\n", step+1, ui.Cyan, ui.Reset)
		fmt.Println()
		fmt.Printf("  %sScaffold new features:%s\n", ui.Bold, ui.Reset)
		fmt.Printf("    %sgostrata make:module <Name> [field:type ...]%s  # Generates and registers a full module\n", ui.Cyan, ui.Reset)
		fmt.Println()
		return nil
	},
}

func init() {
	newCmd.Flags().StringVarP(&flagModule, "module", "m", "", "Go module path (e.g. github.com/username/project)")
	newCmd.Flags().StringVar(&flagDatabase, "database", generator.DBPostgres, "database engine: "+strings.Join(generator.SupportedDatabases, ", "))
	newCmd.Flags().StringVar(&flagRouter, "router", generator.RouterGin, "HTTP router: "+strings.Join(generator.SupportedRouters, ", "))
	newCmd.Flags().StringVar(&flagORM, "orm", generator.ORMGorm, "ORM / data access: "+strings.Join(generator.SupportedORMs, ", "))
	newCmd.Flags().BoolVar(&flagNoGit, "no-git", false, "skip git init")
	newCmd.Flags().BoolVar(&flagNoTidy, "no-tidy", false, "skip go mod tidy")
	newCmd.Flags().StringVarP(&flagDB, "db", "d", "", "PostgreSQL database name (default: <project>_db)")
}
