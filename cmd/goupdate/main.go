package main

import (
	"flag"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
)

func main() {
	var dir string

	flag.StringVar(&dir, "project-dir", "", "Project directory")
	flag.Parse()

	if dir != "" {
		errChDir := os.Chdir(dir)
		if errChDir != nil {
			slog.Error("Failed to change directory.", slog.Any("error", errChDir))
			return
		}
	}

	workDir, errGetWorkDir := os.Getwd()
	if errGetWorkDir != nil {
		slog.Error("Failed to get working directory.", slog.Any("error", errGetWorkDir))
		return
	}

	modFileAddr := path.Join(workDir, "go.mod")
	modFile, errOpen := os.Open(modFileAddr)
	if errOpen != nil {
		slog.Error("Failed to open go.mod file.", slog.Any("error", errOpen))
		return
	}

	defer func() { _ = modFile.Close() }()

	fileBytes, errReadAll := io.ReadAll(modFile)
	if errReadAll != nil {
		slog.Error("Failed to read go.mod file.", slog.Any("error", errReadAll))
		return
	}

	lines := strings.Split(string(fileBytes), "\n")

	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	requireBlockStartIdx := slices.IndexFunc(lines, func(line string) bool {
		return line == "require ("
	})
	if requireBlockStartIdx < 0 {
		slog.Info("No require block found.")
		return
	}

	requireBlockEndIdx := slices.IndexFunc(lines, func(line string) bool {
		return line == ")"
	})
	if requireBlockEndIdx < 0 {
		slog.Error("Did not find the end of require block.")
		return
	}

	directDependencies := lines[requireBlockStartIdx+1 : requireBlockEndIdx]
	for i, entry := range directDependencies {
		dependency, _, _ := strings.Cut(entry, " ")
		directDependencies[i] = dependency
	}

	for _, dependency := range directDependencies {
		cmd := exec.Command("go", "get", "-u", dependency)

		slog.Info("Updating dependency...", slog.String("dependency", dependency))

		errRun := cmd.Run()
		if errRun != nil {
			slog.Error(
				"Failed to update dependency.",
				slog.String("dependency", dependency),
				slog.Any("error", errRun),
			)
			break
		}
	}

	slog.Info("Running go mod tidy...")
	errModTidy := exec.Command("go", "mod", "tidy").Run()
	if errModTidy != nil {
		slog.Error("Failed to run go mod tidy.", slog.Any("error", errModTidy))
	}
}
