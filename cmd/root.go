package cmd

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/skryfon-org/goclean/pkg/ui"
	"github.com/spf13/cobra"
)

// AppVersion is overridden at build time: -ldflags "-X github.com/skryfon-org/goclean/cmd.AppVersion=x.y.z".
// When unset it falls back to the module version embedded by `go install`.
var AppVersion = "dev"

// Flags shared by the generators.
var (
	flagForce      bool
	flagDryRun     bool
	flagNoRegister bool
)

func init() {
	if AppVersion != "dev" {
		return
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		AppVersion = strings.TrimPrefix(bi.Main.Version, "v")
	}
}

var rootCmd = &cobra.Command{
	Use:   "goclean",
	Short: "goclean — Modern Go Clean Architecture generator & CLI tool",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = AppVersion
	rootCmd.PersistentFlags().BoolVar(&flagForce, "force", false, "overwrite existing files")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "show what would be generated without writing anything")
	rootCmd.Long = fmt.Sprintf(`%s%s  ____      ____ _                  
 / ___|___ / ___| | ___  __ _ _ __  
| |  _/ _ \ |   | |/ _ \/ _`+"`"+` | '_ \ 
| |_| | (_) | |___| |  __/ (_| | | | |
 \____|\___/ \____|_|\___|\__,_|_| |_|  v%s%s

Go Clean Architecture Scaffolding & Code Generation Tool.
Quickly bootstrap production-ready Go projects and scaffold Clean Architecture modules.`, ui.Bold, ui.Cyan, AppVersion, ui.Reset)
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(makeModuleCmd)
	rootCmd.AddCommand(makeEntityCmd)
	rootCmd.AddCommand(makeTypesCmd)
	rootCmd.AddCommand(makeRepoCmd)
	rootCmd.AddCommand(makeUsecaseCmd)
	rootCmd.AddCommand(makeHandlerCmd)
	rootCmd.AddCommand(makeMigrationCmd)
	rootCmd.AddCommand(doctorCmd)
}
