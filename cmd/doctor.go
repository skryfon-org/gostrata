package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/skryfon-org/goclean/pkg/ui"
	"github.com/spf13/cobra"
)

var goVersionRe = regexp.MustCompile(`go(\d+)\.(\d+)`)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check that Go, git and docker are installed and recent enough",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Header("goclean doctor")
		failed := false

		// Go (required, >= 1.22; 1.23+ recommended since goclean itself needs 1.23)
		if out, err := run("go", "version"); err != nil {
			ui.Error("go: not found in PATH (required)")
			failed = true
		} else if m := goVersionRe.FindStringSubmatch(out); m == nil {
			ui.Warn("go: could not parse version: %s", out)
		} else {
			major, _ := strconv.Atoi(m[1])
			minor, _ := strconv.Atoi(m[2])
			if major > 1 || minor >= 23 {
				ui.Success("go: %s", out)
			} else if minor == 22 {
				ui.Warn("go: %s (works, but goclean and generated projects need Go 1.23 and Go will auto-download that toolchain)", out)
			} else {
				ui.Error("go: %s (Go 1.22+ required, 1.23+ recommended)", out)
				failed = true
			}
		}

		// git (needed for `goclean new` repo init)
		if out, err := run("git", "--version"); err != nil {
			ui.Warn("git: not found (new projects won't get a repository; use --no-git to silence)")
		} else {
			ui.Success("git: %s", out)
		}

		// docker (needed for the local PostgreSQL in docker-compose)
		if out, err := run("docker", "--version"); err != nil {
			ui.Warn("docker: not found (needed for `docker compose up -d`)")
		} else {
			ui.Success("docker: %s", out)
			if cout, err := run("docker", "compose", "version"); err != nil {
				ui.Warn("docker compose: plugin not available")
			} else {
				ui.Success("docker compose: %s", cout)
			}
		}

		if failed {
			return errors.New("required tools are missing")
		}
		fmt.Println()
		ui.Success("All required tools are available")
		return nil
	},
}

func run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
