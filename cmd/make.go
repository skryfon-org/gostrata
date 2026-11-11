package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/skryfon-org/goclean/pkg/generator"
	"github.com/spf13/cobra"
)

var makeModuleCmd = &cobra.Command{
	Use:     "make:module <Name> [field:type ...]",
	Aliases: []string{"make:resource"},
	Short:   "Generate an entire Clean Architecture module (Entity, Repo, Usecase, Handler, Migration)",
	Args:    cobra.MinimumNArgs(1),
	Long:    fieldHelp("Generate every layer of a feature (entity, DTOs, repository, usecase, handler, migration)\nand register it in container.go and routes.go."),
	Example: "  goclean make:module Product name:string price:float stock:int active:bool",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to determine current directory: %w", err)
		}
		opts, err := buildOptions(args[1:])
		if err != nil {
			return err
		}
		return generator.GenerateModule(cwd, args[0], opts)
	},
}

var makeEntityCmd = &cobra.Command{
	Use:   "make:entity <Name>",
	Short: "Generate a domain entity",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSingleLayerGenerator("entity", args)
	},
}

var makeTypesCmd = &cobra.Command{
	Use:     "make:types <Name>",
	Aliases: []string{"make:type", "make:dto"},
	Short:   "Generate request, input, and response DTO types",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSingleLayerGenerator("types", args)
	},
}

var makeRepoCmd = &cobra.Command{
	Use:     "make:repo <Name>",
	Aliases: []string{"make:repository"},
	Short:   "Generate repository interface and GORM persistence implementation",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSingleLayerGenerator("repo", args)
	},
}

var makeUsecaseCmd = &cobra.Command{
	Use:   "make:usecase <Name>",
	Short: "Generate usecase interface and business logic implementation",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSingleLayerGenerator("usecase", args)
	},
}

var makeHandlerCmd = &cobra.Command{
	Use:   "make:handler <Name>",
	Short: "Generate an HTTP Gin handler",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSingleLayerGenerator("handler", args)
	},
}

var makeMigrationCmd = &cobra.Command{
	Use:   "make:migration <Name>",
	Short: "Generate up and down database migration files",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSingleLayerGenerator("migration", args)
	},
}

func runSingleLayerGenerator(layerType string, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}
	opts, err := buildOptions(args[1:])
	if err != nil {
		return err
	}
	return generator.GenerateSingleLayer(cwd, layerType, args[0], opts)
}

func buildOptions(fieldDefs []string) (generator.Options, error) {
	fields, err := generator.ParseFields(fieldDefs)
	if err != nil {
		return generator.Options{}, err
	}
	return generator.Options{Fields: fields, Force: flagForce, DryRun: flagDryRun, NoRegister: flagNoRegister}, nil
}

func fieldHelp(intro string) string {
	return intro + "\n\nFields are name:type pairs (type defaults to string). Types: " + strings.Join(generator.SupportedFieldTypes(), ", ") + "."
}

func init() {
	makeModuleCmd.Flags().BoolVar(&flagNoRegister, "no-register", false, "don't patch container.go and routes.go")
}
