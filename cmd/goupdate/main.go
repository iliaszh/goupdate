package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/iliaszh/goupdate/internal/logerr"
)

func main() {
	var dir string

	slog.Default()

	flag.StringVar(&dir, "project-dir", "", "Project directory.")
	flag.Parse()

	goModFilePath, errGetGoModFilePath := getGoModFilePath(dir)
	if errGetGoModFilePath != nil {
		handleError(errGetGoModFilePath)
		return
	}

	lines, errReadLines := readLines(goModFilePath)
	if errReadLines != nil {
		handleError(errReadLines)
		return
	}

	dependencies, errGetDependencies := getDependencies(lines)
	if errGetDependencies != nil {
		handleError(errGetDependencies)
		return
	}

	slog.Info(
		"Starting updates.",
		slog.Int("number_of_dependencies", len(dependencies)),
	)
	updateStartTime := time.Now()

	for _, dependency := range dependencies {
		updateDependency(dependency)
	}

	errTidy := runGoModTidy()
	if errTidy != nil {
		handleError(errTidy)
		return
	}

	slog.Info(
		"Done.",
		slog.Duration("time_taken", time.Since(updateStartTime).Round(time.Second)),
	)
}

func getGoModFilePath(directory string) (string, error) {
	if directory != "" {
		errChDir := os.Chdir(directory)
		if errChDir != nil {
			return "", logerr.Error{
				Message:     "Failed to change directory.",
				InternalErr: errChDir,
			}
		}
	}

	workDir, errGetWorkDir := os.Getwd()
	if errGetWorkDir != nil {
		return "", logerr.Error{
			Message:     "Failed to get working directory.",
			InternalErr: errGetWorkDir,
		}
	}

	return path.Join(workDir, "go.mod"), nil
}

func readLines(goModFilePath string) ([]string, error) {
	goModFile, errOpen := os.Open(goModFilePath)
	if errOpen != nil {
		return nil, logerr.Error{
			Message:     "Failed to open go.mod file.",
			InternalErr: errOpen,
		}
	}
	defer func() { _ = goModFile.Close() }()

	slog.Info("Found go.mod file.")

	fileBytes, errReadAll := io.ReadAll(goModFile)
	if errReadAll != nil {
		return nil, logerr.Error{
			Message:     "Failed to read go.mod file.",
			InternalErr: errReadAll,
		}
	}

	lines := strings.Split(string(fileBytes), "\n")

	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	return lines, nil
}

func handleError(err error) {
	var logErr logerr.Error
	if errors.As(err, &logErr) {
		slog.Error(logErr.Message, slog.Any("error", logErr.InternalErr))
		return
	}

	slog.Error("Unexpected error!", slog.Any("error", err))
}

func getDependencies(lines []string) ([]string, error) {
	equalTo := func(value string) func(string) bool {
		return func(line string) bool {
			return line == value
		}
	}

	const (
		requireBlockStart = "require ("
		requireBlockEnd   = ")"
	)

	requireBlockStartIdx := slices.IndexFunc(lines, equalTo(requireBlockStart))
	if requireBlockStartIdx < 0 {
		slog.Info("No require block found.")
		return nil, nil
	}

	requireBlockEndIdx := slices.IndexFunc(lines, equalTo(requireBlockEnd))
	if requireBlockEndIdx < 0 {
		return nil, logerr.Error{
			Message:     "Did not find the end of require block.",
			InternalErr: fmt.Errorf("no %q found", requireBlockEnd),
		}
	}

	if requireBlockStartIdx > requireBlockEndIdx {
		return nil, logerr.Error{
			Message:     "Invalid syntax in go.mod file.",
			InternalErr: fmt.Errorf("%q found after %q", requireBlockStart, requireBlockEnd),
		}
	}

	directDependencies := lines[requireBlockStartIdx+1 : requireBlockEndIdx]
	for i, entry := range directDependencies {
		dependency, _, _ := strings.Cut(entry, " ")
		directDependencies[i] = dependency
	}

	return directDependencies, nil
}

func updateDependency(dependency string) {
	start := time.Now()

	cmd := exec.Command("go", "get", "-u", dependency)

	errRun := cmd.Run()
	if errRun != nil {
		slog.Error(
			"Failed to update dependency.",
			slog.String("dependency", dependency),
			slog.Any("error", errRun),
		)
		return
	}

	slog.Info(
		"Update successful.",
		slog.String("dependency", dependency),
		slog.Duration("time_taken", time.Since(start).Round(time.Second)),
	)
}

func runGoModTidy() error {
	slog.Info("Running go mod tidy...")

	errModTidy := exec.Command("go", "mod", "tidy").Run()
	if errModTidy != nil {
		return logerr.Error{
			Message:     "Failed to run go mod tidy.",
			InternalErr: errModTidy,
		}
	}

	return nil
}
