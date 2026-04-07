package version

import (
	"runtime/debug"
	"strings"
)

const (
	RepositoryOwner = "mewtyunjay"
	RepositoryName  = "xer-tui"
	BinaryName      = "xv"
)

var Version = "dev"

func Current() string {
	if value := strings.TrimSpace(Version); value != "" && value != "dev" {
		return value
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	value := strings.TrimSpace(info.Main.Version)
	if value == "" || value == "(devel)" {
		return "dev"
	}
	return value
}
