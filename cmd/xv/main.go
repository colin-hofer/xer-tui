package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"xer-tui/internal/update"
	"xer-tui/internal/version"
	"xer-tui/internal/viewer"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "xv: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	switch len(args) {
	case 1:
		switch args[0] {
		case "-h", "--help", "help":
			printUsage()
			return nil
		case "-v", "--version", "version":
			fmt.Println(version.Current())
			return nil
		case "update":
			return runUpdate(context.Background())
		}

		return runViewer(args[0])
	default:
		printUsage()
		if len(args) == 0 {
			os.Exit(2)
		}
		return nil
	}
}

func runViewer(path string) error {
	data, err := viewer.LoadFile(path)
	if err != nil {
		return err
	}

	model := viewer.NewModel(data)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return err
	}

	if m, ok := finalModel.(viewer.Model); ok && m.UpdateRequested {
		return runUpdate(context.Background())
	}
	return nil
}

func runUpdate(ctx context.Context) error {
	updater, err := update.New(update.Config{
		RepoOwner:      version.RepositoryOwner,
		RepoName:       version.RepositoryName,
		BinaryName:     version.BinaryName,
		CurrentVersion: version.Current(),
	})
	if err != nil {
		return err
	}

	fmt.Printf("checking latest release for %s/%s...\n", version.RepositoryOwner, version.RepositoryName)
	result, err := updater.Update(ctx)
	if err != nil {
		return err
	}
	if !result.Updated {
		fmt.Printf("already up to date (%s)\n", result.LatestVersion)
		return nil
	}
	fmt.Printf("updated %s -> %s using %s\n", displayVersion(result.PreviousVersion), result.LatestVersion, result.AssetName)
	fmt.Printf("installed binary: %s\n", result.ExecutablePath)
	if result.RestartRequired {
		fmt.Println("the new binary has been staged and will replace the old one as xv exits")
	}
	fmt.Println("restart xv to use the new version")
	return nil
}

func displayVersion(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: xv <file.xer>")
	fmt.Fprintln(os.Stderr, "       xv update")
	fmt.Fprintln(os.Stderr, "       xv version")
}
