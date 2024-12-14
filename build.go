//go:build ignore
// +build ignore

// Build "script" for the stropt project release package generation
// Use by executing "go run build.go"

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	version := ExecCommand("git", "describe", "--tags")
	if version[0] == 'v' {
		version = strings.TrimSpace(version[1:])
	}

	archs := [...]string{"386", "amd64", "arm", "arm64"}
	oss := [...]string{"linux", "darwin", "windows"}

	os.Mkdir("dist", 0755)

	defer func() {
		_ = os.Remove("stropt")
		_ = os.Remove("stropt.exe")
	}()

	for _, osName := range oss {
		for _, arch := range archs {
			if osName == "darwin" && (arch == "386" || arch == "arm") {
				continue
			}
			ExecBuild(arch, osName)
			arName := fmt.Sprintf("stropt_%s_%s_%s", version, osName, arch)
			if osName == "windows" {
				arName += ".zip"
				ExecCommand("zip", arName, "README.md", "LICENSE", "stropt.exe")
			} else {
				arName += ".tar.gz"
				ExecCommand("tar", "-czvf", arName, "README.md", "LICENSE", "stropt")
			}

			if err := os.Rename(arName, "dist/"+arName); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
				return
			}
		}
	}
}

func ExecCommand(c string, args ...string) string {
	cmd := exec.Command(c, args...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	return buf.String()
}

func ExecBuild(arch, osName string) {
	cmd := exec.Command("go", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "GOARCH="+arch, "GOOS="+osName)

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
