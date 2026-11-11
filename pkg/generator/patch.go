package generator

import (
	"fmt"
	"go/format"
	"os"
	"strings"
)

// Markers that gostrata-created projects carry so modules can be registered automatically.
const (
	markerHandlerFields = "// gostrata:handler-fields"
	markerRepositories  = "// gostrata:repositories"
	markerUsecases      = "// gostrata:usecases"
	markerHandlers      = "// gostrata:handlers"
	markerHandlerInit   = "// gostrata:handler-init"
	markerRoutes        = "// gostrata:routes"
)

// legacyName is the tool's name before v1.3.0; its config file and markers are still read.
const legacyName = "goclean"

type patch struct{ marker, snippet string }

// insertBeforeMarker inserts snippet (indented like the marker) above the marker line.
// It returns the new content and false if the marker is absent.
func insertBeforeMarker(src, marker, snippet string) (string, bool) {
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		if !strings.Contains(line, marker) {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		var ins []string
		for _, l := range strings.Split(strings.TrimRight(snippet, "\n"), "\n") {
			if l == "" {
				ins = append(ins, "")
			} else {
				ins = append(ins, indent+l)
			}
		}
		out := append(append(append([]string{}, lines[:i]...), ins...), lines[i:]...)
		return strings.Join(out, "\n"), true
	}
	return src, false
}

// applyPatches applies every patch to the Go file at path (all or nothing).
// It returns:
//   - applied=false, err=nil when the file is missing a marker (caller falls back to manual steps)
//   - skipped=true when key is already present (module already registered)
func (r *runner) applyPatches(path, key string, patches []patch) (applied, skipped bool, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, false, nil
	}
	src := string(raw)
	if strings.Contains(src, key) {
		return false, true, nil
	}
	for _, p := range patches {
		var ok bool
		src, ok = insertBeforeMarker(src, p.marker, p.snippet)
		if !ok { // projects generated before the rename carry "// goclean:*" markers
			src, ok = insertBeforeMarker(src, strings.Replace(p.marker, "gostrata:", legacyName+":", 1), p.snippet)
		}
		if !ok {
			return false, false, nil
		}
	}
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return false, false, fmt.Errorf("patching %s produced invalid Go: %w", path, err)
	}
	if r.opts.DryRun {
		return true, false, nil
	}
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return false, false, err
	}
	return true, false, nil
}
